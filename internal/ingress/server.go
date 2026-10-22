// Package ingress provides a safe, durable preview-only webhook intake.
// It NEVER sends a WhatsApp message or calls the CRM.
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
	Enabled        bool
	AdsSecret      string
	GreetingSecret string
	Core           core.Config
	SpoolDir       string
	OnDecision     func(core.Decision)
	Registry       *instances.Registry
	Secrets        map[string]string // Slug -> Secret
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

	r.Body = http.MaxBytesReader(w, r.Body, 65536)
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
		w.WriteHeader(204)
		return
	}

	// Stable key for idempotency. Do not embed numbers/names in filenames.
	key := sha256.Sum256([]byte(uuid + "\x00" + "Message" + "\x00" + msgID))
	name := hex.EncodeToString(key[:]) + ".json"

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
	err = os.Link(tmp.Name(), filepath.Join(s.settings.SpoolDir, name))
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
	if s.settings.OnDecision != nil {
		go s.settings.OnDecision(decision)
	}
	w.WriteHeader(202)
}
