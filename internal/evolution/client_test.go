package evolution

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTextMediaAndUncertain(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("apikey") != "test-token" {
			t.Error("missing auth")
		}
		if r.URL.Path != "/send/text" && r.URL.Path != "/send/media" {
			t.Error(r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"success","data":{"Info":{"ID":"wa-message-123"}}}`))
	}))
	defer server.Close()

	c := Client{BaseURL: server.URL, Token: "test-token"}
	if id, err := c.SendText(context.Background(), "5548996425287", "oi"); err != nil || id != "wa-message-123" {
		t.Fatal(id, err)
	}
	if id, err := c.SendMedia(context.Background(), "5548996425287", "image", []byte("fake-test-fixture"), "legenda"); err != nil || id != "wa-message-123" {
		t.Fatal(id, err)
	}
	if id, err := c.SendDocument(context.Background(), "5548996425287", []byte("%PDF-1.7 dummy"), "Proposta_Natal.pdf"); err != nil || id != "wa-message-123" {
		t.Fatal(id, err)
	}
	if calls != 3 {
		t.Fatal(calls)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer bad.Close()
	_, err := (Client{BaseURL: bad.URL, Token: "test-token"}).SendText(context.Background(), "55", "hello")
	if !errors.Is(err, ErrUnconfirmed) {
		t.Fatal("missing receipt must be unknown", err)
	}
	_, err = c.SendMedia(context.Background(), "55", "document", []byte("bad"), "")
	if !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err = c.SendDocument(context.Background(), "", []byte("%PDF"), "doc.pdf"); !errors.Is(err, ErrInvalid) {
		t.Fatal("empty phone should be invalid", err)
	}
	if _, err = c.SendText(context.Background(), "", strings.Repeat("x", 5)); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}
