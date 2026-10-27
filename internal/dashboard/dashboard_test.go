package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/outbox"
	"github.com/hypeneural/botwhatsapp/internal/workflow"
)

func TestFormatPhone(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"5548996553954", "(48) 99655-3954"},
		{"554832200440", "(48) 3220-0440"},
		{"+5548996425287", "(48) 99642-5287"},
		{"12345", "12345"},
	}

	for _, tc := range cases {
		got := FormatPhone(tc.input)
		if got != tc.want {
			t.Errorf("FormatPhone(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFriendlyStep(t *testing.T) {
	if FriendlyStep("send_ad_reaction") != "Reação Natalina 🎄" {
		t.Errorf("unexpected step description")
	}
	if FriendlyStep("schedule_natal_crm") != "Proposta Natal (CRM)" {
		t.Errorf("unexpected step description")
	}
	if FriendlyStep("send_greeting_text_menu") != "Menu de Opções 1–5" {
		t.Errorf("unexpected step description")
	}
}

func TestDashboardServiceAndHandler(t *testing.T) {
	tmpDir := t.TempDir()
	contactsDir := filepath.Join(tmpDir, "contacts")
	tasksDir := filepath.Join(tmpDir, "tasks")
	if err := os.MkdirAll(contactsDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tasksDir, 0700); err != nil {
		t.Fatal(err)
	}

	// 1. Write an Ads Contact with CRM schedule
	now := time.Now().UTC()
	adsContact := outbox.ContactState{
		PhoneE164:     "5548996553954",
		LastAdsFlowAt: now.Add(-1 * time.Hour), // Cooldown active
		UpdatedAt:     now.Add(-1 * time.Hour),
	}
	rawContact, _ := json.Marshal(adsContact)
	if err := os.WriteFile(filepath.Join(contactsDir, "contact1.json"), rawContact, 0600); err != nil {
		t.Fatal(err)
	}

	// Ads Task: Reaction
	task1 := outbox.Task{
		ID:          "task-1",
		PhoneE164:   "5548996553954",
		ContactName: "Anderson Teste",
		Step:        workflow.Step("send_ad_reaction"),
		State:       outbox.TaskConfirmed,
		CreatedAt:   now.Add(-1 * time.Hour),
	}
	rawTask1, _ := json.Marshal(task1)
	_ = os.WriteFile(filepath.Join(tasksDir, "task-1.json"), rawTask1, 0600)

	// Ads Task: CRM Schedule
	task2 := outbox.Task{
		ID:          "task-2",
		PhoneE164:   "5548996553954",
		ContactName: "Anderson Teste",
		Step:        workflow.Step("schedule_natal_crm"),
		State:       outbox.TaskConfirmed,
		ExternalID:  "crm-schedule-28547",
		CreatedAt:   now.Add(-59 * time.Minute),
	}
	rawTask2, _ := json.Marshal(task2)
	_ = os.WriteFile(filepath.Join(tasksDir, "task-2.json"), rawTask2, 0600)

	// 2. Write an Organic Contact
	orgContact := outbox.ContactState{
		PhoneE164:      "554898307398",
		LastGreetingAt: now.Add(-2 * time.Hour),
		UpdatedAt:      now.Add(-2 * time.Hour),
	}
	rawOrg, _ := json.Marshal(orgContact)
	_ = os.WriteFile(filepath.Join(contactsDir, "contact2.json"), rawOrg, 0600)

	orgTask := outbox.Task{
		ID:          "task-3",
		PhoneE164:   "554898307398",
		ContactName: "Stéfany Ferreira",
		Step:        workflow.Step("send_greeting_text_menu"),
		State:       outbox.TaskConfirmed,
		CreatedAt:   now.Add(-2 * time.Hour),
	}
	rawOrgTask, _ := json.Marshal(orgTask)
	_ = os.WriteFile(filepath.Join(tasksDir, "task-3.json"), rawOrgTask, 0600)

	// Test Service
	svc := NewService(tmpDir)
	summary, err := svc.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}

	if summary.Stats.TotalContacts != 2 {
		t.Errorf("TotalContacts = %d, want 2", summary.Stats.TotalContacts)
	}
	if summary.Stats.AdsContacts != 1 {
		t.Errorf("AdsContacts = %d, want 1", summary.Stats.AdsContacts)
	}
	if summary.Stats.OrganicContacts != 1 {
		t.Errorf("OrganicContacts = %d, want 1", summary.Stats.OrganicContacts)
	}
	if summary.Stats.CRMScheduled != 1 {
		t.Errorf("CRMScheduled = %d, want 1", summary.Stats.CRMScheduled)
	}
	if summary.Stats.ActiveCooldown != 1 {
		t.Errorf("ActiveCooldown = %d, want 1", summary.Stats.ActiveCooldown)
	}

	// Verify Contact 1 (Ads)
	rowAds := summary.Contacts[0] // Most recent
	if !rowAds.IsAds || rowAds.Channel != "Meta Ads" {
		t.Errorf("expected row to be Meta Ads")
	}
	if rowAds.CRMScheduleID != "crm-schedule-28547" {
		t.Errorf("expected CRM schedule ID 28547, got %s", rowAds.CRMScheduleID)
	}
	if !rowAds.CooldownActive {
		t.Errorf("expected cooldown active on Ads contact")
	}

	// Test HTTP Handler
	handler := NewHandler(tmpDir, "secret-token-123")

	// 1. Unauthorized request
	reqUnauth := httptest.NewRequest(http.MethodGet, "/painel", nil)
	wUnauth := httptest.NewRecorder()
	handler.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized without token, got %d", wUnauth.Code)
	}

	// 2. Authorized HTML request
	reqAuth := httptest.NewRequest(http.MethodGet, "/painel", nil)
	reqAuth.SetBasicAuth("operador", "secret-token-123")
	wAuth := httptest.NewRecorder()
	handler.ServeHTTP(wAuth, reqAuth)
	if wAuth.Code != http.StatusOK {
		t.Errorf("expected 200 OK with token, got %d", wAuth.Code)
	}
	if wAuth.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type")
	}

	// 3. Authorized JSON API request
	reqJSON := httptest.NewRequest(http.MethodGet, "/api/dashboard/contacts", nil)
	reqJSON.SetBasicAuth("operador", "secret-token-123")
	wJSON := httptest.NewRecorder()
	handler.ServeHTTP(wJSON, reqJSON)
	if wJSON.Code != http.StatusOK {
		t.Errorf("expected 200 OK for API, got %d", wJSON.Code)
	}

	var jsonSummary Summary
	if err := json.NewDecoder(wJSON.Body).Decode(&jsonSummary); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if jsonSummary.Stats.TotalContacts != 2 {
		t.Errorf("JSON API TotalContacts = %d, want 2", jsonSummary.Stats.TotalContacts)
	}
}

