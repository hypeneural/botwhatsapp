// Package crm models the existing Evydencia CRM campaign API. No CRM changes.
package crm

import (
 "encoding/json"
 "errors"
 "regexp"
 "strings"
)

const (
 WithName = 415
 WithoutName = 416 // The DEFAULT for inbound Meta Ads leads.
 CampaignKind = "campaign"
)

var (
 phoneRE = regexp.MustCompile(`^[0-9]{10,11}$`)
 ErrInvalidContact = errors.New("crm: expected 10 or 11 national digits, no DDI")
 ErrInvalidName = errors.New("crm: invalid recipient name")
)

type Request struct {
 Campaign int `json:"campaign"`
 Contacts string `json:"contacts"`
 Instance *int `json:"instance,omitempty"`
 UseLeadsSystem bool `json:"use_leads_system"`
 Type string `json:"type"`
}

// Build deliberately ignores the caller's name. It preserves the historical
// signature for existing callers, while ALWAYS selecting the no-name campaign.
// Don't infer campaign 415 from PushName or other unverified WhatsApp metadata.
func Build(phone, _ string, instance *int) (Request, error) {
 return BuildWithoutName(phone,instance)
}
func BuildWithoutName(phone string, instance *int) (Request,error) {
 if !phoneRE.MatchString(phone){return Request{},ErrInvalidContact}
 return Request{
  Campaign:WithoutName, Contacts:phone, Instance:instance,
  UseLeadsSystem:false, Type:CampaignKind,
 },nil
}

// Explicit future opt-in only. Not used by the two current workflows.
func BuildWithName(phone,name string,instance *int)(Request,error){
 if !phoneRE.MatchString(phone){return Request{},ErrInvalidContact}
 name=strings.TrimSpace(name)
 if name=="" || strings.ContainsAny(name,";\r\n") || len([]rune(name))>70{
  return Request{},ErrInvalidName
 }
 return Request{
  Campaign:WithName,Contacts:phone+";"+name,Instance:instance,
  UseLeadsSystem:false,Type:CampaignKind,
 },nil
}
func Encode(r Request)([]byte,error){return json.Marshal(r)}
