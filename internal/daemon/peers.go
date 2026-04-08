package daemon

import (
	"net/http"
	"strings"

	"agentnetwork-red/internal/p2p"
)

func (d *Daemon) handleDiscoverPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/discover",
		})
		return
	}

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	limit := queryInt(r, "limit", 20)
	peers := d.connectedPeers()
	results := make([]p2p.PeerInfo, 0, len(peers))
	for _, peer := range peers {
		if q != "" && !discoverPeerMatches(peer, q) {
			continue
		}
		results = append(results, peer)
		if len(results) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":   len(results),
		"query":   r.URL.Query().Get("q"),
		"skills":  r.URL.Query().Get("skills"),
		"results": results,
	})
}

func discoverPeerMatches(peer p2p.PeerInfo, q string) bool {
	if strings.Contains(strings.ToLower(peer.PeerID), q) {
		return true
	}
	for _, addr := range peer.Addrs {
		if strings.Contains(strings.ToLower(addr), q) {
			return true
		}
	}
	return false
}
