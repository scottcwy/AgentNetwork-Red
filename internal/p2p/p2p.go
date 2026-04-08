package p2p

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"agentnetwork-red/internal/config"
	"agentnetwork-red/internal/identity"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	dmTopicID    = "/anet/dm"
	tasksTopicID = "/anet/tasks"
	dmStreamID   = protocol.ID("/anet/dm/1.0.0")
	tipStreamID  = protocol.ID("/anet/tip/1.0.0")
)

type Node struct {
	Host   host.Host
	PubSub *pubsub.PubSub
	DHT    *dht.IpfsDHT

	cancel        context.CancelFunc
	closeOnce     sync.Once
	topics        map[string]*pubsub.Topic
	subscriptions []*pubsub.Subscription
	dmHandler     func(context.Context, []byte) error
	taskHandler   func(context.Context, []byte) error
	tipHandler    func(context.Context, TipRequest) (TipResponse, error)
}

type TipRequest struct {
	DID         string `json:"did"`
	IncludeData bool   `json:"include_data,omitempty"`
}

type TipResponse struct {
	Found     bool   `json:"found"`
	Filename  string `json:"filename,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Data      string `json:"data,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type PeerInfo struct {
	PeerID string   `json:"peer_id"`
	DID    string   `json:"did,omitempty"`
	Addrs  []string `json:"addrs"`
}

func New(parent context.Context, cfg config.Config, ident *identity.Identity) (*Node, error) {
	ctx, cancel := context.WithCancel(parent)

	privKey, err := libp2pPrivateKey(ident.PrivateKey)
	if err != nil {
		cancel()
		return nil, err
	}

	hostNode, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(cfg.Listen...),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		cancel()
		return nil, err
	}

	bootstrapPeers, err := bootstrapPeersFromConfig(cfg.BootstrapPeers)
	if err != nil {
		cancel()
		_ = hostNode.Close()
		return nil, err
	}

	dhtOptions := []dht.Option{dht.Mode(dht.ModeAutoServer)}
	if len(bootstrapPeers) > 0 {
		dhtOptions = append(dhtOptions, dht.BootstrapPeers(bootstrapPeers...))
	}

	kad, err := dht.New(ctx, hostNode, dhtOptions...)
	if err != nil {
		cancel()
		_ = hostNode.Close()
		return nil, err
	}

	if err := kad.Bootstrap(ctx); err != nil {
		cancel()
		_ = kad.Close()
		_ = hostNode.Close()
		return nil, err
	}

	ps, err := pubsub.NewGossipSub(ctx, hostNode)
	if err != nil {
		cancel()
		_ = kad.Close()
		_ = hostNode.Close()
		return nil, err
	}

	node := &Node{
		Host:          hostNode,
		PubSub:        ps,
		DHT:           kad,
		cancel:        cancel,
		topics:        make(map[string]*pubsub.Topic, 2),
		subscriptions: make([]*pubsub.Subscription, 0, 2),
	}

	node.Host.SetStreamHandler(dmStreamID, node.handleDMStream)
	node.Host.SetStreamHandler(tipStreamID, node.handleTipStream)

	if err := node.subscribeTopic(ctx, dmTopicID); err != nil {
		_ = node.Close()
		return nil, err
	}
	if err := node.subscribeTopic(ctx, tasksTopicID); err != nil {
		_ = node.Close()
		return nil, err
	}

	node.connectBootstrapPeers(ctx, bootstrapPeers)
	return node, nil
}

