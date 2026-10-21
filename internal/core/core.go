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
   IsFromMe bool `json:"IsFromMe"`
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
 AdsFallback bool
}
type Action string
const (
 Ignore Action = "ignore"
 AdsProposal Action = "ads_proposal"
 Greeting Action = "greeting"
 MenuOption Action = "menu_option"
)
type Decision struct {
 Action Action `json:"action"`
 InstanceID string `json:"instance_id"`
 MessageID string `json:"message_id"`
 PhoneE164 string `json:"phone_e164"`
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
 if e.Event!="Message" || e.Data.Info.IsFromMe || e.Data.Info.ID=="" {d.Reason="not_new_inbound";return d}
 if e.InstanceID!=c.AdsUUID && e.InstanceID!=c.GreetingUUID {d.Reason="unknown_instance";return d}
 phone,ok:=JIDPhone(e.Data.Info.Chat)
 if !ok {d.Reason="unresolved_phone_or_non_private_chat";return d}
 d.PhoneE164=phone
 d.CRMPhone,_=CRMPhone(phone)
 name:=strings.TrimSpace(e.Data.Info.PushName)
 if len([]rune(name))>70 || strings.ContainsAny(name,";\r\n") {name=""}
 d.Name=name
 body:=e.Data.Message.Conversation
 if body=="" {body=e.Data.Message.Extended.Text}
 body=Normalize(body)
 if body=="" {d.Reason="no_text";return d}
 if e.InstanceID==c.AdsUUID {
  for _,t:=range c.AdsTriggers {if body!="" && body==Normalize(t) {d.Action=AdsProposal;return d}}
  if !c.AdsFallback {d.Reason="unmatched_ads";return d}
 }
 switch body {
 case "1","2","3","4","5": d.Action=MenuOption;d.Option=body
 default: d.Action=Greeting
 }
 return d
}
