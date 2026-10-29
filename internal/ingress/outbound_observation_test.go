package ingress

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

func TestOwnMessageUpdatesConversationButNeverCreatesInboundJournal(t *testing.T) {
	dir:=t.TempDir()
	secret:=strings.Repeat("S",40)
	seen:=0
	srv,err:=New(Settings{Enabled:true,SpoolDir:dir,
		AdsSecret:secret,GreetingSecret:strings.Repeat("T",40),
		Core:core.Config{AdsUUID:"ads",GreetingUUID:"greeting",
			BusinessPhoneE164:"5548996425287",
			AdsTriggers:[]string{"Quero conhecer os pacotes de Natal em Tijucas"}},
		OnDecision:func(d core.Decision)error{
			seen++
			if d.Action!=core.OutboundObserved||d.PhoneE164!="5548999991000"{
				t.Fatalf("own event would be incorrectly routed: %+v",d)
			}
			return nil
		},
	})
	if err!=nil{t.Fatal(err)}
	body:=`{"event":"Message","instanceId":"ads","data":{"Info":{
		"ID":"from-studio","Chat":"5548999991000@s.whatsapp.net",
		"IsFromMe":true},"Message":{"conversation":"Confirme seu horário"}}}`
	w:=httptest.NewRecorder()
	srv.ServeHTTP(w,httptest.NewRequest(http.MethodPost,"/webhook/ads/"+secret,strings.NewReader(body)))
	if w.Code!=http.StatusNoContent||seen!=1{
		t.Fatalf("own event should ACK 204 with durable activity, code=%d seen=%d",w.Code,seen)
	}
	files,err:=filepath.Glob(filepath.Join(dir,"*.json"))
	if err!=nil||len(files)!=0{t.Fatalf("outbound content must not be journaled: %v %v",err,files)}
	entries,err:=os.ReadDir(dir)
	if err!=nil||len(entries)!=0{t.Fatalf("inbound directory unexpectedly modified: %v %v",err,entries)}
}
