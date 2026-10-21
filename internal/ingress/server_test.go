package ingress

import (
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"

 "github.com/hypeneural/botwhatsapp/internal/core"
)
func TestWebhookPreviewDurableAndIdempotent(t *testing.T){
 dir:=t.TempDir()
 secret:=strings.Repeat("a",40)
 srv,err:=New(Settings{Enabled:true,AdsSecret:secret,GreetingSecret:strings.Repeat("b",40),SpoolDir:dir,Core:core.Config{
  AdsUUID:"uuid-ads",GreetingUUID:"uuid-greet",AdsTriggers:[]string{"quero proposta natal"},
 }})
 if err!=nil {t.Fatal(err)}
 body:=`{"event":"Message","instanceId":"uuid-ads","instanceToken":"SHOULD_NOT_PERSIST","data":{"Info":{"ID":"msgA","Chat":"5548996425287@s.whatsapp.net","IsFromMe":false},"Message":{"conversation":"quero proposta natal"}}}`
 for i:=0;i<2;i++{
  req:=httptest.NewRequest(http.MethodPost,"/webhook/ads/"+secret,strings.NewReader(body))
  w:=httptest.NewRecorder();srv.ServeHTTP(w,req)
  if w.Code!=202 {t.Fatalf("attempt %d code %d: %s",i,w.Code,w.Body)}
 }
 paths,err:=filepath.Glob(filepath.Join(dir,"*.json"));if err!=nil||len(paths)!=1{t.Fatal(err,paths)}
 buf,err:=os.ReadFile(paths[0]);if err!=nil{t.Fatal(err)}
 if strings.Contains(string(buf),"SHOULD_NOT_PERSIST"){t.Fatal("instanceToken was persisted")}
 var rec struct{Decision core.Decision `json:"decision"`;Status string `json:"status"`}
 if err:=json.Unmarshal(buf,&rec);err!=nil||rec.Decision.Action!=core.AdsProposal||rec.Status!="planned_preview_only"{t.Fatal(err,rec)}
 wrong:=httptest.NewRecorder()
 srv.ServeHTTP(wrong,httptest.NewRequest("POST","/webhook/ads/"+strings.Repeat("z",40),strings.NewReader(body)))
 if wrong.Code!=401{t.Fatal("missing webhook verification",wrong.Code)}
 wrong=httptest.NewRecorder()
 srv.ServeHTTP(wrong,httptest.NewRequest("POST","/webhook/ads/"+secret,strings.NewReader(strings.ReplaceAll(body,"uuid-ads","uuid-greet"))))
 if wrong.Code!=403{t.Fatal("accepted mismatched UUID",wrong.Code)}
}
func TestDisabledFailsClosed(t *testing.T){
 srv,err:=New(Settings{SpoolDir:t.TempDir()});if err!=nil{t.Fatal(err)}
 w:=httptest.NewRecorder();srv.ServeHTTP(w,httptest.NewRequest("POST","/webhook/ads/s",strings.NewReader("{}")))
 if w.Code!=503{t.Fatal(w.Code)}
}
