package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/outbox"
)

var brtLocation *time.Location

func init() {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	brtLocation = loc
}

// Service loads and aggregates contact and task information.
type Service struct {
	dataDir      string
	cacheMu      sync.Mutex
	eventCacheAt time.Time
	eventCache   map[int]EventResult
}

// NewService creates a new dashboard data service.
func NewService(dataDir string) *Service {
	return &Service{dataDir: dataDir}
}

// FormatPhone converts a raw E164 Brazilian number to a readable format.
// e.g., 5548996553954 -> (48) 99655-3954, 554832200440 -> (48) 3220-0440
func FormatPhone(raw string) string {
	digits := strings.TrimPrefix(raw, "+")
	if strings.HasPrefix(digits, "55") {
		digits = strings.TrimPrefix(digits, "55")
	}
	if len(digits) == 11 {
		return fmt.Sprintf("(%s) %s-%s", digits[:2], digits[2:7], digits[7:])
	}
	if len(digits) == 10 {
		return fmt.Sprintf("(%s) %s-%s", digits[:2], digits[2:6], digits[6:])
	}
	return raw
}

// FriendlyStep returns a human-readable description for an outbox step.
func FriendlyStep(step string) string {
	switch step {
	case "send_ad_reaction":
		return "Reação Natalina 🎄"
	case "send_ad_text":
		return "Boas-vindas (Ads)"
	case "send_ad_audio":
		return "Áudio PTT Oficial"
	case "send_ad_document":
		return "PDF aceito pela Evolution"
	case "send_ad_closing_text":
		return "Texto de Fechamento"
	case "schedule_natal_crm":
		return "Proposta Natal (CRM)"
	case "send_greeting_text_menu":
		return "Menu de Opções 1–5"
	case "send_menu_reply":
		return "Resposta do Menu"
	case "send_greeting":
		return "Saudação Enviada"
	default:
		return step
	}
}

// GetSummary inspects contacts and tasks in dataDir to produce an aggregated summary.
func (s *Service) GetSummary() (Summary, error) {
	now := time.Now().UTC()
	contactsDir := filepath.Join(s.dataDir, "contacts")
	tasksDir := filepath.Join(s.dataDir, "tasks")

	// 1. Read all tasks and index by phone
	tasksByPhone := make(map[string][]outbox.Task)
	if entries, err := os.ReadDir(tasksDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			path := filepath.Join(tasksDir, e.Name())
			if raw, err := os.ReadFile(path); err == nil {
				var t outbox.Task
				if err := json.Unmarshal(raw, &t); err == nil && t.PhoneE164 != "" {
					tasksByPhone[t.PhoneE164] = append(tasksByPhone[t.PhoneE164], t)
				}
			}
		}
	}

	// 2. Read all contacts
	var rows []ContactRow
	seenPhones := make(map[string]bool)

	if entries, err := os.ReadDir(contactsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			path := filepath.Join(contactsDir, e.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var c outbox.ContactState
			if err := json.Unmarshal(raw, &c); err != nil || c.PhoneE164 == "" {
				continue
			}
			seenPhones[c.PhoneE164] = true
			row := s.buildRow(c, tasksByPhone[c.PhoneE164], now)
			rows = append(rows, row)
		}
	}

	// 3. Check for any phones in tasks not present in contacts
	for phone, tasks := range tasksByPhone {
		if !seenPhones[phone] && len(tasks) > 0 {
			var latest time.Time
			for _, t := range tasks {
				if t.CreatedAt.After(latest) {
					latest = t.CreatedAt
				}
			}
			c := outbox.ContactState{
				PhoneE164: phone,
				UpdatedAt: latest,
			}
			row := s.buildRow(c, tasks, now)
			rows = append(rows, row)
		}
	}

	// 4. Sort rows by LastInteraction descending (most recent first)
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].LastInteraction.After(rows[j].LastInteraction)
	})

	// 5. Compute aggregate stats
	stats := Stats{
		TotalContacts:  len(rows),
		GeneratedAt:    now,
		GeneratedAtBRT: now.In(brtLocation).Format("02/01/2006 15:04:05"),
	}

	for _, r := range rows {
		if r.IsAds {
			stats.AdsContacts++
		} else {
			stats.OrganicContacts++
		}
		if r.CRMScheduleID != "" {
			stats.CRMScheduled++
		}
		if r.CooldownActive {
			stats.ActiveCooldown++
		}
		if r.OptOut {
			stats.OptOutCount++
		}
		if r.HumanPaused {
			stats.PausedCount++
		}
	}

	return Summary{
		Stats:    stats,
		Contacts: rows,
	}, nil
}