func (n *Node) Close() error {
	var closeErr error
	n.closeOnce.Do(func() {
		if n.cancel != nil {
			n.cancel()
		}
		for _, sub := range n.subscriptions {
			sub.Cancel()
		}
		for _, topic := range n.topics {
			if err := topic.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
		if n.DHT != nil {
			if err := n.DHT.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
		if n.Host != nil {
			if err := n.Host.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
	})
	return closeErr
}

func (n *Node) PeerID() string {
	if n == nil || n.Host == nil {
		return ""
	}
	return n.Host.ID().String()
}

func (n *Node) ConnectedPeers() []PeerInfo {
	if n == nil || n.Host == nil {
		return nil
	}

	peerIDs := n.Host.Network().Peers()
	peers := make([]PeerInfo, 0, len(peerIDs))
	for _, peerID := range peerIDs {
		addrs := n.Host.Peerstore().Addrs(peerID)
		peerInfo := PeerInfo{
			PeerID: peerID.String(),
			Addrs:  make([]string, 0, len(addrs)),
		}
		for _, addr := range addrs {
			peerInfo.Addrs = append(peerInfo.Addrs, addr.String())
		}
		peers = append(peers, peerInfo)
	}
	return peers
}

func (n *Node) ListenAddrs() []string {
	if n == nil || n.Host == nil {
		return nil
	}

	addrs := n.Host.Addrs()
	values := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		values = append(values, fmt.Sprintf("%s/p2p/%s", addr.String(), n.Host.ID().String()))
	}
	return values
}

func (n *Node) Connect(ctx context.Context, addr string) (*peer.AddrInfo, error) {
	info, err := peer.AddrInfoFromString(addr)
	if err != nil {
		return nil, err
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := n.Host.Connect(connectCtx, *info); err != nil {
		return nil, err
	}
	return info, nil
}

func (n *Node) SetDMHandler(handler func(context.Context, []byte) error) {
	n.dmHandler = handler
}

func (n *Node) SetTaskHandler(handler func(context.Context, []byte) error) {
	n.taskHandler = handler
}

func (n *Node) SetTipHandler(handler func(context.Context, TipRequest) (TipResponse, error)) {
	n.tipHandler = handler
}

func (n *Node) PublishDM(ctx context.Context, payload []byte) error {
	topic, ok := n.topics[dmTopicID]
	if !ok {
		return fmt.Errorf("dm topic unavailable")
	}
	return topic.Publish(ctx, payload)
}

func (n *Node) PublishTask(ctx context.Context, payload []byte) error {
	topic, ok := n.topics[tasksTopicID]
	if !ok {
		return fmt.Errorf("tasks topic unavailable")
	}
	return topic.Publish(ctx, payload)
}

func (n *Node) SendDMStream(ctx context.Context, peerID peer.ID, payload []byte) error {
	streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stream, err := n.Host.NewStream(streamCtx, peerID, dmStreamID)
	if err != nil {
		return err
	}
	defer stream.Close()

	if err := json.NewEncoder(stream).Encode(json.RawMessage(payload)); err != nil {
		return err
	}
	if err := stream.CloseWrite(); err != nil {
		return err
	}

	var ack struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(stream).Decode(&ack); err != nil {
		return err
	}
	if !ack.OK {
		if ack.Error == "" {
			ack.Error = "remote dm handler rejected payload"
		}
		return errors.New(ack.Error)
	}
	return nil
}

func (n *Node) RequestTip(ctx context.Context, peerID peer.ID, req TipRequest) (TipResponse, error) {
	streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stream, err := n.Host.NewStream(streamCtx, peerID, tipStreamID)
	if err != nil {
		return TipResponse{}, err
	}
	defer stream.Close()

	if err := json.NewEncoder(stream).Encode(req); err != nil {
		return TipResponse{}, err
	}
	if err := stream.CloseWrite(); err != nil {
		return TipResponse{}, err
	}

	var resp struct {
		OK    bool        `json:"ok"`
		Error string      `json:"error,omitempty"`
		Tip   TipResponse `json:"tip"`
	}
	if err := json.NewDecoder(io.LimitReader(stream, 4<<20)).Decode(&resp); err != nil {
		return TipResponse{}, err
	}
	if !resp.OK {
		if resp.Error == "" {
			resp.Error = "remote tip handler rejected request"
		}
		return TipResponse{}, errors.New(resp.Error)
	}
	return resp.Tip, nil
}

func (n *Node) subscribeTopic(ctx context.Context, topicName string) error {
	topic, err := n.PubSub.Join(topicName)
	if err != nil {
		return err
	}

	subscription, err := topic.Subscribe()
	if err != nil {
		_ = topic.Close()
		return err
	}

	n.topics[topicName] = topic
	n.subscriptions = append(n.subscriptions, subscription)

	go func() {
		for {
			msg, err := subscription.Next(ctx)
			if err != nil {
				return
			}
			if msg.ReceivedFrom == n.Host.ID() {
				continue
			}
			if topicName == dmTopicID && n.dmHandler != nil {
				go func(data []byte) {
					if err := n.dmHandler(context.Background(), data); err != nil {
						log.Printf("dm gossip handler error: %v", err)
					}
				}(append([]byte(nil), msg.Data...))
			}
			if topicName == tasksTopicID && n.taskHandler != nil {
				go func(data []byte) {
					if err := n.taskHandler(context.Background(), data); err != nil {
						log.Printf("task gossip handler error: %v", err)
					}
				}(append([]byte(nil), msg.Data...))
			}
		}
	}()

	return nil
}

func (n *Node) handleDMStream(stream network.Stream) {
	defer stream.Close()

	var payload json.RawMessage
	if err := json.NewDecoder(io.LimitReader(stream, 1<<20)).Decode(&payload); err != nil {
		_ = json.NewEncoder(stream).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	if n.dmHandler == nil {
		_ = json.NewEncoder(stream).Encode(map[string]any{
			"ok":    false,
			"error": "dm handler unavailable",
		})
		return
	}

	if err := n.dmHandler(context.Background(), []byte(payload)); err != nil {
		_ = json.NewEncoder(stream).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(stream).Encode(map[string]any{"ok": true})
}

func (n *Node) handleTipStream(stream network.Stream) {
	defer stream.Close()

	var req TipRequest
	if err := json.NewDecoder(io.LimitReader(stream, 1<<20)).Decode(&req); err != nil {
		_ = json.NewEncoder(stream).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	if req.DID == "" {
		_ = json.NewEncoder(stream).Encode(map[string]any{
			"ok":    false,
			"error": "missing did",
		})
		return
	}
	if n.tipHandler == nil {
		_ = json.NewEncoder(stream).Encode(map[string]any{
			"ok":    false,
			"error": "tip handler unavailable",
		})
		return
	}

	resp, err := n.tipHandler(context.Background(), req)
	if err != nil {
		_ = json.NewEncoder(stream).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	if !req.IncludeData {
		resp.Data = ""
	} else if resp.Data != "" {
		// Keep binary payload ASCII-safe on the wire.
		if _, err := base64.StdEncoding.DecodeString(resp.Data); err != nil {
			_ = json.NewEncoder(stream).Encode(map[string]any{
				"ok":    false,
				"error": "invalid tip payload encoding",
			})
			return
		}
	}

	_ = json.NewEncoder(stream).Encode(map[string]any{
		"ok":  true,
		"tip": resp,
	})
}

func (n *Node) connectBootstrapPeers(ctx context.Context, peers []peer.AddrInfo) {
	for _, bootstrapPeer := range peers {
		bootstrapPeer := bootstrapPeer
		go func() {
			connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := n.Host.Connect(connectCtx, bootstrapPeer); err != nil {
				log.Printf("bootstrap connect failed peer=%s err=%v", bootstrapPeer.ID.String(), err)
				return
			}
			log.Printf("bootstrap connected peer=%s", bootstrapPeer.ID.String())
		}()
	}
}

func bootstrapPeersFromConfig(values []string) ([]peer.AddrInfo, error) {
	if len(values) == 0 {
		return dht.GetDefaultBootstrapPeerAddrInfos(), nil
	}

	peers := make([]peer.AddrInfo, 0, len(values))
	for _, value := range values {
		info, err := peer.AddrInfoFromString(value)
		if err != nil {
			return nil, fmt.Errorf("parse bootstrap peer %q: %w", value, err)
		}
		peers = append(peers, *info)
	}
	return peers, nil
}

func libp2pPrivateKey(key ed25519.PrivateKey) (libp2pcrypto.PrivKey, error) {
	stdKey := key
	priv, _, err := libp2pcrypto.KeyPairFromStdKey(&stdKey)
	if err != nil {
		return nil, err
	}
	return priv, nil
}
