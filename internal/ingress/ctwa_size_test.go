package ingress

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

func TestCTWAReferralThumbnailEnvelopeLimit(t *testing.T) {
	secret := strings.Repeat("s", 40)
	srv, err := New(Settings{Enabled: true, SpoolDir: t.TempDir(),
		AdsSecret: secret, GreetingSecret: strings.Repeat("t", 40),
		Core: core.Config{AdsUUID: "ads-uuid", GreetingUUID: "greeting-uuid",
			AdsTriggers: []string{"Quero conhecer os pacotes do Primeiro Natal em Tijucas"}},
		OnDecision: func(core.Decision) error { return nil }, MaxBodyBytes: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	prefix := `{"event":"Message","instanceId":"ads-uuid","data":{"Info":{"ID":"long-ctwa","Chat":"5548999999999@s.whatsapp.net"},"Message":{"extendedTextMessage":{"text":"Quero conhecer os pacotes do Primeiro Natal em Tijucas","contextInfo":{"externalAdReply":{"thumbnail":"`
	suffix := `"}}}}}}`
	for _, tc := range []struct{ size, want int }{{300 << 10, 202}, {2 << 20, 413}} {
		body := prefix + strings.Repeat("A", tc.size) + suffix
		req := httptest.NewRequest(http.MethodPost, "/webhook/ads/"+secret, strings.NewReader(body))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Fatalf("size=%d status=%d want=%d", tc.size, w.Code, tc.want)
		}
	}
}
