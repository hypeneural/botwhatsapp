package ingress

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

func TestWebhookWaitsForDurableOutboxAndCanRetry(t *testing.T) {
	dir:=t.TempDir()
	secret:=strings.Repeat("A",40)
	attempts:=0
	server,err:=New(Settings{
		Enabled:true,SpoolDir:dir,AdsSecret:secret,GreetingSecret:strings.Repeat("B",40),
		Core:core.Config{AdsUUID:"ads",GreetingUUID:"greet",AdsTriggers:[]string{"quero proposta"}},
		OnDecision:func(d core.Decision)error{
			attempts++
			if attempts==1{return errors.New("disk failure")}
			return nil
		},
	})
	if err!=nil{t.Fatal(err)}
	body:=`{"event":"Message","instanceId":"ads","data":{"Info":{"ID":"ack-test","Chat":"5548999999999@s.whatsapp.net"},"Message":{"conversation":"quero proposta"}}}`
	for i,want:=range []int{503,202,202}{
		w:=httptest.NewRecorder()
		server.ServeHTTP(w,httptest.NewRequest("POST","/webhook/ads/"+secret,strings.NewReader(body)))
		if w.Code!=want{t.Fatalf("try %d: got %d, want %d",i,w.Code,want)}
	}
	if attempts!=2{t.Fatalf("callback invoked %d times: must not duplicate after ACK",attempts)}
	files,err:=filepath.Glob(filepath.Join(dir,"*.json"))
	if err!=nil||len(files)!=1{t.Fatal(err,files)}
}
func TestOptOutDeliveredToDurableHook(t *testing.T){
	secret:=strings.Repeat("S",40)
	var got core.Decision
	server,err:=New(Settings{Enabled:true,SpoolDir:t.TempDir(),AdsSecret:secret,
		GreetingSecret:strings.Repeat("G",40),
		Core:core.Config{AdsUUID:"ads",GreetingUUID:"greet",AdsTriggers:[]string{"ads"}},
		OnDecision:func(d core.Decision)error{got=d;return nil},
	})
	if err!=nil{t.Fatal(err)}
	body:=`{"event":"Message","instanceId":"ads","data":{"Info":{"ID":"stop-1","Chat":"5548999999999@s.whatsapp.net"},"Message":{"conversation":"PARAR"}}}`
	w:=httptest.NewRecorder()
	server.ServeHTTP(w,httptest.NewRequest(http.MethodPost,"/webhook/ads/"+secret,strings.NewReader(body)))
	if w.Code!=202||got.Reason!="opt_out_requires_persistent_suppression"||got.PhoneE164==""{
		t.Fatalf("opt-out was discarded: status=%d decision=%+v",w.Code,got)
	}
}
