package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/outbox"
)

func writeIdentityFixture(t *testing.T, dir, key string, d core.Decision, timestamp time.Time) {
	t.Helper()
	fixture := map[string]interface{}{
		"event_key": key,
		"received_at": timestamp.Format(time.RFC3339Nano),
		"decision": d,
		"inbound": map[string]interface{}{"source": "normal", "message_type": "texto"},
		"status": "received",
	}
	raw, err := json.Marshal(fixture)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, key+".json"), raw, 0600); err != nil { t.Fatal(err) }
}

func TestDashboardUnresolvedLIDNeverBecomesFakePhone(t *testing.T) {
	dir := t.TempDir()
	lid := "998877@lid"
	stamp := time.Now().UTC()
	writeIdentityFixture(t, dir, strings.Repeat("a", 64), core.Decision{
		Action: core.Ignore, MessageID:"u1", InstanceID:"ads",
		ReplyJID: lid, Name: "Nome declarado", Reason: "unresolved_phone_or_non_private_chat",
	}, stamp)
	service := NewService(dir)
	got, err := service.GetEvents(10)
	if err != nil { t.Fatal(err) }
	if len(got.Events) != 1 { t.Fatalf("unexpected events: %d", len(got.Events)) }
	row := got.Events[0]
	if row.Phone != "Número oculto pelo WhatsApp (LID)" || row.Name != "Nome declarado" ||
		row.Status != "telefone_nao_resolvido" || !row.NeedsAttention {
		t.Fatalf("unresolved LID presentation: %+v", row)
	}
	if row.phoneE164 != "" { t.Fatal("LID digits must never be used as phone") }
}

func TestDashboardExactLIDMappingAndAmbiguity(t *testing.T) {
	dir := t.TempDir()
	lid := "12345@lid"
	now := time.Now().UTC()
	phone := "5548999990001"
	writeIdentityFixture(t, dir, strings.Repeat("a",64), core.Decision{
		Action:core.Ignore, InstanceID:"ads", MessageID:"a", ReplyJID:lid,
		Reason:"unresolved_phone_or_non_private_chat",
	}, now)
	writeIdentityFixture(t, dir, strings.Repeat("b",64), core.Decision{
		Action:core.Ignore, InstanceID:"ads", MessageID:"b", ReplyJID:lid,
		PhoneE164:phone, Name:"Nome do WhatsApp",
	}, now.Add(-time.Minute))
	got,err := NewService(dir).GetEvents(10)
	if err != nil {t.Fatal(err)}
	for _,row := range got.Events {
		if row.phoneE164 != phone || row.Phone != "(48) 99999-0001" {
			t.Fatalf("exact LID evidence not applied: %+v", row)
		}
	}
	// Conflicting verified phones for the same LID must fail closed.
	writeIdentityFixture(t, dir, strings.Repeat("c",64), core.Decision{
		Action:core.Ignore, InstanceID:"ads", MessageID:"c", ReplyJID:lid,
		PhoneE164:"5548999990002",
	}, now.Add(-2*time.Minute))
	got,err = NewService(dir).GetEvents(10)
	if err != nil {t.Fatal(err)}
	for _,row := range got.Events {
		if row.EventKey == strings.Repeat("a",64) && row.phoneE164 != "" {
			t.Fatal("conflicting LID mapping assigned incorrect phone")
		}
	}
}

func TestDashboardLIDSenderAltSwapMapsOnlyByExplicitAlias(t *testing.T) {
	dir := t.TempDir()
	lid := "777777@lid"
	phone := "5548999990001"
	now := time.Now().UTC()
	writeIdentityFixture(t, dir, strings.Repeat("e",64), core.Decision{
		Action:core.Ignore, InstanceID:"ads", MessageID:"e",
		ReplyJID:lid, LIDAlias:lid, Reason:"unresolved_phone_or_non_private_chat",
	}, now)
	writeIdentityFixture(t, dir, strings.Repeat("f",64), core.Decision{
		Action:core.Ignore, InstanceID:"ads", MessageID:"f",
		ReplyJID:phone+"@s.whatsapp.net", LIDAlias:lid,
		PhoneE164:phone, Name:"Contato declarado",
	}, now.Add(-time.Minute))
	got,err := NewService(dir).GetEvents(10)
	if err != nil {t.Fatal(err)}
	for _,row := range got.Events {
		if row.phoneE164!=phone {t.Fatalf("explicit senderAlt pair not applied: %+v",row)}
	}
}

func TestDashboardCRMNameCacheIsBoundToVerifiedPhone(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir,"contacts"),0700); err != nil {t.Fatal(err)}
	if err := os.Mkdir(filepath.Join(dir,"tasks"),0700); err != nil {t.Fatal(err)}
	contact := outbox.ContactState{PhoneE164:"5548999990001", UpdatedAt:time.Now().UTC()}
	blob,err := json.Marshal(contact)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir,"contacts","example.json"),blob,0600); err != nil {t.Fatal(err)}
	cache := map[string]interface{}{
		"generated_at":time.Now().UTC().Format(time.RFC3339Nano),
		"names":map[string]string{"5548999990001":"Nome confirmado no CRM"},
	}
	blob,err = json.Marshal(cache)
	if err != nil {t.Fatal(err)}
	if err := os.WriteFile(filepath.Join(dir,"crm-names-cache.json"),blob,0600); err != nil {t.Fatal(err)}
	writeIdentityFixture(t, dir, strings.Repeat("d",64), core.Decision{
		Action:core.Ignore, InstanceID:"ads", MessageID:"d",
		PhoneE164:"5548999990001", ReplyJID:"5548999990001@s.whatsapp.net",
		Name:"Apelido WhatsApp",
	}, time.Now().UTC())
	events,err := NewService(dir).GetEvents(10)
	if err != nil { t.Fatal(err) }
	if len(events.Events)!=1 || events.Events[0].Name!="Nome confirmado no CRM" ||
		events.Events[0].NameOrigin!="crm" {
		t.Fatalf("CRM cache must win on event display, got %+v",events.Events)
	}
	summary,err := NewService(dir).GetSummary()
	if err != nil {t.Fatal(err)}
	if len(summary.Contacts)!=1 || summary.Contacts[0].Name!="Nome confirmado no CRM" {
		t.Fatalf("unique CRM phone must prevail: %+v",summary.Contacts)
	}
}
