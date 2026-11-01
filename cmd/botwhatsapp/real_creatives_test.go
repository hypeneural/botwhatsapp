package main

import (
 "os"
 "path/filepath"
 "testing"

 "github.com/hypeneural/botwhatsapp/internal/core"
)

func TestConfiguredRealCreativesBothMatchNoBroadTrigger(t *testing.T){
 path:=filepath.Join("..","..","config","ads-triggers.txt")
 if _,err:=os.Stat(path);err!=nil{t.Fatal(err)}
 t.Setenv("BOT_ADS_TRIGGERS_FILE",path)
 t.Setenv("BOT_ADS_TRIGGERS","")
 phrases,err:=loadAdTriggers()
 if err!=nil {t.Fatal(err)}
 expected:=[]string{
 "Quero conhecer os pacotes do Primeiro Natal em Tijucas",
 "Quero conhecer os pacotes de Natal em Tijucas",
 }
 cfg:=core.Config{AdsUUID:"ads",AdsFallback:true,AdsTriggers:phrases}
 e:=core.Envelope{Event:"Message",InstanceID:"ads"}
 e.Data.Info.ID="synthetic"
 e.Data.Info.Chat="5548999999999@s.whatsapp.net"
 e.Data.Info.SenderAlt="fictitious@lid"
 for _,phrase:=range expected {
  e.Data.Message.Extended.Text=phrase
  if d:=core.Select(e,cfg);d.Action!=core.AdsProposal{
   t.Fatalf("%q must launch Christmas 416 funnel, got %+v",phrase,d)
  }
 }
 for _,str:=range []string{"Oi, Natal","Quero orçamento","Quero conhecer os pacotes de natal em Florianópolis"}{
  e.Data.Message.Extended.Text=str
  if d:=core.Select(e,cfg);d.Action!=core.Ignore{
   t.Fatalf("%q must remain organic and never auto-greet: %+v",str,d)
  }
 }
 // Even when a provider produces an unusual numeric Chat JID, the explicit
 // group flag must win over advertisement text.
 e.Data.Info.IsGroup=true
 e.Data.Message.Extended.Text=expected[1]
 if d:=core.Select(e,cfg);d.Action!=core.Ignore{t.Fatalf("group should never schedule: %+v",d)}
 e.Data.Info.IsGroup=false
 // Some CTWA reports carry ONLY a LID; NEVER invent a phone/CRM recipient.
 e.Data.Info.Chat="123456789@lid"
 e.Data.Info.Sender=""
 e.Data.Info.SenderAlt=""
 e.Data.Message.Extended.Text=expected[1]
 if d:=core.Select(e,cfg);d.Action!=core.Ignore{
  t.Fatalf("LID-only event cannot schedule: %+v",d)
 }
}
