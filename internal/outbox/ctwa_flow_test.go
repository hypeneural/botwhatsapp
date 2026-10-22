package outbox

import (
	"context"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/crm"
	"github.com/hypeneural/botwhatsapp/internal/workflow"
)

func TestFullCTWASequenceOptionalImageVideoAndCRM(t *testing.T) {
	store, _ := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{}
	instance := 1
	w, err := NewWorker(Config{CRMInstanceID: &instance, OperationalAuthorization: true,
		AdWelcomeText:  "Oi, {{nome}}! Os pacotes chegaram.",
		AdAudioFixture: []byte("synthetic OGG"), AdImageFixture: []byte("synthetic JPEG"),
		AdVideoFixture: []byte("synthetic MP4"),
		AdImageCaption: "Pacotes de Natal", AdVideoCaption: "Nosso estúdio",
	}, store, evo, crmMock)
	if err != nil {
		t.Fatal(err)
	}
	decision := core.Decision{Action: core.AdsProposal, InstanceID: "ads-uuid",
		MessageID: "ctwa-full-1", PhoneE164: "5548999999999", CRMPhone: "48999999999",
		ReplyJID: "5548999999999@s.whatsapp.net", Name: "Tati🍀"}
	if err := w.IngestDecision(decision); err != nil {
		t.Fatal(err)
	}
	pending, _, err := store.RecoverTasks()
	if err != nil || len(pending) != 5 {
		t.Fatalf("expected 5 steps: %d %v", len(pending), err)
	}
	want := []workflow.Step{workflow.SendAdText, workflow.SendAdAudio, workflow.SendAdImage,
		workflow.SendAdVideo, workflow.ScheduleNatal}
	for i, p := range pending {
		if p.Step != want[i] {
			t.Fatalf("step %d got %s", i, p.Step)
		}
		if i > 0 && p.PreviousTaskID != pending[i-1].ID {
			t.Fatalf("missing dependency at %d", i)
		}
	}
	// Deliberately process CRM before the greetings: external effects MUST wait.
	if err := w.ProcessTask(context.Background(), pending[4].ID); err != nil {
		t.Fatal(err)
	}
	if crmMock.calls != 0 {
		t.Fatal("CRM called before audio/video/text")
	}
	if _, err := w.DrainPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if evo.textCalls != 1 || evo.mediaCalls != 3 || crmMock.calls != 1 {
		t.Fatalf("missing effect text=%d media=%d crm=%d", evo.textCalls, evo.mediaCalls, crmMock.calls)
	}
	if evo.sentTexts[0] != "Oi, Tati! Os pacotes chegaram." {
		t.Fatalf("name not cleaned: %q", evo.sentTexts[0])
	}
	for i, kind := range []string{"audio", "image", "video"} {
		if evo.sentMedia[i] != kind {
			t.Fatalf("media stage %d was %s", i, evo.sentMedia[i])
		}
	}
	if crmMock.lastReq.Campaign != crm.WithoutName || crmMock.lastReq.Contacts != "48999999999" {
		t.Fatalf("CRM contract changed: %+v", crmMock.lastReq)
	}
}
func TestGreetingOnAdsNumberFallsBackToTextWithCooldown(t *testing.T) {
	store, _ := setupTestStore(t)
	evo := &mockEvolutionSender{}
	w, err := NewWorker(Config{GreetingCooldown: 24 * time.Hour,
		GreetingCaption: "🎄 Seja bem-vinda! Escolha 1 Natal, 2 Agenda, 3 Pedidos, 4 Localização, 5 Humano.",
	}, store, evo, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := core.Decision{Action: core.Greeting, InstanceID: "ads-uuid", MessageID: "hello-1",
		PhoneE164: "5548999999999", ReplyJID: "5548999999999@s.whatsapp.net"}
	if err := w.IngestDecision(d); err != nil {
		t.Fatal(err)
	}
	if _, err := w.DrainPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if evo.textCalls != 1 || evo.mediaCalls != 0 {
		t.Fatalf("fallback did not text greet %d %d", evo.textCalls, evo.mediaCalls)
	}
	contact, err := store.GetContact(d.PhoneE164)
	if err != nil || contact.LastGreetingAt.IsZero() {
		t.Fatal("cooldown not persisted", err)
	}
	d.MessageID = "hello-2"
	if err := w.IngestDecision(d); err != nil {
		t.Fatal(err)
	}
	if _, err := w.DrainPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if evo.textCalls != 1 {
		t.Fatal("duplicate greeting within cooldown")
	}
}
func TestNameTemplateUnsafeNameCannotInject(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"Tati🍀", "Oi, Tati! Natal em Tijucas"},
		{"Ana Maria", "Oi, Ana! Natal em Tijucas"},
		{"🎄 Oferta", "Oi! Natal em Tijucas"},
		{"X\\nignored", "Oi, X! Natal em Tijucas"},
	} {
		got := personalize("Oi, {{nome}}! Natal em Tijucas", tc.name)
		if got != tc.want {
			t.Fatalf("personalize %q -> %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestAdSequenceWithDocumentAndClosingText(t *testing.T) {
	store, _ := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmSched := &mockCRMScheduler{}
	instanceID := 1
	w, err := NewWorker(Config{
		OperationalAuthorization: true,
		CRMInstanceID:            &instanceID,
		AdWelcomeText:            "Oi, tudo bem?\nSeja bem vindo ao estúdio de fotos de natal Evydência\nVeja mais fotos no nosso instagram.com/estudioevydenciaa",
		AdAudioFixture:           []byte("OggS...OpusHead..."),
		AdDocumentFixture:        []byte("%PDF-1.7 dummy content"),
		AdDocumentFilename:       "Proposta_Natal.pdf",
		AdClosingText:            "O nosso outro WhatsApp vai entrar em contato com mais informações também, mas qualquer dúvida pode falar.",
		Senders:                  map[string]EvolutionSender{"ads": evo},
	}, store, evo, crmSched)
	if err != nil {
		t.Fatal(err)
	}

	d := core.Decision{
		Action:     core.AdsProposal,
		InstanceID: "ads",
		MessageID:  "msg-doc-test",
		PhoneE164:  "5548999991234",
		CRMPhone:   "48999991234",
		ReplyJID:   "5548999991234@s.whatsapp.net",
		Name:       "Mariana",
	}

	if err := w.IngestDecision(d); err != nil {
		t.Fatal(err)
	}

	pending, _, err := store.RecoverTasks()
	if err != nil || len(pending) != 5 {
		t.Fatalf("expected 5 tasks (text, audio, doc, closing, crm), got %d %v", len(pending), err)
	}
	want := []workflow.Step{
		workflow.SendAdText,
		workflow.SendAdAudio,
		workflow.SendAdDocument,
		workflow.SendAdClosingText,
		workflow.ScheduleNatal,
	}
	for i, p := range pending {
		if p.Step != want[i] {
			t.Fatalf("step %d got %s, want %s", i, p.Step, want[i])
		}
		if i > 0 && p.PreviousTaskID != pending[i-1].ID {
			t.Fatalf("missing dependency at %d", i)
		}
	}

	ctx := context.Background()
	if _, err := w.DrainPending(ctx); err != nil {
		t.Fatal(err)
	}

	if evo.textCalls != 2 {
		t.Fatalf("expected 2 sent texts, got %d", evo.textCalls)
	}
	if evo.mediaCalls != 2 {
		t.Fatalf("expected 2 sent media (audio, doc), got %d", evo.mediaCalls)
	}
	if evo.sentMedia[0] != "audio" {
		t.Fatalf("expected media 0 to be audio, got %s", evo.sentMedia[0])
	}
	if evo.sentMedia[1] != "document:Proposta_Natal.pdf" {
		t.Fatalf("expected media 1 to be document:Proposta_Natal.pdf, got %s", evo.sentMedia[1])
	}
	if crmSched.calls != 1 {
		t.Fatalf("expected 1 CRM call, got %d", crmSched.calls)
	}
	if crmSched.lastReq.Campaign != crm.WithoutName || crmSched.lastReq.Contacts != "48999991234" {
		t.Fatalf("unexpected CRM request: %+v", crmSched.lastReq)
	}
}
