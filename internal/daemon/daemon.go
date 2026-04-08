package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"agentnetwork-red/internal/config"
	"agentnetwork-red/internal/identity"
	"agentnetwork-red/internal/p2p"
	"agentnetwork-red/internal/store"
)

type Daemon struct {
	cfg      config.Config
	identity *identity.Identity
	p2p      *p2p.Node
	store    *store.Store
	version  string
	started  time.Time
	server   *http.Server
}

type statusResponse struct {
	Version        string `json:"version"`
	DID            string `json:"did"`
	PeerID         string `json:"peer_id"`
	ConnectedPeers int    `json:"connected_peers"`
	Uptime         string `json:"uptime"`
}

type errorResponse struct {
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func New(cfg config.Config, ident *identity.Identity, st *store.Store, version string) *Daemon {
	return &Daemon{
		cfg:      cfg,
		identity: ident,
		store:    st,
		version:  version,
	}
}

func (d *Daemon) WithP2P(node *p2p.Node) *Daemon {
	d.p2p = node
	if d.p2p != nil {
		d.p2p.SetDMHandler(d.receiveDMPayload)
		d.p2p.SetTaskHandler(d.receiveTaskPayload)
	}
	return d
}

func (d *Daemon) Start(ctx context.Context) error {
	d.started = time.Now().UTC()

	mux := http.NewServeMux()
	mux.Handle("/api/status", d.withOptionalAuth(http.HandlerFunc(d.handleStatus)))
	mux.Handle("/api/shutdown", d.withOptionalAuth(http.HandlerFunc(d.handleShutdown)))
	mux.Handle("/api/credits/balance", d.withOptionalAuth(http.HandlerFunc(d.handleBalance)))
	mux.Handle("/api/credits/events", d.withOptionalAuth(http.HandlerFunc(d.handleCreditEvents)))
	mux.Handle("/api/peers", d.withOptionalAuth(http.HandlerFunc(d.handlePeers)))
	mux.Handle("/api/peers/connect", d.withOptionalAuth(http.HandlerFunc(d.handleConnectPeer)))
	mux.Handle("/api/dm/send", d.withOptionalAuth(http.HandlerFunc(d.handleDMSend)))
	mux.Handle("/api/dm/inbox", d.withOptionalAuth(http.HandlerFunc(d.handleDMInbox)))
	mux.Handle("/api/dm/thread/", d.withOptionalAuth(http.HandlerFunc(d.handleDMThread)))
	mux.Handle("/api/tasks/board/stats", d.withOptionalAuth(http.HandlerFunc(d.handleTaskBoardStats)))
	mux.Handle("/api/tasks/board", d.withOptionalAuth(http.HandlerFunc(d.handleTaskBoard)))
	mux.Handle("/api/tasks/", d.withOptionalAuth(http.HandlerFunc(d.handleTaskItem)))
	mux.Handle("/api/tasks", d.withOptionalAuth(http.HandlerFunc(d.handleTasksCollection)))
	mux.Handle("/ui/board", d.withOptionalAuth(http.HandlerFunc(d.handleBoardUI)))

	d.server = &http.Server{
		Addr:              d.Addr(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := d.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("redanet daemon listening on http://%s", d.Addr())
	return d.server.ListenAndServe()
}

func (d *Daemon) Shutdown(ctx context.Context) error {
	if d.server == nil {
		return nil
	}
	return d.server.Shutdown(ctx)
}

func (d *Daemon) Addr() string {
	return fmt.Sprintf("%s:%d", d.cfg.APIHost, d.cfg.APIPort)
}

func (d *Daemon) withOptionalAuth(next http.Handler) http.Handler {
	if d.cfg.APIToken == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+d.cfg.APIToken {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Message:    "unauthorized",
				Suggestion: "provide a valid Bearer token",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (d *Daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/status",
		})
		return
	}

	resp := statusResponse{
		Version:        d.version,
		DID:            d.identity.DID,
		PeerID:         d.peerID(),
		ConnectedPeers: len(d.connectedPeers()),
		Uptime:         time.Since(d.started).Round(time.Second).String(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (d *Daemon) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use POST /api/shutdown",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	go func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = d.Shutdown(shutdownCtx)
	}()
}

func (d *Daemon) handleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/credits/balance",
		})
		return
	}

	balance, err := d.store.Balance(r.Context(), d.identity.DID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local store health",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"did":     d.identity.DID,
		"balance": balance,
	})
}

func (d *Daemon) handleCreditEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/credits/events",
		})
		return
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message:    "invalid limit",
				Suggestion: "provide a positive integer limit",
			})
			return
		}
		limit = parsed
	}

	events, err := d.store.ListCreditEvents(r.Context(), d.identity.DID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local store health",
		})
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func (d *Daemon) handlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/peers",
		})
		return
	}

	peers := d.connectedPeers()
	writeJSON(w, http.StatusOK, map[string]any{
		"count": peersCount(peers),
		"peers": peers,
	})
}

func (d *Daemon) handleConnectPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use POST /api/peers/connect",
		})
		return
	}
	if d.p2p == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{
			Message:    "p2p node unavailable",
			Suggestion: "start the daemon with p2p enabled",
		})
		return
	}

	var req struct {
		Addr string `json:"addr"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "provide JSON like {\"addr\":\"/ip4/.../tcp/.../p2p/...\"}",
		})
		return
	}
	if req.Addr == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "missing addr",
			Suggestion: "provide a peer multiaddr in the addr field",
		})
		return
	}

	info, err := d.p2p.Connect(r.Context(), req.Addr)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{
			Message:    err.Error(),
			Suggestion: "verify the peer multiaddr and confirm the remote node is listening",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"peer_id": info.ID.String(),
	})
}

func (d *Daemon) peerID() string {
	if d.p2p == nil {
		return ""
	}
	return d.p2p.PeerID()
}

func (d *Daemon) connectedPeers() []p2p.PeerInfo {
	if d.p2p == nil {
		return nil
	}
	return d.p2p.ConnectedPeers()
}

func peersCount(peers []p2p.PeerInfo) int {
	return len(peers)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSONBody(r *http.Request, target any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty request body")
	}
	return json.Unmarshal(body, target)
}
