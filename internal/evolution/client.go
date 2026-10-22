// Package evolution provides a minimal Evolution Go 0.7.2 HTTP adapter.
// Not wired to the preview server. Caller must persist attempt BEFORE invoking.
package evolution

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrUnconfirmed       = errors.New("evolution: outbound result unconfirmed; do not auto-retry")
	ErrInvalid           = errors.New("evolution: invalid request")
	ErrStatusUnavailable = errors.New("evolution: status unavailable")
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type Status struct {
	Connected bool   `json:"Connected"`
	LoggedIn  bool   `json:"LoggedIn"`
	Name      string `json:"Name"`
}

func (c Client) base() (string, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") ||
		u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		u.Path != "" || c.Token == "" {
		return "", ErrInvalid
	}
	return c.BaseURL, nil
}

func (c Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	// Media transcription/ffmpeg can take longer than text transmission.
	// Explicit timeouts still classify unknown outcomes as unconfirmed.
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (c Client) send(ctx context.Context, path string, payload any) (string, error) {
	base, err := c.base()
	if err != nil {
		return "", ErrInvalid
	}
	b, err := json.Marshal(payload)
	if err != nil || len(b) > 40<<20 {
		return "", ErrInvalid
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(b))
	if err != nil {
		return "", ErrInvalid
	}
	req.Header.Set("apikey", c.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client().Do(req)
	if err != nil {
		return "", ErrUnconfirmed
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil || len(raw) > 65536 || res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", ErrUnconfirmed
	}
	var response struct {
		Data struct {
			Info struct {
				ID string `json:"ID"`
			} `json:"Info"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &response) != nil || response.Data.Info.ID == "" {
		return "", ErrUnconfirmed
	}
	return response.Data.Info.ID, nil
}

func (c Client) SendText(ctx context.Context, phone, text string) (string, error) {
	return c.SendTextWithID(ctx, phone, text, "")
}

// ID is a correlation primitive, NOT proof that retry after timeout is safe.
func (c Client) SendTextWithID(ctx context.Context, phone, text, id string) (string, error) {
	if phone == "" || strings.TrimSpace(text) == "" {
		return "", ErrInvalid
	}
	p := map[string]string{"number": phone, "text": text}
	if id != "" {
		p["id"] = id
	}
	return c.send(ctx, "/send/text", p)
}

func (c Client) SendMedia(ctx context.Context, phone, mediaType string, file []byte, caption string) (string, error) {
	return c.SendMediaWithID(ctx, phone, mediaType, file, caption, "")
}

func (c Client) SendMediaWithID(ctx context.Context, phone, mediaType string, file []byte, caption, id string) (string, error) {
	if phone == "" || (mediaType != "image" && mediaType != "audio" && mediaType != "video") || len(file) == 0 || len(file) > 25<<20 {
		return "", ErrInvalid
	}
	p := map[string]string{
		"number":  phone,
		"type":    mediaType,
		"url":     base64.StdEncoding.EncodeToString(file),
		"caption": caption,
	}
	if id != "" {
		p["id"] = id
	}
	return c.send(ctx, "/send/media", p)
}

func (c Client) SendDocument(ctx context.Context, phone string, file []byte, filename string) (string, error) {
	return c.SendDocumentWithID(ctx, phone, file, filename, "")
}

func (c Client) SendDocumentWithID(ctx context.Context, phone string, file []byte, filename, id string) (string, error) {
	if phone == "" || len(file) == 0 || len(file) > 25<<20 {
		return "", ErrInvalid
	}
	if filename == "" {
		filename = "document.pdf"
	}
	p := map[string]string{
		"number":   phone,
		"type":     "document",
		"url":      base64.StdEncoding.EncodeToString(file),
		"filename": filename,
	}
	if id != "" {
		p["id"] = id
	}
	return c.send(ctx, "/send/media", p)
}

// Status uses the existing read-only Evolution Go endpoint. The status is a
// hint, not a guarantee; an instance may disconnect immediately after read.
func (c Client) Status(ctx context.Context) (Status, error) {
	base, err := c.base()
	if err != nil {
		return Status{}, ErrInvalid
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/instance/status", nil)
	if err != nil {
		return Status{}, ErrInvalid
	}
	req.Header.Set("apikey", c.Token)
	res, err := c.client().Do(req)
	if err != nil {
		return Status{}, ErrStatusUnavailable
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 32769))
	if err != nil || len(body) > 32768 || res.StatusCode != http.StatusOK {
		return Status{}, ErrStatusUnavailable
	}
	var payload struct {
		Message string `json:"message"`
		Data    Status `json:"data"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.Message != "success" {
		return Status{}, ErrStatusUnavailable
	}
	return payload.Data, nil
}
