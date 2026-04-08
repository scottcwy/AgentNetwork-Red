package daemon

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"agentnetwork-red/internal/identity"
	"agentnetwork-red/internal/p2p"
	"agentnetwork-red/internal/store"
)

const maxTipQRCodeBytes = 2 << 20

type tipInfo struct {
	Available   bool   `json:"available"`
	ClaimantDID string `json:"claimant_did,omitempty"`
	Message     string `json:"message,omitempty"`
	QRCodeURL   string `json:"qrcode_url,omitempty"`
}

func (d *Daemon) handleTipQRCodeCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		d.handleTipQRCodeUpload(w, r)
	case http.MethodDelete:
		if err := d.store.DeleteTipQRCode(r.Context(), d.identity.DID); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Message:    err.Error(),
				Suggestion: "check the local tip store health",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use POST or DELETE /api/tip/qrcode",
		})
	}
}

func (d *Daemon) handleTipQRCodeUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxTipQRCodeBytes+(1<<20))

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "missing file",
			Suggestion: "upload multipart/form-data with a file field",
		})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxTipQRCodeBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "retry with a valid png or jpeg file",
		})
		return
	}
	if len(data) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "empty file",
			Suggestion: "upload a non-empty png or jpeg image",
		})
		return
	}
	if len(data) > maxTipQRCodeBytes {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "file too large",
			Suggestion: "limit the qrcode image to 2MB or less",
		})
		return
	}

	mimeType, ok := detectTipMimeType(data)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "unsupported image type",
			Suggestion: "upload a png or jpeg image with valid magic bytes",
		})
		return
	}

	filename := filepath.Base(strings.TrimSpace(header.Filename))
	if filename == "." || filename == "/" || filename == "" {
		filename = "tip-qrcode"
	}

	if err := d.store.UpsertTipQRCode(r.Context(), store.TipQRCode{
		DID:       d.identity.DID,
		Filename:  filename,
		MimeType:  mimeType,
		Data:      data,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local tip store health",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":  true,
		"did": d.identity.DID,
		"url": tipQRCodeURL(d.identity.DID),
	})
}

func (d *Daemon) handleTipQRCodeItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/tip/qrcode/{did}",
		})
		return
	}

	did, err := tipDIDFromPath(r.URL.Path, "/api/tip/qrcode/")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "request /api/tip/qrcode/{did:key...}",
		})
		return
	}

	tip, found, err := d.resolveTipQRCode(r.Context(), did, true)
	if err != nil {
		status := http.StatusBadGateway
		suggestion := "ensure the claimant node is reachable and has a qrcode configured"
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
			suggestion = "the requested did has not uploaded a qrcode"
		}
		if strings.Contains(err.Error(), "invalid did:key") {
			status = http.StatusBadRequest
			suggestion = "provide a valid did:key path segment"
		}
		writeJSON(w, status, errorResponse{
			Message:    err.Error(),
			Suggestion: suggestion,
		})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, errorResponse{
			Message:    "tip qrcode not found",
			Suggestion: "the requested did has not uploaded a qrcode",
		})
		return
	}

	w.Header().Set("Content-Type", tip.MimeType)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(tip.Data)
}

func (d *Daemon) handleTipStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/tip/status/{did}",
		})
		return
	}

	did, err := tipDIDFromPath(r.URL.Path, "/api/tip/status/")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "request /api/tip/status/{did:key...}",
		})
		return
	}

	_, found, err := d.resolveTipQRCode(r.Context(), did, false)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusBadGateway, errorResponse{
			Message:    err.Error(),
			Suggestion: "ensure the target node is reachable and has p2p enabled",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"did":        did,
		"has_qrcode": found,
	})
}

func (d *Daemon) handleTipRequest(ctx context.Context, req p2p.TipRequest) (p2p.TipResponse, error) {
	if req.DID != d.identity.DID {
		return p2p.TipResponse{Found: false}, nil
	}

	tip, err := d.store.GetTipQRCode(ctx, req.DID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return p2p.TipResponse{Found: false}, nil
		}
		return p2p.TipResponse{}, err
	}

	resp := p2p.TipResponse{
		Found:     true,
		Filename:  tip.Filename,
		MimeType:  tip.MimeType,
		UpdatedAt: tip.UpdatedAt,
	}
	if req.IncludeData {
		resp.Data = base64.StdEncoding.EncodeToString(tip.Data)
	}
	return resp, nil
}

func (d *Daemon) resolveTipInfo(ctx context.Context, did string) tipInfo {
	if strings.TrimSpace(did) == "" {
		return tipInfo{Available: false}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, found, err := d.resolveTipQRCode(timeoutCtx, did, false)
	if err != nil || !found {
		return tipInfo{Available: false}
	}

	return tipInfo{
		Available:   true,
		ClaimantDID: did,
		Message:     "任务已完成！认领者设置了收款码，可扫码打赏。",
		QRCodeURL:   tipQRCodeURL(did),
	}
}

func (d *Daemon) resolveTipQRCode(ctx context.Context, did string, includeData bool) (store.TipQRCode, bool, error) {
	tip, err := d.store.GetTipQRCode(ctx, did)
	if err == nil {
		if !includeData {
			tip.Data = nil
		}
		return tip, true, nil
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return store.TipQRCode{}, false, err
	}

	if did == d.identity.DID {
		return store.TipQRCode{}, false, store.ErrNotFound
	}
	if d.p2p == nil {
		return store.TipQRCode{}, false, store.ErrNotFound
	}

	peerID, err := identity.PeerIDFromDID(did)
	if err != nil {
		return store.TipQRCode{}, false, err
	}

	resp, err := d.p2p.RequestTip(ctx, peerID, p2p.TipRequest{
		DID:         did,
		IncludeData: includeData,
	})
	if err != nil {
		return store.TipQRCode{}, false, err
	}
	if !resp.Found {
		return store.TipQRCode{}, false, store.ErrNotFound
	}

	remoteTip := store.TipQRCode{
		DID:       did,
		Filename:  resp.Filename,
		MimeType:  resp.MimeType,
		UpdatedAt: resp.UpdatedAt,
	}
	if includeData && resp.Data != "" {
		data, err := base64.StdEncoding.DecodeString(resp.Data)
		if err != nil {
			return store.TipQRCode{}, false, err
		}
		remoteTip.Data = data
	}

	return remoteTip, true, nil
}

func detectTipMimeType(data []byte) (string, bool) {
	switch http.DetectContentType(data) {
	case "image/png":
		return "image/png", true
	case "image/jpeg":
		return "image/jpeg", true
	default:
		return "", false
	}
}

func tipDIDFromPath(path string, prefix string) (string, error) {
	if !strings.HasPrefix(path, prefix) {
		return "", errors.New("invalid tip path")
	}
	raw := strings.TrimPrefix(path, prefix)
	if raw == "" {
		return "", errors.New("missing did")
	}
	return url.PathUnescape(raw)
}

func tipQRCodeURL(did string) string {
	return "/api/tip/qrcode/" + url.PathEscape(did)
}
