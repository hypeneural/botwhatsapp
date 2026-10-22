// Offline demo: reads synthetic Evolution Go events, prints the workflow plan.
// Does not use credentials, contact APIs, Evolution Go or CRM.
package main
import (
 "encoding/json"
 "fmt"
 "os"
 "github.com/hypeneural/botwhatsapp/internal/core"
 "github.com/hypeneural/botwhatsapp/internal/crm"
 "github.com/hypeneural/botwhatsapp/internal/workflow"
)
func main(){
 if len(os.Args)!=2 {fmt.Fprintln(os.Stderr,"usage: go run ./cmd/simulate testdata/ads.json");os.Exit(2)}
 raw,err:=os.ReadFile(os.Args[1]);if err!=nil{fmt.Fprintln(os.Stderr,"fixture unavailable");os.Exit(1)}
 ev,err:=core.Parse(raw);if err!=nil{fmt.Fprintln(os.Stderr,"invalid fixture");os.Exit(1)}
 cfg:=core.Config{AdsUUID:"ads-demo",GreetingUUID:"greeting-demo",AdsTriggers:[]string{"Quero a proposta de Natal 2026"},AdsFallback:true}
 d:=core.Select(ev,cfg)
 p:=workflow.Create(d)
 out:=struct{
  Mode string `json:"mode"`
  Decision core.Decision `json:"decision"`
  Plan workflow.Plan `json:"plan"`
  CRMPayload *crm.Request `json:"crm_payload,omitempty"`
 }{Mode:"OFFLINE_SIMULATION_NO_SEND",Decision:d,Plan:p}
 if d.Action==core.AdsProposal{
  // Dummy CRM instance ID is for PRINT ONLY, never a production credential.
  demo:=999999
  req,err:=crm.BuildWithoutName(d.CRMPhone,&demo)
  if err!=nil{fmt.Fprintln(os.Stderr,"contact format rejected");os.Exit(1)}
  out.CRMPayload=&req
 }
 enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ")
 if err:=enc.Encode(out);err!=nil{os.Exit(1)}
}
