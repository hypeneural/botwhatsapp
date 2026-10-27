package crm

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// NativeClient calls the CRM's existing individual Natal proposal scheduler.
// The extension endpoint requires a dedicated Sanctum token with the
// extension:proposal-write ability and a user permitted to send the Natal flow.
// Never reuse the legacy raw Authorization token or fall back to the batch API.
type NativeClient struct {
	BaseURL     string
	BearerToken string
	HTTP        *http.Client
}

var nativeUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// NativeRequestID deterministically binds an outbox task to its CRM dispatch.
// Stable IDs allow manual reconciliation and CRM-native idempotent replays.
func NativeRequestID(taskID string) string {
	// RFC 4122 UUIDv5 namespace URL.
	namespace := [16]byte{0x6b, 0xa7, 0xb8, 0x11, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	input := append(namespace[:], []byte("evydencia:botwhatsapp:natal:v1:"+taskID)...)
	digest := sha1.Sum(input)
	digest[6] = (digest[6] & 0x0f) | 0x50
	digest[8] = (digest[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", digest[0:4], digest[4:6], digest[6:8], digest[8:10], digest[10:16])
}

func (c NativeClient) preflight() (string, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" ||
		u.Fragment != "" || u.Path != "" || (u.Scheme != "https" && u.Scheme != "http") ||
		(u.Scheme == "http" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") ||
		strings.TrimSpace(c.BearerToken) == "" || strings.TrimSpace(c.BearerToken) != c.BearerToken ||
		strings.ContainsAny(c.BearerToken, "\r\n") {
		return "", ErrInvalidRequest
	}
	return c.BaseURL, nil
}

// ScheduleNative returns the authoritative proposal dispatch UUID, not a legacy
// campaign_scheduled ID. A 2xx response is accepted only with a matching native
// receipt. Transport failures and missing receipts are UNCERTAIN, never retried.
func (c NativeClient) ScheduleNative(ctx context.Context, nationalPhone, requestID string) (string, error) {
	base, err := c.preflight()
	if err != nil || !phoneRE.MatchString(nationalPhone) || !nativeUUIDPattern.MatchString(requestID) {
		return "", ErrInvalidRequest
	}
	body, err := json.Marshal(struct {
		RequestID  string  `json:"request_id"`
		Phone      string  `json:"phone"`
		Name       *string `json:"name"`
		ResendMode string  `json:"resend_mode"`
	}{RequestID: requestID, Phone: nationalPhone, Name: nil, ResendMode: "normal"})
	if err != nil {
		return "", ErrInvalidRequest
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/extension/proposals/schedule", bytes.NewReader(body))
	if err != nil {
		return "", ErrInvalidRequest
	}
	req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", requestID)
	client := &http.Client{Timeout: 15 * time.Second}
	if c.HTTP != nil {
		copyClient := *c.HTTP
		client = &copyClient
	}
	// Do not follow redirects with credentials.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return "", ErrUnconfirmed
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(raw) > 65536 {
		return "", ErrUnconfirmed
	}
	switch resp.StatusCode {
	case 400, 401, 403, 404, 422:
		return "", &RejectedError{StatusCode: resp.StatusCode}
	case 200, 201:
		// The extension response is enveloped: {"data":{...}}.
		var receipt struct {
			Data struct {
				Status       string `json:"status"`
				DispatchUUID string `json:"dispatch_uuid"`
			} `json:"data"`
		}
		if json.Unmarshal(raw, &receipt) != nil ||
			receipt.Data.Status != "scheduled" ||
			!nativeUUIDPattern.MatchString(receipt.Data.DispatchUUID) {
			return "", ErrUnconfirmed
		}
		return receipt.Data.DispatchUUID, nil
	default:
		// Includes 409, 408, 429, 5xx, and unexpected responses.
		return "", ErrUnconfirmed
	}
}
