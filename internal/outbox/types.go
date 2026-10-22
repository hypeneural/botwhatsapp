package outbox

import (
	"context"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/crm"
	"github.com/hypeneural/botwhatsapp/internal/workflow"
)

type TaskState string

const (
	TaskPending        TaskState = "pending"
	TaskAttemptStarted TaskState = "attempt_started"
	TaskConfirmed      TaskState = "confirmed"
	TaskUncertain      TaskState = "uncertain"
	TaskSuppressed     TaskState = "suppressed"
)

type Task struct {
	ID               string        `json:"id"`
	EventKey         string        `json:"event_key"`
	LeadKey          string        `json:"lead_key,omitempty"`
	InstanceID       string        `json:"instance_id"`
	RecipientJID     string        `json:"recipient_jid"`
	PhoneE164        string        `json:"phone_e164"`
	CRMPhone         string        `json:"crm_phone,omitempty"`
	ContactName      string        `json:"contact_name,omitempty"`
	Step             workflow.Step `json:"step"`
	State            TaskState     `json:"state"`
	PayloadText      string        `json:"payload_text,omitempty"`
	PayloadCaption   string        `json:"payload_caption,omitempty"`
	ExternalID       string        `json:"external_id,omitempty"`
	Error            string        `json:"error,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	AttemptStartedAt time.Time     `json:"attempt_started_at,omitempty"`
	CompletedAt      time.Time     `json:"completed_at,omitempty"`
}

type ContactState struct {
	PhoneE164      string    `json:"phone_e164"`
	LastGreetingAt time.Time `json:"last_greeting_at,omitempty"`
	HumanPaused    bool      `json:"human_paused"`
	HumanPausedAt  time.Time `json:"human_paused_at,omitempty"`
	OptOut         bool      `json:"opt_out"`
	OptOutAt       time.Time `json:"opt_out_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Config struct {
	DataDir                  string
	GreetingCooldown         time.Duration
	CRMInstanceID            *int
	OperationalAuthorization bool
	GreetingCaption          string
	AdWelcomeText            string
	AdAudioFixture           []byte
	GreetingImageFixture     []byte
}

type EvolutionSender interface {
	SendText(ctx context.Context, phone, text string) (string, error)
	SendMedia(ctx context.Context, phone, mediaType string, file []byte, caption string) (string, error)
}

type CRMScheduler interface {
	Schedule(ctx context.Context, req crm.Request) (int, error)
}
