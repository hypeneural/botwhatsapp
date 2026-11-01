package dashboard

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"
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
		return false
	}
	given := r.Header.Get("X-Dashboard-Token")
	if _, password, ok := r.BasicAuth(); ok {
		given = password
	}
	if given == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(given), []byte(h.token)) == 1
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.isAuthorized(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Evydencia Bot Panel", charset="UTF-8"`)
		http.Error(w, "unauthorized: dashboard credentials required", http.StatusUnauthorized)
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

	case path == "/api/dashboard/events":
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		history, err := h.service.GetEvents(limit)
		if err != nil {
			http.Error(w, "failed to load event history", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(history)

	case path == "/painel" || path == "/dashboard":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(dashboardHTML))

	default:
		http.NotFound(w, r)
	}
}
