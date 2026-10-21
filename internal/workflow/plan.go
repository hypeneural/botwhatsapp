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
 SendAdText Step="send_ad_text"
 SendAdAudio Step="send_ad_audio"
 ScheduleNatal Step="schedule_natal_crm"
 SendGreetingImage Step="send_greeting_image_caption"
 SendMenuReply Step="send_menu_reply"
 PauseHuman Step="pause_for_human"
)
type Plan struct {
 Steps []Step `json:"steps"`
 Message string `json:"message,omitempty"`
}
func Create(d core.Decision) Plan {
 switch d.Action {
 case core.AdsProposal:
  return Plan{Steps:[]Step{SendAdText,SendAdAudio,ScheduleNatal}}
 case core.Greeting:
  return Plan{Steps:[]Step{SendGreetingImage}}
 case core.MenuOption:
  if d.Option=="5" {return Plan{Steps:[]Step{SendMenuReply,PauseHuman},Message:Reply("5")}}
  return Plan{Steps:[]Step{SendMenuReply},Message:Reply(d.Option)}
 }
 return Plan{}
}
func Reply(option string)string {
 switch strings.TrimSpace(option){
 case "1":return "🎄 Veja nossos ensaios de Natal 2026: https://evydencia.com/catalogo/natal"
 case "2":return "📅 Confira as opções de horários: https://horarios.fotosdenatal.com/"
 case "3":return "📸 Vamos ajudar com suas fotos e pedidos. Nossa equipe continuará o atendimento por aqui."
 case "4":return "📍 Nossa localização: https://ir.fotosdenatal.com/"
 case "5":return "👩‍💼 Vou deixar seu atendimento com nossa equipe. Assim que possível, responderemos por aqui."
 default:return ""
 }
}
// IsGreetingDue returns true when no previous greeting exists, or the
// cooldown has already elapsed. If a human has taken over, return false.
func IsGreetingDue(last,now time.Time,cooldown time.Duration,humanPaused bool)bool{
 if humanPaused{return false}
 if last.IsZero(){return true}
 if cooldown<=0{return false}
 return !now.Before(last.Add(cooldown))
}
