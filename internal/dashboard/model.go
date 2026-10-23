package dashboard

import (
	"time"
)

// ContactRow represents an aggregated row in the dashboard table.
type ContactRow struct {
	PhoneRaw           string    `json:"phone_raw"`
	PhoneFormatted     string    `json:"phone_formatted"`
	Name               string    `json:"name"`
	Channel            string    `json:"channel"` // "Meta Ads" or "Orgânico"
	IsAds              bool      `json:"is_ads"`
	LastInteraction    time.Time `json:"last_interaction"`
	LastInteractionBRT string    `json:"last_interaction_brt"`
	CRMStatus          string    `json:"crm_status"` // "Agendado (#28547)" or "—"
	CRMScheduleID      string    `json:"crm_schedule_id,omitempty"`
	LastStep           string    `json:"last_step"`
	CooldownActive     bool      `json:"cooldown_active"`
	CooldownRemaining  string    `json:"cooldown_remaining,omitempty"`
	HumanPaused        bool      `json:"human_paused"`
	OptOut             bool      `json:"opt_out"`
	TotalTasks         int       `json:"total_tasks"`
}

// Stats represents overall operational summary metrics.
type Stats struct {
	TotalContacts   int       `json:"total_contacts"`
	AdsContacts     int       `json:"ads_contacts"`
	OrganicContacts int       `json:"organic_contacts"`
	CRMScheduled    int       `json:"crm_scheduled"`
	ActiveCooldown  int       `json:"active_cooldown"`
	OptOutCount     int       `json:"opt_out_count"`
	PausedCount     int       `json:"paused_count"`
	GeneratedAt     time.Time `json:"generated_at"`
	GeneratedAtBRT  string    `json:"generated_at_brt"`
}

// Summary combines metrics and the list of contact rows.
type Summary struct {
	Stats    Stats        `json:"stats"`
	Contacts []ContactRow `json:"contacts"`
}
