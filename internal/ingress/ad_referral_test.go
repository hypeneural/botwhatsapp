package ingress

import (
  "encoding/json"
  "strings"
  "testing"
	"github.com/hypeneural/botwhatsapp/internal/core"
)

func TestParseAdReferenceEvolutionGo072(t *testing.T) {
  tests := []struct {
    name, ref, ext, wantID, wantEvidence string
  }{
    {"protojson_sourceID", `{"sourceType":"ad","sourceID":"120251240551530388","ctwaClid":"NEVER_STORE","sourceURL":"https://example.org/a"}`, "", "120251240551530388", "evolution_referral"},
    {"sourceId", `{"sourceType":"ad","sourceId":"120251241001620388"}`, "", "120251241001620388", "evolution_referral"},
    {"snake_source", `{"source_type":"ad","source_id":"120251241244800388"}`, "", "120251241244800388", "evolution_referral"},
    {"nested_ad", "", `{"sourceType":"ad","sourceID":"120251240551530388"}`, "120251240551530388", "external_ad_reply"},
    {"post_not_ad", `{"sourceType":"post","sourceID":"120251240551530388"}`, "", "", ""},
    {"no_type", `{"sourceID":"120251240551530388"}`, "", "", ""},
    {"textual_guess", `{"sourceType":"ad","sourceID":"VID01"}`, "", "", "evolution_referral"},
    {"empty", "{}", "null", "", ""},
    {"broken", "{oops}", "[]", "", ""},
    {"referral_priority", `{"sourceType":"ad","sourceID":"120251240551530388"}`, `{"sourceType":"ad","sourceID":"120251241001620388"}`, "120251240551530388", "evolution_referral"},
    {"malformed_referral_nested_valid", `{oops}`, `{"sourceType":"ad","sourceID":"120251241001620388"}`, "120251241001620388", "external_ad_reply"},
    {"too_large", `{"sourceType":"ad","sourceID":"` + strings.Repeat("1", 8192) + `"}`, "", "", ""},
  }
  for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
      got := parseAdReference(json.RawMessage(tc.ref), json.RawMessage(tc.ext))
      if got.SourceID != tc.wantID || got.Evidence != tc.wantEvidence { t.Fatalf("got %+v, want ID %q evidence %q", got, tc.wantID, tc.wantEvidence) }
      b, err := json.Marshal(got); if err != nil { t.Fatal(err) }
      if strings.Contains(string(b), "NEVER_STORE") || strings.Contains(string(b), "sourceURL") { t.Fatal("private referral data stored") }
    })
  }
}

// A source reference never implies that a campaign was delivered or a sale was made.
func TestInboundAuditRecordsNoClickID(t *testing.T) {
  input := []byte(`{"event":"Message","data":{"referral":{"sourceType":"ad","sourceID":"120251240551530388","ctwaClid":"HIDDEN_CLICK_ID"}}}`)
  ref := parseAdReference(json.RawMessage(`{"sourceType":"ad","sourceID":"120251240551530388","ctwaClid":"HIDDEN_CLICK_ID"}`), nil)
  got, _ := json.Marshal(ref)
  if !strings.Contains(string(got), "120251240551530388") || strings.Contains(string(got), "HIDDEN_CLICK_ID") { t.Fatalf("bad sanitized data: %s", got) }
  if !json.Valid(input) { t.Fatal("invalid fixture") }
}

func TestInboundAuditPersistsSourceWithoutClickIdentifier(t *testing.T) {
	input := []byte(`{"event":"Message","instanceId":"ads","data":{"Info":{"ID":"ad-1","Chat":"5548999999999@s.whatsapp.net"},"Message":{"conversation":"Olá"},"referral":{"sourceType":"ad","sourceID":"120251240551530388","ctwaClid":"DONT_STORE_CLICK"}}}`)
	env, err := core.Parse(input)
	if err != nil { t.Fatal(err) }
	dec := core.Select(env, core.Config{AdsUUID:"ads", AdsTriggers: []string{"Quero conhecer os pacotes de Natal em Tijucas"}, AdsFallback:true})
	got := auditInbound(input, dec)
	if got.AdSourceID != "120251240551530388" || got.AdEvidence != "evolution_referral" || got.Source != "meta_ads" {
		t.Fatalf("wrong source audit: %+v",got)
	}
	persisted, err := json.Marshal(got)
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(persisted),"DONT_STORE_CLICK") { t.Fatal("click identifier leaked into journal") }
}
