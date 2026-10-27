package dashboard

import (
  "encoding/json"
  "os"
  "path/filepath"
  "strings"
  "testing"
  "time"

  "github.com/hypeneural/botwhatsapp/internal/core"
  "github.com/hypeneural/botwhatsapp/internal/outbox"
  "github.com/hypeneural/botwhatsapp/internal/workflow"
)

func TestManagerLatestCRMOutcomeDoesNotInheritOlderSuccess(t *testing.T) {
	now:=time.Now().UTC()
	tasks:=[]outbox.Task{
		{ID:"previous",Step:workflow.ScheduleNatal,State:outbox.TaskConfirmed,
		ExternalID:"proposal-dispatch-11111111-2222-4333-8444-555555555555",
		CreatedAt:now.Add(-time.Hour)},
		{ID:"current",Step:workflow.ScheduleNatal,State:outbox.TaskSuppressed,
		Error:"crm: operational authorization required before dispatch",
		CreatedAt:now.Add(-time.Minute)},
	}
	row:=NewService(t.TempDir()).buildRow(outbox.ContactState{PhoneE164:"5548999999999"},tasks,now)
	if row.CRMScheduleID!="" || !strings.Contains(row.CRMStatus,"Bot bloqueou"){
		t.Fatalf("stale receipt may not mask current failure: %+v",row)
	}
	if row.Channel!="Meta Ads" {t.Fatal("Ads attribution from tasks missing",row.Channel)}
	plain:=NewService(t.TempDir()).buildRow(outbox.ContactState{PhoneE164:"5548999999998"},nil,now)
	if plain.Channel=="Orgânico" {t.Fatal("unknown source was falsely called organic")}
}

func TestForensicLocalBlockAndStalledAdsAreActionable(t *testing.T){
	base:=time.Now().UTC().Add(-15*time.Minute)
	cases:=[]struct{
		id string
		action core.Action
		taskState outbox.TaskState
		errorText string
		want string
		attention bool
	}{
		{"no-task",core.AdsProposal,"","","sem_tarefas",true},
		{"blocked-locally",core.AdsProposal,outbox.TaskSuppressed,"lead already claimed in current window","bloqueado_local",true},
		{"crm-http422",core.AdsProposal,outbox.TaskSuppressed,"crm: HTTP 422 rejected","crm_rejeitado",true},
		{"confirmed-no-receipt",core.AdsProposal,outbox.TaskConfirmed,"","incerto",true},
		{"normal-no-task",core.Ignore,"","","ignorado",false},
	}
	dir:=t.TempDir()
	if err:=os.MkdirAll(filepath.Join(dir,"tasks"),0700);err!=nil {t.Fatal(err)}
	for i,c:=range cases{
		key:=strings.Repeat(string(rune('a'+i)),64)
		if c.id=="no-task"||c.id=="normal-no-task" { /* journal without task */ }
		event:=map[string]interface{}{
			"event_key":key,"received_at":base.Add(time.Duration(i)*time.Second).Format(time.RFC3339Nano),
			"status":"received","decision":map[string]interface{}{
				"action":c.action,"message_id":"msg-"+c.id,"phone_e164":"5548999999999",
			},
			"inbound":map[string]interface{}{"source":"meta_ads","message_type":"texto"},
		}
		raw,_:=json.Marshal(event)
		if err:=os.WriteFile(filepath.Join(dir,key+".json"),raw,0600);err!=nil {t.Fatal(err)}
		if c.taskState!="" {
			task:=outbox.Task{
				ID:key+"-04-schedule_natal_crm",EventKey:key,
				Step:workflow.ScheduleNatal,State:c.taskState,Error:c.errorText,
				PhoneE164:"5548999999999",CreatedAt:base,
			}
			data,_:=json.Marshal(task)
			if err:=os.WriteFile(filepath.Join(dir,"tasks",task.ID+".json"),data,0600);err!=nil {t.Fatal(err)}
		}
	}
	history,err:=NewService(dir).GetEvents(50)
	if err!=nil {t.Fatal(err)}
	for _,c:=range cases{
		found:=false
		for _,e:=range history.Events{
			if e.Decision!=string(c.action) {continue}
			if len(e.Steps)==0 && c.id!="no-task" && c.id!="normal-no-task" {continue}
			if len(e.Steps)>0 && c.taskState=="" {continue}
			if len(e.Steps)>0 && e.Steps[0].State!=string(c.taskState) {continue}
			if c.id=="crm-http422" && e.Steps[0].Reason!="HTTP 422: rejeição do contrato/regra comercial" {continue}
			if c.id=="blocked-locally" && e.Steps[0].Reason!="Cooldown / duplicidade" {continue}
			if e.Status!=c.want || e.NeedsAttention!=c.attention {
				t.Fatalf("%s wanted %s attention %v, got %+v",c.id,c.want,c.attention,e)
			}
			found=true;break
		}
		if !found {t.Fatalf("case not located: %s",c.id)}
	}
}
