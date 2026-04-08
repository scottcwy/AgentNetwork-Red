package daemon

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agentnetwork-red/internal/store"
)

type stringList []string

func (s *stringList) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	switch trimmed {
	case "", "null":
		*s = nil
		return nil
	}

	var array []string
	if err := json.Unmarshal(data, &array); err == nil {
		*s = array
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = splitQueryList(single)
		return nil
	}
	return errors.New("expected string or array of strings")
}

func (d *Daemon) handleProfilePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use POST /api/profile/publish",
		})
		return
	}

	var req struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Skills      stringList `json:"skills"`
		Tags        stringList `json:"tags"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "provide JSON like {\"name\":\"Alice\",\"skills\":[\"coding\"]}",
		})
		return
	}

	profile, err := d.store.UpsertProfile(r.Context(), store.AgentProfile{
		DID:         d.identity.DID,
		Name:        req.Name,
		Description: req.Description,
		Skills:      []string(req.Skills),
		Tags:        []string(req.Tags),
		PeerID:      d.peerID(),
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "verify the profile fields and try again",
		})
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (d *Daemon) handleProfileGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/profile or /api/profile/{did}",
		})
		return
	}

	did := d.identity.DID
	if strings.HasPrefix(r.URL.Path, "/api/profile/") {
		unescaped, err := url.PathUnescape(strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/profile/")))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message:    err.Error(),
				Suggestion: "request /api/profile/{did}",
			})
			return
		}
		did = unescaped
	}

	profile, err := d.store.GetProfile(r.Context(), did)
	if err != nil {
		status := http.StatusInternalServerError
		suggestion := "check the local profile store"
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
			suggestion = "publish a profile first with POST /api/profile/publish"
		}
		writeJSON(w, status, errorResponse{
			Message:    err.Error(),
			Suggestion: suggestion,
		})
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (d *Daemon) handleANSRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use POST /api/ans/register?confirm=yes",
		})
		return
	}
	if strings.ToLower(strings.TrimSpace(r.URL.Query().Get("confirm"))) != "yes" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "missing confirm=yes",
			Suggestion: "retry with /api/ans/register?confirm=yes",
		})
		return
	}

	var req struct {
		Name string     `json:"name"`
		Tags stringList `json:"tags"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message:    err.Error(),
				Suggestion: "provide JSON like {\"name\":\"alice\",\"tags\":[\"coding\"]}",
			})
			return
		}
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(r.URL.Query().Get("name"))
	}
	tags := []string(req.Tags)
	if len(tags) == 0 {
		tags = splitQueryList(r.URL.Query().Get("tags"))
	}

	record, err := d.store.RegisterANS(r.Context(), name, d.identity.DID, tags)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    err.Error(),
			Suggestion: "use a unique lowercase name like alice-bot",
		})
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (d *Daemon) handleANSResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/ans/resolve?name=alice",
		})
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	record, err := d.store.ResolveANS(r.Context(), name)
	if err != nil {
		status := http.StatusBadRequest
		suggestion := "provide a valid ans name"
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
			suggestion = "register the name first with POST /api/ans/register?confirm=yes"
		}
		writeJSON(w, status, errorResponse{
			Message:    err.Error(),
			Suggestion: suggestion,
		})
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (d *Daemon) handleANSLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/ans/lookup?tags=coding,translation",
		})
		return
	}

	tags := splitQueryList(r.URL.Query().Get("tags"))
	if len(tags) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message:    "missing tags",
			Suggestion: "provide one or more comma-separated tags",
		})
		return
	}

	limit := queryInt(r, "limit", 20)
	started := time.Now()
	results, err := d.store.LookupAgents(r.Context(), tags, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local agent directory store",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":   len(results),
		"tags":    tags,
		"results": results,
		"elapsed": time.Since(started).Round(time.Millisecond).String(),
	})
}

func splitQueryList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
