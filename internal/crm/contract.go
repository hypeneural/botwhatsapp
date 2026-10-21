// Package crm holds the *existing* API request format. It performs no HTTP call.
package crm
import (
 "encoding/json"
 "errors"
 "regexp"
 "strings"
)
const WithName=415
const WithoutName=416
var phoneRE=regexp.MustCompile(`^[0-9]{10,11}$`)
var ErrInvalidContact=errors.New("crm: invalid DDD+phone (10 or 11 digits, no DDI)")
type Request struct {
 Campaign int `json:"campaign"`
 Contacts string `json:"contacts"`
 Instance *int `json:"instance,omitempty"`
 UseLeadsSystem bool `json:"use_leads_system"`
}
// Both IDs are defined in the CRM Natal2026ProposalManifest but database readiness is unverified.
func Build(phone,name string,instance *int)(Request,error){
 if !phoneRE.MatchString(phone){return Request{},ErrInvalidContact}
 name=strings.TrimSpace(name)
 if strings.ContainsAny(name,";\r\n") || len([]rune(name))>70 {name=""}
 campaign:=WithoutName
 contacts:=phone
 if name!="" {campaign=WithName;contacts+=";"+name}
 return Request{Campaign:campaign,Contacts:contacts,Instance:instance,UseLeadsSystem:false},nil
}
func Encode(r Request)([]byte,error){return json.Marshal(r)}