func (s *Service) buildRow(c outbox.ContactState, tasks []outbox.Task, now time.Time) ContactRow {
	row := ContactRow{
		PhoneRaw:       c.PhoneE164,
		PhoneFormatted: FormatPhone(c.PhoneE164),
		HumanPaused:    c.HumanPaused,
		OptOut:         c.OptOut,
		TotalTasks:     len(tasks),
		CRMStatus:      "—",
	}

	// Sort tasks by CreatedAt ascending
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	// Check if this contact participated in Meta Ads
	isAds := !c.LastAdsFlowAt.IsZero()
	var lastStep string
	var contactName string
	var crmScheduleID string
	var latestTaskTime time.Time

	for _, t := range tasks {
		if t.ContactName != "" {
			contactName = t.ContactName
		}
		if t.CreatedAt.After(latestTaskTime) {
			latestTaskTime = t.CreatedAt
		}
		lastStep = string(t.Step)

		// Check for Ads step
		if strings.HasPrefix(string(t.Step), "send_ad_") || t.Step == "schedule_natal_crm" {
			isAds = true
		}

		// Check CRM status
		if t.Step == "schedule_natal_crm" {
			if t.State == outbox.TaskConfirmed && t.ExternalID != "" {
				crmScheduleID = t.ExternalID
				row.CRMStatus = fmt.Sprintf("Aceito no CRM (%s)", t.ExternalID)
			} else if t.State == outbox.TaskUncertain {
				row.CRMStatus = "Incerto / revisar"
			} else if t.State == outbox.TaskSuppressed {
				row.CRMStatus = "Rejeitado / não agendado"
			} else if t.State == outbox.TaskAttemptStarted {
				row.CRMStatus = "Em execução"
			} else if t.State == outbox.TaskPending {
				row.CRMStatus = "Pendente"
			}
		}
	}

	row.IsAds = isAds
	if isAds {
		row.Channel = "Meta Ads"
	} else {
		row.Channel = "Sem marcação de anúncio"
	}

	if contactName == "" {
		contactName = "Cliente WhatsApp"
	}
	row.Name = contactName
	row.CRMScheduleID = crmScheduleID

	if lastStep != "" {
		row.LastStep = FriendlyStep(lastStep)
	} else if isAds {
		row.LastStep = "Funil de Anúncio"
	} else {
		row.LastStep = "Saudação / Atendimento"
	}

	// Determine latest interaction
	lastTime := c.UpdatedAt
	if c.LastGreetingAt.After(lastTime) {
		lastTime = c.LastGreetingAt
	}
	if c.LastAdsFlowAt.After(lastTime) {
		lastTime = c.LastAdsFlowAt
	}
	if latestTaskTime.After(lastTime) {
		lastTime = latestTaskTime
	}
	if lastTime.IsZero() {
		lastTime = now
	}

	row.LastInteraction = lastTime
	row.LastInteractionBRT = lastTime.In(brtLocation).Format("02/01/2006 15:04:05")

	// Cooldown calculation (24h standard)
	if !c.LastAdsFlowAt.IsZero() {
		expiresAt := c.LastAdsFlowAt.Add(24 * time.Hour)
		if now.Before(expiresAt) {
			row.CooldownActive = true
			rem := expiresAt.Sub(now).Round(time.Minute)
			hours := int(rem.Hours())
			minutes := int(rem.Minutes()) % 60
			row.CooldownRemaining = fmt.Sprintf("%dh %02dm", hours, minutes)
		}
	}

	return row
}
