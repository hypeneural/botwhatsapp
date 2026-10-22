// Package workflow defines deterministic actions, but deliberately executes none.
// An authorized, durable executor must be added before any production sending.
package workflow

import (
	"strings"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

type Step string

const (
	SendAdReaction    Step = "send_ad_reaction"
	SendAdText        Step = "send_ad_text"
	SendAdAudio       Step = "send_ad_audio"
	SendAdDocument    Step = "send_ad_document"
	SendAdClosingText Step = "send_ad_closing_text"
	SendAdImage       Step = "send_ad_image_caption"
	SendAdVideo       Step = "send_ad_video_caption"
	ScheduleNatal     Step = "schedule_natal_crm"
	SendGreetingImage Step = "send_greeting_image_caption"
	SendGreetingText  Step = "send_greeting_text_menu"
	SendMenuReply     Step = "send_menu_reply"
	PauseHuman        Step = "pause_for_human"
)

type Plan struct {
	Steps   []Step `json:"steps"`
	Message string `json:"message,omitempty"`
}

// Options controls optional visual stages. A missing image/video is NOT a
// dummy asset or a missing CRM prerequisite.
type Options struct {
	AdReaction    bool
	AdDocument    bool
	AdClosingText bool
	AdImage       bool
	AdVideo       bool
	GreetingImage bool
}

func Create(d core.Decision) Plan {
	return CreateWithOptions(d, Options{GreetingImage: true})
}

func CreateWithOptions(d core.Decision, o Options) Plan {
	switch d.Action {
	case core.AdsProposal:
		steps := make([]Step, 0, 8)
		if o.AdReaction {
			steps = append(steps, SendAdReaction)
		}
		steps = append(steps, SendAdText, SendAdAudio)
		if o.AdDocument {
			steps = append(steps, SendAdDocument)
		}
		if o.AdClosingText {
			steps = append(steps, SendAdClosingText)
		}
		if o.AdImage {
			steps = append(steps, SendAdImage)
		}
		if o.AdVideo {
			steps = append(steps, SendAdVideo)
		}
		steps = append(steps, ScheduleNatal)
		return Plan{Steps: steps}
	case core.Greeting:
		if o.GreetingImage {
			return Plan{Steps: []Step{SendGreetingImage}}
		}
		return Plan{Steps: []Step{SendGreetingText}}
	case core.MenuOption:
		if d.Option == "5" {
			return Plan{Steps: []Step{SendMenuReply, PauseHuman}, Message: Reply("5")}
		}
		return Plan{Steps: []Step{SendMenuReply}, Message: Reply(d.Option)}
	}
	return Plan{}
}

func Reply(option string) string {
	switch strings.TrimSpace(option) {
	case "1":
		return "🎄 Veja nossos ensaios de Natal 2026: https://evydencia.com/catalogo/natal"
	case "2":
		return "📅 Confira as opções de horários: https://horarios.fotosdenatal.com/"
	case "3":
		return "📸 Vamos ajudar com suas fotos e pedidos. Nossa equipe continuará o atendimento por aqui."
	case "4":
		return "📍 Nossa localização: https://ir.fotosdenatal.com/"
	case "5":
		return "👩‍💼 Vou deixar seu atendimento com nossa equipe. Assim que possível, responderemos por aqui."
	default:
		return ""
	}
}

// IsGreetingDue returns true when no previous greeting exists, or the
// cooldown has already elapsed. If a human has taken over, return false.
func IsGreetingDue(last, now time.Time, cooldown time.Duration, humanPaused bool) bool {
	if humanPaused {
		return false
	}
	if last.IsZero() {
		return true
	}
	if cooldown <= 0 {
		return false
	}
	return !now.Before(last.Add(cooldown))
}
