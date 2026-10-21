package ingress

import (
 "net/http"
 "net/http/httptest"
 "path/filepath"
 "strings"
 "sync"
 "testing"

 "github.com/hypeneural/botwhatsapp/internal/core"
)
func TestConcurrentDuplicateWritesOneSpoolFile(t *testing.T){
 secret:=strings.Repeat("A",40)
 dir:=t.TempDir()
 srv,err:=New(Settings{Enabled:true,AdsSecret:secret,GreetingSecret:strings.Repeat("B",40),SpoolDir:dir,Core:core.Config{
  AdsUUID:"ads-id",GreetingUUID:"greet-id",AdsTriggers:[]string{"proposta"},AdsFallback:false,
 }})
 if err!=nil{t.Fatal(err)}
 body:=`{"event":"Message","instanceId":"ads-id","data":{"Info":{"ID":"same-msg","Chat":"5548996425287@s.whatsapp.net"},"Message":{"conversation":"proposta"}}}`
 const n=24
 var wg sync.WaitGroup
 codes:=make(chan int,n)
 for i:=0;i<n;i++ {
  wg.Add(1);go func(){
   defer wg.Done()
   w:=httptest.NewRecorder()
   srv.ServeHTTP(w,httptest.NewRequest(http.MethodPost,"/webhook/ads/"+secret,strings.NewReader(body)))
   codes<-w.Code
  }()
 }
 wg.Wait();close(codes)
 for code:=range codes {if code!=202 {t.Fatalf("unexpected response %d",code)}}
 paths,err:=filepath.Glob(filepath.Join(dir,"*.json"))
 if err!=nil||len(paths)!=1 {t.Fatalf("duplicate not deduped: %v %v",paths,err)}
}
