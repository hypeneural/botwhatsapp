package ingress

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/instances"
)

func TestWebhookPreviewDurableAndIdempotent(t *testing.T) {
	dir := t.TempDir()
	secret := strings.Repeat("a", 40)
	srv, err := New(Settings{
		Enabled:        true,
		AdsSecret:      secret,
		GreetingSecret: strings.Repeat("b", 40),
		SpoolDir:       dir,
		Core: core.Config{
			AdsUUID:      "uuid-ads",
			GreetingUUID: "uuid-greet",
			AdsTriggers:  []string{"quero proposta natal"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"event":"Message","instanceId":"uuid-ads","instanceToken":"SHOULD_NOT_PERSIST","data":{"Info":{"ID":"msgA","Chat":"5548996425287@s.whatsapp.net","IsFromMe":false},"Message":{"conversation":"quero proposta natal"}}}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook/ads/"+secret, strings.NewReader(body))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code != 202 {
			t.Fatalf("attempt %d code %d: %s", i, w.Code, w.Body)
		}
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(paths) != 1 {
		t.Fatal(err, paths)
	}
	buf, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(buf), "SHOULD_NOT_PERSIST") {
		t.Fatal("instanceToken was persisted")
	}
	var rec struct {
		Decision core.Decision `json:"decision"`
		Status   string        `json:"status"`
	}
	if err := json.Unmarshal(buf, &rec); err != nil || rec.Decision.Action != core.AdsProposal || rec.Status != "planned_preview_only" {
		t.Fatal(err, rec)
	}
	wrong := httptest.NewRecorder()
	srv.ServeHTTP(wrong, httptest.NewRequest("POST", "/webhook/ads/"+strings.Repeat("z", 40), strings.NewReader(body)))
	if wrong.Code != 401 {
		t.Fatal("missing webhook verification", wrong.Code)
	}
	wrong = httptest.NewRecorder()
	srv.ServeHTTP(wrong, httptest.NewRequest("POST", "/webhook/ads/extra/"+secret, strings.NewReader(body)))
	if wrong.Code != 401 {
		t.Fatal("nested path was accepted", wrong.Code)
	}
	wrong = httptest.NewRecorder()
	srv.ServeHTTP(wrong, httptest.NewRequest("POST", "/webhook/ads/"+secret+"/extra", strings.NewReader(body)))
	if wrong.Code != 401 {
		t.Fatal("trailing segment was accepted", wrong.Code)
	}
	wrong = httptest.NewRecorder()
	srv.ServeHTTP(wrong, httptest.NewRequest("POST", "/webhook/ads/"+secret, strings.NewReader(strings.ReplaceAll(body, "uuid-ads", "uuid-greet"))))
	if wrong.Code != 403 {
		t.Fatal("accepted mismatched UUID", wrong.Code)
	}
}

func TestDisabledFailsClosed(t *testing.T) {
	srv, err := New(Settings{SpoolDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest("POST", "/webhook/ads/s", strings.NewReader("{}")))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
}

func TestWebhookRegistryMode(t *testing.T) {
	dir := t.TempDir()
	reg, err := instances.New([]instances.Definition{
		{
			Slug:              "ads",
			InstanceUUID:      "uuid-ads",
			BusinessPhoneE164: "554832200440",
			Role:              instances.Ads,
			Enabled:           true,
			TokenEnv:          "ADS_TOKEN",
			WebhookSecretEnv:  "ADS_SECRET",
			AdsTriggers:       []string{"quero proposta natal"},
			CRMAllowed:        true,
		},
		{
			Slug:              "greeting",
			InstanceUUID:      "uuid-greeting",
			BusinessPhoneE164: "5548996425287",
			Role:              instances.Greeting,
			Enabled:           true,
			TokenEnv:          "GREETING_TOKEN",
			WebhookSecretEnv:  "GREETING_SECRET",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	adsSecret := strings.Repeat("1", 32)
	greetingSecret := strings.Repeat("2", 32)
	srv, err := New(Settings{
		Enabled:  true,
		Registry: reg,
		Secrets: map[string]string{
			"ads":      adsSecret,
			"greeting": greetingSecret,
		},
		SpoolDir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Post to ads endpoint
	bodyAds := `{"event":"Message","instanceId":"uuid-ads","data":{"Info":{"ID":"msg-reg-1","Chat":"5548996425287@s.whatsapp.net","IsFromMe":false},"Message":{"conversation":"quero proposta natal"}}}`
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhook/ads/"+adsSecret, strings.NewReader(bodyAds)))
	if w.Code != 202 {
		t.Fatalf("expected 202 for ads webhook, got %d: %s", w.Code, w.Body)
	}

	// 2. Post to greeting endpoint
	bodyGreeting := `{"event":"Message","instanceId":"uuid-greeting","data":{"Info":{"ID":"msg-reg-2","Chat":"5548996425287@s.whatsapp.net","IsFromMe":false},"Message":{"conversation":"Olá bom dia"}}}`
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/webhook/greeting/"+greetingSecret, strings.NewReader(bodyGreeting)))
	if w2.Code != 202 {
		t.Fatalf("expected 202 for greeting webhook, got %d: %s", w2.Code, w2.Body)
	}

	// 3. Invalid secret rejected with 401
	wBad := httptest.NewRecorder()
	srv.ServeHTTP(wBad, httptest.NewRequest(http.MethodPost, "/webhook/ads/"+strings.Repeat("x", 32), strings.NewReader(bodyAds)))
	if wBad.Code != 401 {
		t.Fatalf("expected 401 for wrong secret, got %d", wBad.Code)
	}

	// 4. Mismatched instance UUID rejected with 403
	wMismatched := httptest.NewRecorder()
	srv.ServeHTTP(wMismatched, httptest.NewRequest(http.MethodPost, "/webhook/ads/"+adsSecret, strings.NewReader(bodyGreeting)))
	if wMismatched.Code != 403 {
		t.Fatalf("expected 403 for mismatched instance UUID, got %d", wMismatched.Code)
	}
}
