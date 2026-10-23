// Package ingress validates and durably acknowledges inbound Evolution Go events.
// The outbound worker, when enabled, processes events asynchronously.
package ingress

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/instances"
)

type Settings struct {
	Enabled          bool
	AdsSecret        string
	GreetingSecret   string
	Core             core.Config
	SpoolDir         string
	StaticDir        string
	OnDecision       func(core.Decision) error
	Registry         *instances.Registry
	Secrets          map[string]string // Slug -> Secret
	MaxBodyBytes     int64
	DashboardHandler http.Handler
}

type Server struct {
	settings Settings
}

func New(c Settings) (*Server, error) {
	if c.SpoolDir == "" {
		return nil, errors.New("spool dir missing")
	}
	if c.Enabled {
		if c.Registry != nil {
			if len(c.Secrets) == 0 {
				return nil, errors.New("registry mode requires configured secrets")
			}
			for _, sec := range c.Secrets {
				if len(sec) < 32 {
					return nil, errors.New("each instance requires a 32+ character webhook secret")
				}
			}
		} else {
			if len(c.AdsSecret) < 32 || len(c.GreetingSecret) < 32 || c.Core.AdsUUID == "" || c.Core.GreetingUUID == "" || c.Core.AdsUUID == c.Core.GreetingUUID {
				return nil, errors.New("enable requires separate UUID mappings and two 32+ character webhook secrets")
			}
		}
	}
	if err := os.MkdirAll(c.SpoolDir, 0700); err != nil {
		return nil, err
	}
	return &Server{settings: c}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" && r.Method == "GET" {
		w.WriteHeader(200)
		_, _ = io.WriteString(w, "ok")
		return
	}
	if r.URL.Path == "/readyz" && r.Method == "GET" {
		if !s.settings.Enabled {
			http.Error(w, "preview disabled", 503)
			return
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, "ready")
		return
	}
	if strings.HasPrefix(r.URL.Path, "/static/") && (r.Method == http.MethodGet || r.Method == http.MethodHead) && s.settings.StaticDir != "" {
		http.StripPrefix("/static/", http.FileServer(http.Dir(s.settings.StaticDir))).ServeHTTP(w, r)
		return
	}
	if (r.URL.Path == "/painel" || r.URL.Path == "/dashboard" || strings.HasPrefix(r.URL.Path, "/api/dashboard/")) && r.Method == http.MethodGet {
		if s.settings.DashboardHandler != nil {
			s.settings.DashboardHandler.ServeHTTP(w, r)
			return
		}
	}
	// NO request logging: URL contains a secret.
	if r.Method != "POST" {
		http.Error(w, "not found", 404)
		return
	}

	var uuid string
	var decision core.Decision
	var parseAndDecide func(body []byte) (core.Decision, string, error)

	if s.settings.Registry != nil {
		if !s.settings.Enabled {
			http.Error(w, "service disabled", 503)
			return
		}
		// Expect path format: /webhook/<slug>/<secret>
		trimmed := strings.TrimPrefix(r.URL.Path, "/webhook/")
		if trimmed == r.URL.Path {
			http.Error(w, "not found", 404)
			return
		}
		parts := strings.Split(trimmed, "/")
		if len(parts) != 2 {
			http.Error(w, "unauthorized", 401)
			return
		}
		slug := parts[0]
		sec, ok := s.settings.Secrets[slug]
		if !ok || len(sec) < 32 {
			http.Error(w, "unauthorized", 401)
			return
		}
		def, err := s.settings.Registry.VerifyRoute(r.URL.Path, slug, sec)
		if err != nil {
			http.Error(w, "unauthorized", 401)
			return
		}
		uuid = def.InstanceUUID
		parseAndDecide = func(body []byte) (core.Decision, string, error) {
			e, err := core.Parse(body)
			if err != nil {
				return core.Decision{}, "", err
			}
			if e.InstanceID != uuid {
				return core.Decision{}, "", errors.New("wrong instance")
			}
			dec, err := s.settings.Registry.Decide(e)
			return dec, e.Data.Info.ID, err
		}
	} else {
		var expected string
		switch {
		case strings.HasPrefix(r.URL.Path, "/webhook/ads/"):
			expected = s.settings.AdsSecret
			uuid = s.settings.Core.AdsUUID
		case strings.HasPrefix(r.URL.Path, "/webhook/greeting/"):
			expected = s.settings.GreetingSecret
			uuid = s.settings.Core.GreetingUUID
		default:
			http.Error(w, "not found", 404)
			return
		}
		if !s.settings.Enabled {
			http.Error(w, "service disabled", 503)
			return
		}
		prefix := "/webhook/ads/"
		if uuid == s.settings.Core.GreetingUUID {
			prefix = "/webhook/greeting/"
		}
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.Error(w, "unauthorized", 401)
			return
		}
		secret := strings.TrimPrefix(r.URL.Path, prefix)
		if secret == "" || strings.Contains(secret, "/") || strings.Contains(secret, "\\") ||
			subtle.ConstantTimeCompare([]byte(secret), []byte(expected)) != 1 {
			http.Error(w, "unauthorized", 401)
			return
		}
		parseAndDecide = func(body []byte) (core.Decision, string, error) {
			e, err := core.Parse(body)
			if err != nil {
				return core.Decision{}, "", err
			}
			if e.InstanceID != uuid {
				return core.Decision{}, "", errors.New("wrong instance")
			}
			return core.Select(e, s.settings.Core), e.Data.Info.ID, nil
		}
	}

	limit := s.settings.MaxBodyBytes
	if limit <= 0 {
		limit = 32 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "payload too large or unreadable", 413)
		return
	}

	var msgID string
	decision, msgID, err = parseAndDecide(body)
	if err != nil {
		if err.Error() == "wrong instance" {
			http.Error(w, "wrong instance", 403)
			return
		}
		http.Error(w, "bad JSON", 400)
		return
	}

	if decision.Action == core.Ignore {
		// Opt-out is an actionable control event, never silently drop it.
		if decision.Reason == "opt_out_requires_persistent_suppression" && s.settings.OnDecision != nil {
			if err := s.settings.OnDecision(decision); err != nil {
				http.Error(w, "durable suppression unavailable", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Stable key for idempotency. Do not embed numbers/names in filenames.
	key := sha256.Sum256([]byte(uuid + "\x00" + "Message" + "\x00" + msgID))
	name := hex.EncodeToString(key[:]) + ".json"
	journalPath := filepath.Join(s.settings.SpoolDir, name)
	if _, err := os.Stat(journalPath); err == nil {
		// Previously ACKed and durably accepted by both the outbox and spool.
		w.WriteHeader(http.StatusAccepted)
		return
	} else if !os.IsNotExist(err) {
		http.Error(w, "spool unavailable", http.StatusServiceUnavailable)
		return
	}
	// Persist outbox tasks BEFORE acknowledging the webhook. Never detach this
	// operation into a goroutine; a crash could otherwise lose an ACKed event.
	if s.settings.OnDecision != nil {
		if err := s.settings.OnDecision(decision); err != nil {
			http.Error(w, "durable outbox unavailable", http.StatusServiceUnavailable)
			return
		}
	}

	// Atomic: only expose a fully written record. Hard link refuses overwrite.
	tmp, err := os.CreateTemp(s.settings.SpoolDir, ".tmp-*")
	if err != nil {
		http.Error(w, "spool unavailable", 503)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if err = tmp.Chmod(0600); err != nil {
		http.Error(w, "spool unavailable", 503)
		return
	}
	record := struct {
		Decision   core.Decision `json:"decision"`
		ReceivedAt string        `json:"received_at"`
		Status     string        `json:"status"`
	}{decision, time.Now().UTC().Format(time.RFC3339Nano), "planned_preview_only"}
	if err = json.NewEncoder(tmp).Encode(record); err != nil {
		http.Error(w, "spool unavailable", 503)
		return
	}
	if err = tmp.Sync(); err != nil {
		http.Error(w, "spool unavailable", 503)
		return
	}
	if err = tmp.Close(); err != nil {
		http.Error(w, "spool unavailable", 503)
		return
	}
	err = os.Link(tmp.Name(), journalPath)
	if os.IsExist(err) {
		w.WriteHeader(202)
		return
	}
	if err != nil {
		http.Error(w, "spool unavailable", 503)
		return
	}
	// Delivery of inbound webhook is accepted only after a durable record exists.
	if d, err := os.Open(s.settings.SpoolDir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	w.WriteHeader(202)
}
