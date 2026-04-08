package daemon

import (
	"net/http"
	"strings"
	"time"
)

func (d *Daemon) handleDiscoverPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/discover",
		})
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	skills := splitQueryList(r.URL.Query().Get("skills"))
	limit := queryInt(r, "limit", 20)
	started := time.Now()
	results, err := d.store.SearchAgents(r.Context(), q, skills, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local agent directory store",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":   len(results),
		"query":   q,
		"skills":  skills,
		"results": results,
		"elapsed": time.Since(started).Round(time.Millisecond).String(),
	})
}
