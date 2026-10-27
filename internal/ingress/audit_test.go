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

func TestInboundAuditMetadataAndNoSecrets(t *testing.T) {
	body:=[]byte(`{"event":"Message","instanceId":"ads","instanceToken":"NEVER_PERSIST_SECRET","data":{"Info":{"ID":"e1","Chat":"5548999999999@s.whatsapp.net"},"Message":{"extendedTextMessage":{"text":"Quero conhecer os pacotes de Natal em Tijucas","contextInfo":{"externalAdReply":{"sourceId":"123","thumbnail":"secret_media"}}}}}}`)
	env,err:=core.Parse(body)
	if err!=nil {t.Fatal(err)}
	d:=core.Select(env,core.Config{AdsUUID:"ads",AdsTriggers:[]string{"Quero conhecer os pacotes de Natal em Tijucas"}})
	got:=auditInbound(body,d)
	if got.Source!="meta_ads" || !got.HasAdContext || !got.TriggerMatch ||
		got.TriggerText!="Quero conhecer os pacotes de Natal em Tijucas" || got.MessageType!="texto_estendido" {
		t.Fatalf("wrong CTWA attribution %+v",got)
	}
	b,_:=json.Marshal(got)
	for _, private:=range []string{"NEVER_PERSIST_SECRET","secret_media","thumbnail","sourceId"} {
		if strings.Contains(string(b),private) {t.Fatalf("stored webhook private metadata %q",private)}
	}
}

func TestNormalInboundAndIgnoredMediaAreDurablyRecorded(t *testing.T) {
	secret:=strings.Repeat("x",40)
	dir:=t.TempDir()
	s,err:=New(Settings{
		Enabled:true,SpoolDir:dir, AdsSecret:secret, GreetingSecret:strings.Repeat("y",40),
		Core:core.Config{AdsUUID:"ads",GreetingUUID:"greet",AdsTriggers:[]string{"quero proposta"}},
	})
	if err!=nil {t.Fatal(err)}
	cases:=[]struct{id,body,source,kind,status string}{
		{"normal",`{"event":"Message","instanceId":"ads","data":{"Info":{"ID":"normal","Chat":"5548999999999@s.whatsapp.net"},"Message":{"conversation":"Bom dia"}}}`,"normal","texto","received"},
		{"media",`{"event":"Message","instanceId":"ads","data":{"Info":{"ID":"media","Chat":"5548999999999@s.whatsapp.net"},"Message":{"imageMessage":{"url":"TOP_SECRET_MEDIA"}}}}`,"normal","imagem","received"},
	}
	for _,tc:=range cases {
		for i:=0;i<2;i++ {
			w:=httptest.NewRecorder()
			s.ServeHTTP(w,httptest.NewRequest(http.MethodPost,"/webhook/ads/"+secret,strings.NewReader(tc.body)))
			if w.Code!=http.StatusAccepted {t.Fatalf("%s replay=%d got %d: %s",tc.id,i,w.Code,w.Body.String())}
		}
	}
	paths,err:=filepath.Glob(filepath.Join(dir,"*.json"))
	if err!=nil||len(paths)!=2 {t.Fatalf("want two durable inbound events, %v %v",err,paths)}
	for _,p:=range paths {
		raw,err:=os.ReadFile(p);if err!=nil {t.Fatal(err)}
		if strings.Contains(string(raw),"TOP_SECRET_MEDIA"){t.Fatal("stored media payload")}
		var rec struct{Status string `json:"status"`; Inbound InboundAudit `json:"inbound"`}
		if err:=json.Unmarshal(raw,&rec);err!=nil||rec.Status!="received" {t.Fatal(err,rec)}
		if rec.Inbound.Source!="normal" {t.Fatal("normal message misattributed to advertising")}
	}
}

func TestAuditExcerptBoundsAndControls(t *testing.T){
	msg:=strings.Repeat("❤",400)+"\u0000"
	out:=auditExcerpt(msg)
	if len([]rune(out))!=280 || strings.ContainsRune(out,'\x00') {t.Fatal("invalid excerpt",len([]rune(out)))}
}
