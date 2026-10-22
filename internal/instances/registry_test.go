package instances

import (
 "errors"
 "testing"
 "time"
 "github.com/hypeneural/botwhatsapp/internal/core"
)
func TestRegistrySeparatesInstancesAndRoles(t *testing.T){
 r,err:=New([]Definition{
  {Slug:"ads-main",InstanceUUID:"uuid-a",BusinessPhoneE164:"554832200440",
   Role:Ads,Enabled:true,TokenEnv:"ADS_TOKEN",WebhookSecretEnv:"ADS_SECRET",
   AdsTriggers:[]string{"quero natal 2026"},CRMAllowed:true},
  {Slug:"greeting-b",InstanceUUID:"uuid-b",BusinessPhoneE164:"5548996425287",
   Role:Greeting,Enabled:true,TokenEnv:"GREETING_TOKEN",WebhookSecretEnv:"GREET_SECRET"},
  {Slug:"readonly",InstanceUUID:"uuid-c",BusinessPhoneE164:"5548999999999",
   Role:Observe,Enabled:true,TokenEnv:"OBS_TOKEN",WebhookSecretEnv:"OBS_SECRET"},
 })
 if err!=nil{t.Fatal(err)}
 ev:=core.Envelope{}
 ev.Event="Message";ev.InstanceID="uuid-a";ev.Data.Info.ID="id";ev.Data.Info.Chat="5548998888888@s.whatsapp.net"
 ev.Data.Message.Conversation="quero natal 2026"
 got,err:=r.Decide(ev);if err!=nil||got.Action!=core.AdsProposal{t.Fatal(got,err)}
 ev.InstanceID="uuid-b"
 got,err=r.Decide(ev);if err!=nil||got.Action!=core.Greeting{t.Fatal(got,err)}
 ev.InstanceID="uuid-c"
 got,err=r.Decide(ev);if err!=nil||got.Action!=core.Ignore{t.Fatal(got,err)}
 ev.InstanceID="unknown"
 _,err=r.Decide(ev);if !errors.Is(err,ErrUnknown){t.Fatal(err)}
}
func TestRoutingDoesNotAcceptPathConfusion(t *testing.T){
 r,_:=New([]Definition{{Slug:"ads-main",InstanceUUID:"uuid-a",
  BusinessPhoneE164:"554832200440",Role:Ads,Enabled:true,
  TokenEnv:"ADS_TOKEN",WebhookSecretEnv:"SECRET",AdsTriggers:[]string{"foo"}}})
 secret:="1234567890abcdefghijABCDEFGHIJKLxyz"
 cases:=[]struct{path string;ok bool}{
  {"/webhook/ads-main/"+secret,true},
  {"/webhook/ads-main/extra/"+secret,false},
  {"/webhook/ads-main/"+secret+"/",false},
  {"/webhook/ads-other/"+secret,false},
  {"/webhook/ads-main/wrong",false},
 }
 for _,tc:=range cases{
  _,err:=r.VerifyRoute(tc.path,"ads-main",secret)
  if (err==nil)!=tc.ok{t.Errorf("path accepted=%t: %q",err==nil,tc.path)}
 }
}
func TestInvalidRegistryBlocked(t *testing.T){
 cases:=[][]Definition{
  {{Slug:"a",InstanceUUID:"uuid-a",BusinessPhoneE164:"554832200440",Role:Ads,Enabled:true}},
  {{Slug:"x",Role:Ads,Enabled:true}},
  {{Slug:"a",Role:Greeting,CRMAllowed:true}},
  {{Slug:"a",Role:Ads,InstanceUUID:"same",BusinessPhoneE164:"554832200440"},
   {Slug:"b",Role:Greeting,InstanceUUID:"same",BusinessPhoneE164:"5548996425287"}},
  {{Slug:"a",Role:Ads,InstanceUUID:"uuid-a",BusinessPhoneE164:"554832200440"},
   {Slug:"a",Role:Greeting,InstanceUUID:"uuid-b",BusinessPhoneE164:"5548996425287"}},
 }
 for i,defs:=range cases{if _,err:=New(defs);err==nil{t.Fatalf("bad config %d allowed",i)}}
}
func TestEventAndLeadKeys(t *testing.T){
 a,_:=EventKey("uuid-a","Message","id-1")
 b,_:=EventKey("uuid-b","Message","id-1")
 if a==b{t.Fatal("same message ID across instances deduped incorrectly")}
 loc:=time.FixedZone("BRT",-3*3600)
 now:=time.Date(2026,10,22,2,0,0,0,time.UTC) // 21 Oct, 23:00 BRT
 k1,_:=LeadClaimKey("estudio","48996553954",416,now,loc)
 k2,_:=LeadClaimKey("estudio","48996553954",416,now.Add(time.Hour),loc)
 if k1==k2{t.Fatal("date boundary must release next daily claim")}
 k3,_:=LeadClaimKey("estudio","48996553954",416,now,loc)
 if k1!=k3{t.Fatal("same brand lead/slot must dedup across instances")}
 if _,err:=LeadClaimKey("estudio","5548996553954",416,now,loc);err==nil{t.Fatal("DDI must be stripped")}
}
