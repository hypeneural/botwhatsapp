package core

import (
 "encoding/json"
 "testing"
)
func fixture(uuid,chat,txt,id string) Envelope{
 var e Envelope
 e.Event="Message";e.InstanceID=uuid;e.Data.Info.Chat=chat;e.Data.Info.ID=id;e.Data.Message.Conversation=txt
 return e
}
func TestRouting(t *testing.T){
 c:=Config{AdsUUID:"ads-uuid",GreetingUUID:"greet-uuid",AdsTriggers:[]string{"Olá! Quero a proposta Natal 2026."},AdsFallback:true}
 a:=fixture("ads-uuid","5548996425287@s.whatsapp.net","  OLÁ!  QUERO a proposta Natal 2026. ","msg1")
 if got:=Select(a,c);got.Action!=AdsProposal||got.CRMPhone!="48996425287"{t.Fatalf("ads: %+v",got)}
 a.Data.Info.IsFromMe=true
 if got:=Select(a,c);got.Action!=Ignore {t.Fatalf("own message: %+v",got)}
 a.Data.Info.IsFromMe=false;a.Data.Info.Chat="12345@lid"
 if got:=Select(a,c);got.Action!=Ignore{t.Fatalf("lid must not become CRM phone: %+v",got)}
 a=fixture("greet-uuid","5548996425287@s.whatsapp.net","5","msg2")
 if got:=Select(a,c);got.Action!=MenuOption||got.Option!="5"{t.Fatalf("menu: %+v",got)}
 a.Data.Info.Chat="12345@g.us"
 if got:=Select(a,c);got.Action!=Ignore{t.Fatalf("group: %+v",got)}
 a=fixture("greet-uuid","5548996425287@s.whatsapp.net","bom dia","msg3")
 if got:=Select(a,c);got.Action!=Greeting{t.Fatalf("greet: %+v",got)}
 a=fixture("ads-uuid","5548996425287@s.whatsapp.net","quero a proposta","msg4")
 if got:=Select(a,c);got.Action==AdsProposal{t.Fatalf("false positive: %+v",got)}
}
func TestEnvelope(t *testing.T){
 e,err:=Parse([]byte(`{"event":"Message","instanceId":"u","data":{"Info":{"ID":"abc","Chat":"5548996425287@s.whatsapp.net","IsFromMe":false},"Message":{"extendedTextMessage":{"text":"1"}}}}`))
 if err!=nil||e.Data.Message.Extended.Text!="1"{t.Fatal(err,e)}
 if _,err:=json.Marshal(e);err!=nil{t.Fatal(err)}
}
func TestCRMPhone(t *testing.T){
 for _,s:=range []string{"5548996425287","554832200440"} {if _,ok:=CRMPhone(s);!ok{t.Fatal(s)}}
 for _,s:=range []string{"48996425287","551234","5599ABCDEF999","5511111111111111"} {if _,ok:=CRMPhone(s);ok{t.Fatal("accepted",s)}}
}
