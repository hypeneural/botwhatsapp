package dashboard

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// Handler serves HTTP endpoints for the contact and traffic dashboard.
type Handler struct {
	service *Service
	token   string
}

// NewHandler creates a new dashboard HTTP handler.
func NewHandler(dataDir, token string) *Handler {
	return &Handler{
		service: NewService(dataDir),
		token:   strings.TrimSpace(token),
	}
}

func (h *Handler) isAuthorized(r *http.Request) bool {
	if h.token == "" {
		return true
	}
	given := r.URL.Query().Get("token")
	if given == "" {
		given = r.Header.Get("X-Dashboard-Token")
	}
	if given == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(given), []byte(h.token)) == 1
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.isAuthorized(r) {
		http.Error(w, "unauthorized: dashboard token required", http.StatusUnauthorized)
		return
	}

	path := r.URL.Path

	switch {
	case path == "/api/dashboard/stats":
		summary, err := h.service.GetSummary()
		if err != nil {
			http.Error(w, "failed to load stats", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary.Stats)

	case path == "/api/dashboard/contacts":
		summary, err := h.service.GetSummary()
		if err != nil {
			http.Error(w, "failed to load contacts", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary)

	case path == "/painel" || path == "/dashboard":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(dashboardHTML))

	default:
		http.NotFound(w, r)
	}
}
