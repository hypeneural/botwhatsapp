package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/outbox"
	"github.com/hypeneural/botwhatsapp/internal/workflow"
)

func TestOperationalEventTimelineJoinsOutbox(t *testing.T) {
	dir:=t.TempDir()
	if err:=os.MkdirAll(filepath.Join(dir,"tasks"),0700);err!=nil {t.Fatal(err)}
	key:=strings.Repeat("a",64)
	received:=time.Date(2026,10,27,12,5,0,0,time.UTC)
	rec:=map[string]interface{}{
		"event_key":key,"received_at":received.Format(time.RFC3339Nano),"status":"received",
		"decision":map[string]interface{}{"action":"ads_proposal","message_id":"msg1","phone_e164":"5548999999999","name":"Ana<script>alert(1)</script>"},
		"inbound":map[string]interface{}{"message_type":"texto","text":"Quero conhecer os pacotes de Natal em Tijucas",
			"source":"meta_ads","trigger_match":true,"trigger_text":"Quero conhecer os pacotes de Natal em Tijucas"},
	}
	raw,_:=json.Marshal(rec)
	if err:=os.WriteFile(filepath.Join(dir,key+".json"),raw,0600);err!=nil{t.Fatal(err)}
	tasks:=[]outbox.Task{
		{ID:key+"-00-send_ad_text",EventKey:key,Step:workflow.SendAdText,PhoneE164:"5548999999999",State:outbox.TaskConfirmed,ExternalID:"evo-123",AttemptStartedAt:received,CompletedAt:received.Add(time.Second)},
		{ID:key+"-01-schedule_natal_crm",EventKey:key,Step:workflow.ScheduleNatal,PhoneE164:"5548999999999",State:outbox.TaskSuppressed,Error:"crm: HTTP 422 rejected",AttemptStartedAt:received.Add(time.Second),CompletedAt:received.Add(2*time.Second)},
	}
	for _,task:=range tasks{data,_:=json.Marshal(task);if err:=os.WriteFile(filepath.Join(dir,"tasks",task.ID+".json"),data,0600);err!=nil{t.Fatal(err)}}
	svc:=NewService(dir)
	result,err:=svc.GetEvents(5)
	if err!=nil {t.Fatal(err)}
	if len(result.Events)!=1||len(result.Events[0].Steps)!=2 {t.Fatalf("history %v %v",err,result)}
	e:=result.Events[0]
	if e.Status!="crm_rejeitado"||e.Source!="meta_ads"||e.ReceivedAtBRT!="27/10/2026 09:05:00" {
		t.Fatalf("unexpected event %+v",e)
	}
	if e.Steps[0].APIResponse!="API aceitou (entrega não comprovada)" ||
	   e.Steps[1].Reason!="HTTP 422: rejeição do contrato/regra comercial" {t.Fatalf("wrong step responses %+v",e.Steps)}
	if result.Stats.CRMRejected!=1||result.Stats.MetaAds!=1||result.Stats.Received!=1 {t.Fatalf("wrong stats %+v",result.Stats)}
	if safeEventReason("https://example.com/?token=VERY_PRIVATE_SECRET")=="https://example.com/?token=VERY_PRIVATE_SECRET" {t.Fatal("raw provider errors leaked")}
}

func TestEventEndpointAuthAndLimit(t *testing.T){
	s:=NewHandler(t.TempDir(),"unit-test-secret")
	req:=httptest.NewRequest(http.MethodGet,"/api/dashboard/events?limit=100000",nil)
	w:=httptest.NewRecorder()
	s.ServeHTTP(w,req)
	if w.Code!=http.StatusUnauthorized {t.Fatal(w.Code)}
	req=httptest.NewRequest(http.MethodGet,"/api/dashboard/events?limit=100000",nil)
	req.SetBasicAuth("operator","unit-test-secret")
	w=httptest.NewRecorder()
	s.ServeHTTP(w,req)
	if w.Code!=http.StatusOK || w.Header().Get("Cache-Control")!="no-store" {t.Fatal(w.Code)}
	var result EventResult
	if err:=json.Unmarshal(w.Body.Bytes(),&result);err!=nil || result.Limit!=500 {t.Fatal(err,result.Limit)}
}

func TestLegacyHTTP422IsClassifiedAsRejected(t *testing.T) {
	task:=outbox.Task{State:outbox.TaskUncertain,Error:"crm: HTTP 422 rejected"}
	if got:=outcome(task);got!="API recusou (HTTP 422 legado)" {
		t.Fatalf("incorrect historical response: %s",got)
	}
	if got:=safeEventReason(task.Error);got!="HTTP 422: rejeição do contrato/regra comercial" {
		t.Fatalf("incorrect 422 explanation: %s",got)
	}
}