func TestManagerPanelResponsiveAndReadOnly(t *testing.T) {
	h := NewHandler(t.TempDir(), "fake-preview-only")
	req := httptest.NewRequest(http.MethodGet, "/painel", nil)
	req.SetBasicAuth("gestor", "fake-preview-only")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Fatalf("unexpected status %d", w.Code) }
	html := w.Body.String()
	required := []string{
		"Monitor de mensagens", "Histórico de atendimentos", "Precisam de atenção",
		"@media(max-width:700px)", "id=\"eventRows\"", "id=\"filterSource\"",
		"id=\"filterStatus\"", "id=\"contactsDetails\"",
		"/api/dashboard/events", "escapeHTML", // replaced below by safe()
	}
	for _, token := range required {
		if token == "escapeHTML" {token="function safe(value)"}
		if !strings.Contains(html,token) {t.Errorf("manager page missing %q",token)}
	}
	for _, blocked := range []string{"<script src=", "https://cdn.", "fetch('/api/campaigns/schedule", "?token="} {
		if strings.Contains(html,blocked) {t.Errorf("unexpected external/write source %q",blocked)}
	}
	if w.Header().Get("X-Frame-Options")!="DENY" || !strings.Contains(w.Header().Get("Cache-Control"),"no-store") {
		t.Fatal("sensitive management dashboard headers missing")
	}
}
