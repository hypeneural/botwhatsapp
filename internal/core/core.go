// Package core implements pure, side-effect-free Evolution Go v0.7.2 routing rules.
// It never calls the WhatsApp API or the CRM.
package core

import (
 "encoding/json"
 "errors"
 "regexp"
 "strings"
 "unicode"
)

type Envelope struct {
 Event string `json:"event"`
 InstanceID string `json:"instanceId"`
 InstanceName string `json:"instanceName"`
 Data struct {
  Info struct {
   ID string `json:"ID"`
   Chat string `json:"Chat"`
   Sender string `json:"Sender"`
   SenderAlt string `json:"SenderAlt"`
   IsFromMe bool `json:"IsFromMe"`
   IsGroup bool `json:"IsGroup"`
   PushName string `json:"PushName"`
  } `json:"Info"`
  Message struct {
   Conversation string `json:"conversation"`
   Extended struct { Text string `json:"text"` } `json:"extendedTextMessage"`
  } `json:"Message"`
 } `json:"data"`
}

type Config struct {
 AdsUUID string
 GreetingUUID string
 AdsTriggers []string
 AdsFallback bool // legacy only; ads no longer auto-greet on unmatched text
 BusinessPhoneE164 string // verified own line: never a customer
}
type Action string
const (
 Ignore Action = "ignore"
 OutboundObserved Action = "outbound_observed"
 AdsProposal Action = "ads_proposal"
 Greeting Action = "greeting"
 MenuOption Action = "menu_option"
)
type Decision struct {
 Action Action `json:"action"`
 InstanceID string `json:"instance_id"`
 MessageID string `json:"message_id"`
 PhoneE164 string `json:"phone_e164"`
 ReplyJID string `json:"reply_jid,omitempty"`
 LIDAlias string `json:"lid_alias,omitempty"`
 CRMPhone string `json:"crm_phone,omitempty"`
 Name string `json:"name,omitempty"`
 Option string `json:"option,omitempty"`
 Reason string `json:"reason,omitempty"`
}
var errInvalid = errors.New("invalid webhook envelope")
var digits = regexp.MustCompile(`^[0-9]+$`)
func Parse(b []byte) (Envelope,error) {
 var e Envelope
 if err:=json.Unmarshal(b,&e);err!=nil{return e,errInvalid}
 if len(e.InstanceID)>128 || len(e.Data.Info.ID)>200 {return e,errInvalid}
 return e,nil
}
func Normalize(s string) string {
 s=strings.TrimSpace(strings.ToLower(s))
 fields:=strings.Fields(s)
 return strings.TrimFunc(strings.Join(fields," "),func(r rune)bool{
  return unicode.IsPunct(r) || unicode.IsSpace(r)
 })
}
func JIDPhone(chat string) (string,bool) {
 // Do not confuse WhatsApp LIDs / groups / newsletters with a phone number.
 suffix:="@s.whatsapp.net"
 if !strings.HasSuffix(chat,suffix) {return "",false}
 value:=strings.TrimSuffix(chat,suffix)
 if len(value)<12 || len(value)>13 || !strings.HasPrefix(value,"55") || !digits.MatchString(value) {return "",false}
 return value,true
}
func CRMPhone(e164 string)(string,bool) {
 if !strings.HasPrefix(e164,"55"){return "",false}
 n:=strings.TrimPrefix(e164,"55")
 if (len(n)!=10 && len(n)!=11)||!digits.MatchString(n){return "",false}
 return n,true
}
func Select(e Envelope,c Config) Decision {
 d:=Decision{Action:Ignore,InstanceID:e.InstanceID,MessageID:e.Data.Info.ID}
 if e.Event!="Message" || e.Data.Info.ID=="" {d.Reason="not_new_inbound";return d}
 if e.Data.Info.IsGroup {d.Reason="non_private_chat";return d}
 if e.InstanceID!=c.AdsUUID && e.InstanceID!=c.GreetingUUID {d.Reason="unknown_instance";return d}
 chat:=e.Data.Info.Chat
 if strings.HasSuffix(chat,"@g.us")||strings.HasSuffix(chat,"@newsletter")||strings.Contains(chat,"broadcast"){d.Reason="non_private_chat";return d}
 // Keep the chat JID for replies. A LID's digits are never a phone number.
 d.ReplyJID=chat
 // Some Evolution versions swap phone JID into Chat and preserve the LID in
 // SenderAlt. Store ONLY the explicit paired alias; never treat it as a phone.
 switch {
 case strings.HasSuffix(chat,"@lid"): d.LIDAlias=chat
 case strings.HasSuffix(e.Data.Info.SenderAlt,"@lid"): d.LIDAlias=e.Data.Info.SenderAlt
 case strings.HasSuffix(e.Data.Info.Sender,"@lid"): d.LIDAlias=e.Data.Info.Sender
 }
 // Preserve the self-declared WhatsApp display name on unresolved @lid events.
 // Do not infer a phone number from the name or from LID digits.
 name:=strings.TrimSpace(e.Data.Info.PushName)
 if len([]rune(name))>70 || strings.ContainsAny(name,";\r\n") {name=""}
 d.Name=name
 phone,ok:=JIDPhone(chat)
 if !ok && strings.HasSuffix(chat,"@lid"){
  phone,ok=JIDPhone(e.Data.Info.Sender)
  if !ok {phone,ok=JIDPhone(e.Data.Info.SenderAlt)}
 }
 if !ok {d.Reason="unresolved_phone_or_non_private_chat";return d}
 // A looped/self-chat may be delivered with IsFromMe=false. Never treat
 // the official business number as a lead, even on an exact ad phrase.
 if c.BusinessPhoneE164!="" && phone==c.BusinessPhoneE164 {
  d.Reason="own_business_number";return d
 }
 d.PhoneE164=phone
 d.CRMPhone,_=CRMPhone(phone)
 if e.Data.Info.IsFromMe {
  // Outbound activity is a suppressor for future automatic greetings.
  // Never enqueue a reply or a CRM proposal from the echo itself.
  d.Action=OutboundObserved
  d.Reason="outbound_activity_observed"
  return d
 }
 body:=e.Data.Message.Conversation
 if body=="" {body=e.Data.Message.Extended.Text}
 body=Normalize(body)
 if body=="" {d.Reason="no_text";return d}
 switch body {
 case "parar","sair","cancelar","não quero","nao quero","stop","unsubscribe":
  d.Reason="opt_out_requires_persistent_suppression";return d
 }
 if e.InstanceID==c.AdsUUID {
  for _,t:=range c.AdsTriggers {if body!="" && body==Normalize(t) {d.Action=AdsProposal;return d}}
  // P0: the paid-traffic number is also used for human chats. A generic
  // inbound message, including a menu digit, must not start an automated
  // greeting merely because it reached this instance. The old fallback
  // flag is intentionally ignored until an explicitly consented flow exists.
  d.Reason="ads_unmatched_no_automation"
  return d
 }
 switch body {
 case "1","2","3","4","5": d.Action=MenuOption;d.Option=body
 default: d.Action=Greeting
 }
 return d
}
