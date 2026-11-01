package crm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNativeProposalScheduleReceipt(t *testing.T) {
	taskID := "event-123-04-schedule_natal_crm"
	id := NativeRequestID(taskID)
	if id != NativeRequestID(taskID) || len(id) != 36 {
		t.Fatal("unstable UUID", id)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/extension/proposals/schedule" || r.Method != "POST" ||
			r.Header.Get("Authorization") != "Bearer scoped-token" ||
			r.Header.Get("X-Idempotency-Key") != id {
			t.Error("invalid request contract")
		}
		body := make([]byte, 2048)
		n, _ := r.Body.Read(body)
		if !strings.Contains(string(body[:n]), `"name":null`) || !strings.Contains(string(body[:n]), `"request_id":"`+id+`"`) {
			t.Error("invalid body")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"status":"scheduled","dispatch_uuid":"b6aa2767-1b38-44e1-a112-85a87e021343"}}`))
	}))
	defer server.Close()
	uuid, err := (NativeClient{BaseURL: server.URL, BearerToken: "scoped-token"}).ScheduleNative(context.Background(), "4898133288", id)
	if err != nil || uuid != "b6aa2767-1b38-44e1-a112-85a87e021343" || calls != 1 {
		t.Fatal(uuid, err, calls)
	}
}
func TestNativeRejectAndUnknownNeverRetry(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		reject bool
	}{
		{422, `{"error":{"code":"recipient_suppressed"}}`, true},
		{401, `{"error":"unauthorized"}`, true},
		{503, `{"error":"unavailable"}`, false},
		{201, `{"data":{"status":"scheduled"}}`, false},
	} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		_, err := (NativeClient{BaseURL: server.URL, BearerToken: "scoped-token"}).ScheduleNative(context.Background(), "4898133288", NativeRequestID("test"))
		server.Close()
		if tc.reject && !errors.Is(err, ErrRejected) || !tc.reject && !errors.Is(err, ErrUnconfirmed) || calls != 1 {
			t.Fatalf("status %d err %v calls %d", tc.status, err, calls)
		}
	}
}

func TestNativeBlocksUnapprovedCredentialOrigin(t *testing.T) {
	for _, origin := range []string{
		"https://example.invalid",
		"https://evydencia.com.attacker.invalid",
		"https://evydencia.com:8443",
		"http://evydencia.com",
		"https://user:pass@evydencia.com",
	} {
		client := NativeClient{BaseURL: origin, BearerToken: "sensitive-fixture-only"}
		_, err := client.ScheduleNative(context.Background(), "4898133288", NativeRequestID("no-network"))
		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s should fail before HTTP request; got %v", origin, err)
		}
	}
}
