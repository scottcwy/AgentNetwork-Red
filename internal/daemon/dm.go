package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agentnetwork-red/internal/identity"
	"agentnetwork-red/internal/store"
)

type dmEnvelope struct {
	Msg store.DirectMessage `json:"msg"`
}

func (d *Daemon) handleDMSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use POST /api/dm/send",
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
		To         string `json:"to"`
		Plaintext  string `json:"plaintext,omitempty"`
		Ciphertext string `json:"ciphertext,omitempty"`
		Nonce      string `json:"nonce,omitempty"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "provide JSON like {\"to\":\"did:key:...\",\"plaintext\":\"hello\"}",
		})
		return
	}
	if req.To == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "missing to",
			Suggestion: "provide the recipient did:key in the to field",
		})
		return
	}

	msg := store.DirectMessage{
		ID:        randomID(),
		FromDID:   d.identity.DID,
		ToDID:     req.To,
		Algo:      "nacl",
		Read:      false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	switch {
	case req.Plaintext != "":
		ciphertext, nonce, err := identity.EncryptPlaintext(d.identity.PrivateKey, req.To, req.Plaintext)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message:    err.Error(),
				Suggestion: "verify the recipient DID and plaintext payload",
			})
			return
		}
		msg.Ciphertext = ciphertext
		msg.Nonce = nonce
		msg.LocalPlaintext = req.Plaintext
	case req.Ciphertext != "" && req.Nonce != "":
		msg.Ciphertext = req.Ciphertext
		msg.Nonce = req.Nonce
	default:
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "missing message payload",
			Suggestion: "provide plaintext or ciphertext+nonce",
		})
		return
	}

	peerID, err := identity.PeerIDFromDID(req.To)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "provide a valid recipient did:key",
		})
		return
	}

	if err := d.store.InsertDirectMessage(r.Context(), msg); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local dm store health",
		})
		return
	}

	payload, err := json.Marshal(dmEnvelope{Msg: msg})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "retry the request",
		})
		return
	}

	sendErr := d.p2p.SendDMStream(r.Context(), peerID, payload)
	transport := "stream"
	if sendErr != nil {
		if err := d.p2p.PublishDM(r.Context(), payload); err != nil {
			writeJSON(w, http.StatusBadGateway, errorResponse{
				Message:    err.Error(),
				Suggestion: "ensure the recipient is connected or reachable on pubsub",
			})
			return
		}
		transport = "gossipsub"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sent":      true,
		"id":        msg.ID,
		"transport": transport,
	})
}

func (d *Daemon) handleDMInbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/dm/inbox",
		})
		return
	}

	messages, err := d.store.Inbox(r.Context(), d.identity.DID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local dm store health",
		})
		return
	}

	writeJSON(w, http.StatusOK, d.presentMessages(messages))
}

func (d *Daemon) handleDMThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/dm/thread/{peerDID}",
		})
		return
	}

	peerDID, err := threadPeerDID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "request /api/dm/thread/{peerDID}",
		})
		return
	}

	messages, err := d.store.Thread(r.Context(), d.identity.DID, peerDID, 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local dm store health",
		})
		return
	}

	writeJSON(w, http.StatusOK, d.presentMessages(messages))
}

func (d *Daemon) receiveDMPayload(ctx context.Context, payload []byte) error {
	var envelope dmEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return err
	}

	msg := envelope.Msg
	if msg.ID == "" || msg.FromDID == "" || msg.ToDID == "" || msg.Ciphertext == "" || msg.Nonce == "" {
		return errors.New("invalid dm payload")
	}
	if msg.ToDID != d.identity.DID {
		return nil
	}
	if msg.Algo == "" {
		msg.Algo = "nacl"
	}
	return d.store.InsertDirectMessage(ctx, msg)
}

func (d *Daemon) presentMessages(messages []store.DirectMessage) []store.DirectMessage {
	out := make([]store.DirectMessage, 0, len(messages))
	for _, msg := range messages {
		presented := msg
		if presented.LocalPlaintext == "" && presented.ToDID == d.identity.DID {
			plaintext, err := identity.DecryptCiphertext(d.identity.PrivateKey, presented.FromDID, presented.Ciphertext, presented.Nonce)
			if err == nil {
				presented.LocalPlaintext = plaintext
			}
		}
		out = append(out, presented)
	}
	return out
}

func threadPeerDID(path string) (string, error) {
	const prefix = "/api/dm/thread/"
	if !strings.HasPrefix(path, prefix) {
		return "", errors.New("invalid thread path")
	}
	raw := strings.TrimPrefix(path, prefix)
	if raw == "" {
		return "", errors.New("missing peer did")
	}
	peerDID, err := url.PathUnescape(raw)
	if err != nil {
		return "", err
	}
	return peerDID, nil
}

func randomID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf[:])
}
