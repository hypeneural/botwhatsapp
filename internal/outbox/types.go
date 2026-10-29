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
	ID                string        `json:"id"`
	EventKey          string        `json:"event_key"`
	LeadKey           string        `json:"lead_key,omitempty"`
	InstanceID        string        `json:"instance_id"`
	RecipientJID      string        `json:"recipient_jid"`
	InboundMessageID  string        `json:"inbound_message_id,omitempty"`
	PhoneE164         string        `json:"phone_e164"`
	CRMPhone          string        `json:"crm_phone,omitempty"`
	ContactName       string        `json:"contact_name,omitempty"`
	PreviousTaskID    string        `json:"previous_task_id,omitempty"`
	Step              workflow.Step `json:"step"`
	AllowOnHumanPause bool          `json:"allow_on_human_pause,omitempty"`
	State             TaskState     `json:"state"`
	PayloadText       string        `json:"payload_text,omitempty"`
	PayloadCaption    string        `json:"payload_caption,omitempty"`
	ExternalID        string        `json:"external_id,omitempty"`
	Error             string        `json:"error,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	AttemptStartedAt  time.Time     `json:"attempt_started_at,omitempty"`
	CompletedAt       time.Time     `json:"completed_at,omitempty"`
}

type ContactState struct {
	PhoneE164      string    `json:"phone_e164"`
	LastGreetingAt time.Time `json:"last_greeting_at,omitempty"`
	LastAdsFlowAt  time.Time `json:"last_ads_flow_at,omitempty"`
	LastInboundAt  time.Time `json:"last_inbound_at,omitempty"`
	LastOutboundAt time.Time `json:"last_outbound_at,omitempty"`
	LastActivityAt time.Time `json:"last_activity_at,omitempty"`
	HumanPaused    bool      `json:"human_paused"`
	HumanPausedAt  time.Time `json:"human_paused_at,omitempty"`
	OptOut         bool      `json:"opt_out"`
	OptOutAt       time.Time `json:"opt_out_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Config struct {
	DataDir                  string
	GreetingCooldown         time.Duration
	AdsFlowCooldown          time.Duration
	CRMInstanceID            *int
	OperationalAuthorization bool
	NativeProposalScheduler  NativeProposalScheduler
	GreetingCaption          string
	AdReactionEmoji          string
	AdWelcomeText            string
	AdAudioFixture           []byte
	AdDocumentFixture        []byte
	AdDocumentFilename       string
	AdDocumentURL            string
	AdDocumentCaption        string
	AdClosingText            string
	AdImageFixture           []byte
	AdVideoFixture           []byte
	AdImageCaption           string
	AdVideoCaption           string
	GreetingImageFixture     []byte
	// When set, never fall back to a shared token/sender for another UUID.
	Senders       map[string]EvolutionSender
	Tenant        string
	Location      *time.Location
	TaskRetention time.Duration
	MaxTasksCount int
}

type EvolutionSender interface {
	SendText(ctx context.Context, phone, text string) (string, error)
	SendMedia(ctx context.Context, phone, mediaType string, file []byte, caption string) (string, error)
	SendDocument(ctx context.Context, phone string, file []byte, filename string) (string, error)
	SendDocumentURL(ctx context.Context, phone, documentURL, filename, caption string) (string, error)
	SendReaction(ctx context.Context, phone, messageID, reaction string) (string, error)
}

type NativeProposalScheduler interface {
	ScheduleNative(ctx context.Context, nationalPhone, requestID string) (string, error)
}

type CRMScheduler interface {
	Schedule(ctx context.Context, req crm.Request) (int, error)
}
