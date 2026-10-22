package outbox

import (
 "context"
 "testing"
 "time"

 "github.com/hypeneural/botwhatsapp/internal/core"
 "github.com/hypeneural/botwhatsapp/internal/crm"
)

func TestSecondCreativeDoesNotResendSameLeadSequence(t *testing.T){
 store,_:=setupTestStore(t)
 sender:=&mockEvolutionSender{}
 scheduler:=&mockCRMScheduler{}
 instance:=1
 w,err:=NewWorker(Config{CRMInstanceID:&instance,OperationalAuthorization:true,
  AdAudioFixture:[]byte("synthetic-audio-only-in-test"),
  AdsFlowCooldown:24*time.Hour,
 },store,sender,scheduler)
 if err!=nil{t.Fatal(err)}
 first:=core.Decision{Action:core.AdsProposal,InstanceID:"ads-uuid",
  MessageID:"first-creative",PhoneE164:"5548999999999",
  ReplyJID:"5548999999999@s.whatsapp.net",CRMPhone:"48999999999"}
 if err:=w.IngestDecision(first);err!=nil{t.Fatal(err)}
 if _,err:=w.DrainPending(context.Background());err!=nil{t.Fatal(err)}
 second:=first;second.MessageID="different-message-second-creative"
 if err:=w.IngestDecision(second);err!=nil{t.Fatal(err)}
 if _,err:=w.DrainPending(context.Background());err!=nil{t.Fatal(err)}
 if sender.textCalls!=1||sender.mediaCalls!=1||scheduler.calls!=1 {
  t.Fatalf("duplicate marketing sequence: text=%d audio=%d crm=%d",sender.textCalls,sender.mediaCalls,scheduler.calls)
 }
 if scheduler.lastReq.Campaign!=crm.WithoutName {t.Fatal("CRM campaign must remain 416")}
 state,err:=store.GetContact(first.PhoneE164)
 if err!=nil||state.LastAdsFlowAt.IsZero(){t.Fatal("ads cooldown was not durably persisted",err)}
}

func TestContactPolicyMutationsNeverClearOptOut(t *testing.T){
 store,_:=setupTestStore(t)
 phone:="5548999999999"
 if _,err:=store.UpdateContact(phone,func(c *ContactState)error{
  c.OptOut=true
  c.OptOutAt=time.Now().UTC()
  return nil
 });err!=nil{t.Fatal(err)}
 // Simulate a greeting text/audio completion racing with PARAR.
 if _,err:=store.UpdateContact(phone,func(c *ContactState)error{
  c.LastGreetingAt=time.Now().UTC()
  c.LastAdsFlowAt=time.Now().UTC()
  return nil
 });err!=nil{t.Fatal(err)}
 c,err:=store.GetContact(phone)
 if err!=nil||!c.OptOut||c.OptOutAt.IsZero(){
  t.Fatal("opt-out overwritten by unrelated contact metadata",err)
 }
}
