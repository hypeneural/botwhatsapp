package core

import (
 "os"
 "testing"
)
func TestRealisticCTWAAndAdsFallback(t *testing.T){
 raw,err:=os.ReadFile("../../testdata/ctwa-first-natal.json")
 if err!=nil{t.Fatal(err)}
 e,err:=Parse(raw)
 if err!=nil{t.Fatal(err)}
 cfg:=Config{AdsUUID:"ads-uuid",AdsFallback:true,
  AdsTriggers:[]string{"Quero conhecer os pacotes do Primeiro Natal em Tijucas",
   "Quero receber o orçamento de Natal em Tijucas"}}
 got:=Select(e,cfg)
 if got.Action!=AdsProposal||got.CRMPhone!="48999999999"||
  got.ReplyJID!="5548999999999@s.whatsapp.net" {
   t.Fatalf("ad webhook not recognized: %+v",got)
 }
 e.Data.Message.Extended.Text="Boa tarde, quanto custa?"
 if got:=Select(e,cfg);got.Action!=Greeting{t.Fatalf("unmatched should greet: %+v",got)}
 e.Data.Message.Extended.Text="5"
 if got:=Select(e,cfg);got.Action!=MenuOption||got.Option!="5"{t.Fatal("menu on ads phone",got)}
 e.Data.Info.Chat="123456789@lid";e.Data.Info.Sender="";e.Data.Info.SenderAlt="567891@lid"
 e.Data.Message.Extended.Text="Quero conhecer os pacotes do Primeiro Natal em Tijucas"
 if got:=Select(e,cfg);got.Action!=Ignore{t.Fatal("LID is not a number",got)}
}
