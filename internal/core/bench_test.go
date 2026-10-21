package core

import "testing"

func BenchmarkSelectAd(b *testing.B){
 cfg:=Config{AdsUUID:"ads-id",GreetingUUID:"greet-id",AdsTriggers:[]string{"ola quero proposta de natal"}}
 ev:=fixture("ads-id","5548996425287@s.whatsapp.net","ola quero proposta de natal","msg1")
 b.ReportAllocs()
 for i:=0;i<b.N;i++ { _=Select(ev,cfg) }
}
func BenchmarkParseWebhook(b *testing.B){
 raw:=[]byte(`{"event":"Message","instanceId":"ads-id","data":{"Info":{"ID":"msg123","Chat":"5548996425287@s.whatsapp.net","IsFromMe":false},"Message":{"conversation":"ola quero proposta de natal"}}}`)
 b.ReportAllocs()
 for i:=0;i<b.N;i++ { _,_=Parse(raw) }
}
