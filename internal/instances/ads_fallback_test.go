package instances

import (
 "testing"
 "github.com/hypeneural/botwhatsapp/internal/core"
)
func TestAdsRegistryHonorsFallbackAndSeparateGreetingInstance(t *testing.T){
 reg,err:=New([]Definition{
  {Slug:"ads",InstanceUUID:"ads-uuid",BusinessPhoneE164:"554832200440",
   Role:Ads,Enabled:true,TokenEnv:"EVOLUTION_ADS_APIKEY",
   WebhookSecretEnv:"ADS_WEBHOOK_SECRET",AdsTriggers:[]string{"pacotes de Natal"},AdsFallback:true,CRMAllowed:true},
  {Slug:"greeting",InstanceUUID:"greeting-uuid",BusinessPhoneE164:"5548996425287",
   Role:Greeting,Enabled:true,TokenEnv:"EVOLUTION_GREETING_APIKEY",WebhookSecretEnv:"GREETING_WEBHOOK_SECRET"},
 })
 if err!=nil{t.Fatal(err)}
 for _,tc:=range []struct{id,text string;action core.Action}{
  {"ads-uuid","pacotes de Natal",core.AdsProposal},
  {"ads-uuid","Oi tudo bem",core.Greeting},
  {"greeting-uuid","pacotes de Natal",core.Greeting},
 }{
  e:=core.Envelope{Event:"Message",InstanceID:tc.id}
  e.Data.Info.ID="sample-"+tc.id
  e.Data.Info.Chat="5548999999999@s.whatsapp.net"
  e.Data.Message.Conversation=tc.text
  d,err:=reg.Decide(e)
  if err!=nil||d.Action!=tc.action{t.Fatalf("%v: %+v %v",tc,d,err)}
 }
}
