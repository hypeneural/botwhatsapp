package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/outbox"
)

const maxEventRows = 500

// EventStep exposes provider request progress using the outbox's durable
// record. A confirmed Evolution ID is an API acceptance, NOT a delivered/read
// WhatsApp receipt. Unknown network outcomes must never be displayed as sent.
type EventStep struct {
	TaskID       string `json:"-"`
	Step         string `json:"step"`
	Provider     string `json:"provider"`
	State        string `json:"state"`
	APIResponse  string `json:"api_response"`
	ExternalID   string `json:"external_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
	StartedBRT   string `json:"started_brt,omitempty"`
	CompletedBRT string `json:"completed_brt,omitempty"`
}

type EventRow struct {
	EventKey       string      `json:"event_key"`
	ReceivedAt     time.Time   `json:"received_at"`
	ReceivedAtBRT  string      `json:"received_at_brt"`
	Phone          string      `json:"phone"`
	Name           string      `json:"name"`
	Source         string      `json:"source"`
	InboundType    string      `json:"inbound_type"`
	InboundText    string      `json:"inbound_text,omitempty"`
	TriggerMatched bool        `json:"trigger_matched"`
	TriggerText    string      `json:"trigger_text,omitempty"`
	HasAdContext   bool        `json:"has_ad_context"`
	Decision       string      `json:"decision"`
	DecisionReason string      `json:"decision_reason,omitempty"`
	Status         string      `json:"status"`
	NeedsAttention bool        `json:"needs_attention"`
	CRMReceipt     string      `json:"crm_receipt,omitempty"`
	Steps          []EventStep `json:"steps"`
}

type EventStats struct {
	Received     int `json:"received"`
	MetaAds      int `json:"meta_ads"`
	Normal       int `json:"normal"`
	CRMAccepted  int `json:"crm_accepted"`
	CRMRejected  int `json:"crm_rejected"`
	Uncertain    int `json:"uncertain"`
	Attention    int `json:"attention"`
	LocalBlocked int `json:"local_blocked"`
}

// EventResult contains only the most recent N received events, never the full
// spool contents. Stats refer to the returned window, not all-time totals.
type EventResult struct {
	Events []EventRow `json:"events"`
	Stats  EventStats `json:"stats"`
	Limit  int        `json:"limit"`
}

type inboundJournal struct {
	Decision core.Decision `json:"decision"`
	Inbound  struct {
		MessageType  string `json:"message_type"`
		Text         string `json:"text"`
		Source       string `json:"source"`
		HasAdContext bool   `json:"has_ad_context"`
		TriggerMatch bool   `json:"trigger_match"`
		TriggerText  string `json:"trigger_text"`
	} `json:"inbound"`
	EventKey   string `json:"event_key"`
	ReceivedAt string `json:"received_at"`
	Status     string `json:"status"`
}

type eventFile struct {
	path string
	mod  time.Time
}

// GetEvents uses a short 8-second shared snapshot to avoid repeated full
// directory scans when multiple managers open the panel simultaneously.
// Cached results are defensively copied so callers cannot mutate history.
func (s *Service) GetEvents(limit int) (EventResult, error) {
	if limit <= 0 {
		limit = 150
	}
	if limit > maxEventRows {
		limit = maxEventRows
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if !s.eventCacheAt.IsZero() && time.Since(s.eventCacheAt) < 8*time.Second {
		if result, ok := s.eventCache[limit]; ok {
			return copyEvents(result), nil
		}
	}
	if s.eventCache == nil || time.Since(s.eventCacheAt) >= 8*time.Second {
		s.eventCache = make(map[int]EventResult, 3)
		s.eventCacheAt = time.Now()
	}
	result, err := s.getEventsUncached(limit)
	if err != nil {
		return result, err
	}
	s.eventCache[limit] = copyEvents(result)
	return result, nil
}

func copyEvents(in EventResult) EventResult {
	out := in
	out.Events = make([]EventRow, len(in.Events))
	copy(out.Events, in.Events)
	for i := range out.Events {
		out.Events[i].Steps = append([]EventStep(nil), in.Events[i].Steps...)
	}
	return out
}

// getEventsUncached reads the newest ingress records, then joins only task
// files with matching event keys. All calls are read-only.
func (s *Service) getEventsUncached(limit int) (EventResult, error) {
	if limit <= 0 {
		limit = 150
	}
	if limit > maxEventRows {
		limit = maxEventRows
	}
	result := EventResult{Events: make([]EventRow, 0, limit), Limit: limit}
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return result, err
	}
	files := make([]eventFile, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, eventFile{filepath.Join(s.dataDir, entry.Name()), info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	if len(files) > limit {
		files = files[:limit]
	}
	byKey := make(map[string]int, len(files))
	for _, entry := range files {
		raw, err := os.ReadFile(entry.path)
		if err != nil || len(raw) > 1<<20 {
			continue
		}
		var rec inboundJournal
		if err := json.Unmarshal(raw, &rec); err != nil || rec.Decision.MessageID == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, rec.ReceivedAt)
		if err != nil {
			at = entry.mod.UTC()
		}
		key := rec.EventKey
		if key == "" {
			key = strings.TrimSuffix(filepath.Base(entry.path), ".json")
		}
		source := rec.Inbound.Source
		if source != "meta_ads" && source != "normal" {
			source = "normal"
			if rec.Decision.Action == core.AdsProposal {
				source = "meta_ads"
			}
		}
		inboundType := rec.Inbound.MessageType
		if inboundType == "" {
			inboundType = "não registrado (legado)"
		}
		row := EventRow{
			EventKey: key, ReceivedAt: at, ReceivedAtBRT: formatEventBRT(at),
			Phone: FormatPhone(rec.Decision.PhoneE164), Name: rec.Decision.Name,
			Source: source, InboundType: inboundType,
			InboundText: rec.Inbound.Text, TriggerMatched: rec.Inbound.TriggerMatch ||
				rec.Decision.Action == core.AdsProposal,
			TriggerText: rec.Inbound.TriggerText, HasAdContext: rec.Inbound.HasAdContext,
			Decision: string(rec.Decision.Action), DecisionReason: rec.Decision.Reason,
			Status: "recebido", Steps: make([]EventStep, 0),
		}
		if row.Phone == "" {
			row.Phone = "Telefone não resolvido"
		}
		if row.Name == "" {
			row.Name = "Contato WhatsApp"
		}
		if row.Decision == string(core.Ignore) {
			row.Status = "ignorado"
			if row.DecisionReason == "opt_out_requires_persistent_suppression" {
				row.Status = "opt_out"
			}
		}
		byKey[key] = len(result.Events)
		result.Events = append(result.Events, row)
	}
	// A single scan of task FILENAMES, with JSON reads only for matching events.
	tasks, err := os.ReadDir(filepath.Join(s.dataDir, "tasks"))
	if err != nil && !os.IsNotExist(err) {
		return result, err
	}
	for _, entry := range tasks {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := entry.Name()
		if len(name) < 66 {
			continue
		}
		idx, ok := byKey[name[:64]]
		if !ok || name[64] != '-' {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.dataDir, "tasks", name))
		if err != nil || len(raw) > 1<<20 {
			continue
		}
		var t outbox.Task
		if json.Unmarshal(raw, &t) != nil || t.EventKey != result.Events[idx].EventKey {
			continue
		}
		step := EventStep{
			TaskID: t.ID,
			Step:   FriendlyStep(string(t.Step)), Provider: "Evolution Go",
			State: string(t.State), APIResponse: outcome(t),
			ExternalID: t.ExternalID, Reason: safeEventReason(t.Error),
			StartedBRT:   formatEventBRT(t.AttemptStartedAt),
			CompletedBRT: formatEventBRT(t.CompletedAt),
		}
		if t.Step == "schedule_natal_crm" {
			step.Provider = "Evydência CRM"
			if t.State == outbox.TaskConfirmed {
				result.Events[idx].CRMReceipt = t.ExternalID
			}
		} else if t.Step == "pause_for_human" {
			step.Provider = "Bot local"
		}
		result.Events[idx].Steps = append(result.Events[idx].Steps, step)
	}
	for i := range result.Events {
		row := &result.Events[i]
		sort.Slice(row.Steps, func(a, b int) bool {
			return row.Steps[a].TaskID < row.Steps[b].TaskID
		})
		if len(row.Steps) > 0 {
			row.Status = "em_processamento"
		} else if row.Decision != string(core.Ignore) {
			// Ingress was journaled, but the outbox has no durable stage.
			// This may be a suppressed greeting, a preview or a historical gap.
			row.Status = "sem_tarefas"
		}
		uncertain, rejected, localBlocked, pending, crmStep := false, false, false, false, false
		for _, step := range row.Steps {
			if step.Provider == "Evydência CRM" {
				crmStep = true
				if step.State == string(outbox.TaskConfirmed) && step.ExternalID == "" {
					uncertain = true // No receipt: never claim success.
				}
				if strings.HasPrefix(step.Reason, "HTTP 422:") ||
					strings.HasPrefix(step.Reason, "HTTP 401:") ||
					strings.HasPrefix(step.Reason, "HTTP 403:") ||
					strings.HasPrefix(step.Reason, "HTTP 400:") ||
					strings.HasPrefix(step.Reason, "HTTP 404:") {
					rejected = true
				} else if step.State == string(outbox.TaskSuppressed) {
					// A local guard may suppress a task BEFORE calling the CRM.
					localBlocked = true
				}
			}
			if step.State == string(outbox.TaskUncertain) || step.State == string(outbox.TaskAttemptStarted) {
				uncertain = true
			}
			if step.State == string(outbox.TaskPending) {
				pending = true
			}
		}
		switch {
		case row.CRMReceipt != "":
			row.Status = "crm_aceito"
		case rejected:
			row.Status = "crm_rejeitado"
		case localBlocked:
			row.Status = "bloqueado_local"
		case uncertain:
			row.Status = "incerto"
		case pending:
			row.Status = "pendente"
		case len(row.Steps) > 0 && row.Decision == string(core.AdsProposal) && !crmStep:
			row.Status = "crm_nao_registrado"
		case len(row.Steps) > 0:
			row.Status = "finalizado_local"
		}
		// One-time, grounded triage signal. Reception by itself is not a failure;
		// allow five minutes for the worker to commit the remaining stages.
		age := time.Since(row.ReceivedAt)
		if row.Status == "crm_rejeitado" || row.Status == "incerto" ||
			row.Status == "bloqueado_local" ||
			(age > 5*time.Minute && row.Decision == string(core.AdsProposal) &&
				(row.Status == "sem_tarefas" || row.Status == "crm_nao_registrado" ||
					row.Status == "pendente" || row.Status == "em_processamento")) {
			row.NeedsAttention = true
			result.Stats.Attention++
		}
		if row.Status == "bloqueado_local" {
			result.Stats.LocalBlocked++
		}
		result.Stats.Received++
		if row.Source == "meta_ads" {
			result.Stats.MetaAds++
		} else {
			result.Stats.Normal++
		}
		if row.Status == "crm_aceito" {
			result.Stats.CRMAccepted++
		}
		if row.Status == "crm_rejeitado" {
			result.Stats.CRMRejected++
		}
		if row.Status == "incerto" {
			result.Stats.Uncertain++
		}
	}
	// Journal file mtime can change during backup/restore. Display by the
	// actual persisted reception timestamp, with a deterministic tie-breaker.
	sort.Slice(result.Events, func(i, j int) bool {
		if result.Events[i].ReceivedAt.Equal(result.Events[j].ReceivedAt) {
			return result.Events[i].EventKey > result.Events[j].EventKey
		}
		return result.Events[i].ReceivedAt.After(result.Events[j].ReceivedAt)
	})
	return result, nil
}

func formatEventBRT(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(brtLocation).Format("02/01/2006 15:04:05")
}

func outcome(t outbox.Task) string {
	switch t.State {
	case outbox.TaskConfirmed:
		return "API aceitou (entrega não comprovada)"
	case outbox.TaskUncertain, outbox.TaskAttemptStarted:
		if strings.Contains(strings.ToLower(t.Error), "http 422") {
			return "API recusou (HTTP 422 legado)"
		}
		return "Sem confirmação conclusiva"
	case outbox.TaskSuppressed:
		if strings.Contains(strings.ToLower(t.Error), "http ") {
			return "API recusou"
		}
		return "Não executada / bloqueada"
	case outbox.TaskPending:
		return "Ainda não executada"
	default:
		return "Sem informação"
	}
}

func safeEventReason(reason string) string {
	// Errors from remote providers can contain untrusted content or credentials:
	// only expose recognized operational classes on the dashboard.
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "http 400"):
		return "HTTP 400: requisição recusada"
	case strings.Contains(lower, "http 404"):
		return "HTTP 404: rota indisponível"
	case strings.Contains(lower, "http 422"):
		return "HTTP 422: rejeição do contrato/regra comercial"
	case strings.Contains(lower, "http 401"):
		return "HTTP 401: autorização recusada"
	case strings.Contains(lower, "http 403"):
		return "HTTP 403: permissão insuficiente"
	case strings.Contains(lower, "http 429"):
		return "HTTP 429: limite de requisições"
	case strings.Contains(lower, "cooldown"), strings.Contains(lower, "already claimed"), strings.Contains(lower, "duplicate"):
		return "Cooldown / duplicidade"
	case strings.Contains(lower, "suppressed"), strings.Contains(lower, "opt-out"), strings.Contains(lower, "human"):
		return "Supressão ou atendimento humano"
	case strings.Contains(lower, "invalid"), strings.Contains(lower, "missing"):
		return "Configuração ou destinatário inválido"
	case reason == "":
		return ""
	default:
		return "Falha ou resultado não confirmado; consultar log protegido"
	}
}
