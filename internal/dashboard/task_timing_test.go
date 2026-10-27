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

func TestAttemptStartedUsesTaskTimestampNotOldInbound(t *testing.T) {
    dir := t.TempDir()
    if err := os.Mkdir(filepath.Join(dir, "tasks"), 0700); err != nil { t.Fatal(err) }
    now := time.Now().UTC()
    key := strings.Repeat("f",64)
    writeIdentityFixture(t,dir,key,core.Decision{
        Action:core.AdsProposal, InstanceID:"ads", MessageID:"test-timing",
        ReplyJID:"5548999990001@s.whatsapp.net", PhoneE164:"5548999990001",
    },now.Add(-25*time.Minute))
    name := key+"-05-schedule_natal_crm"
    task := outbox.Task{
        ID:name,EventKey:key,Step:workflow.ScheduleNatal,PhoneE164:"5548999990001",
        State:outbox.TaskAttemptStarted,CreatedAt:now.Add(-25*time.Minute),
        AttemptStartedAt:now.Add(-15*time.Second),
    }
    raw,err := json.Marshal(task)
    if err != nil {t.Fatal(err)}
    if err:=os.WriteFile(filepath.Join(dir,"tasks",name+".json"),raw,0600);err!=nil{t.Fatal(err)}
    fresh,err:=NewService(dir).GetEvents(10)
    if err!=nil{t.Fatal(err)}
    if len(fresh.Events)!=1 || fresh.Events[0].Status!="em_processamento" {
        t.Fatalf("recent task should be in flight despite old webhook; got %+v", fresh.Events)
    }
    task.AttemptStartedAt=now.Add(-4*time.Minute)
    raw,err=json.Marshal(task)
    if err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(filepath.Join(dir,"tasks",name+".json"),raw,0600);err!=nil{t.Fatal(err)}
    stale,err:=NewService(dir).GetEvents(10)
    if err!=nil{t.Fatal(err)}
    if len(stale.Events)!=1 || stale.Events[0].Status!="incerto" || !stale.Events[0].NeedsAttention {
        t.Fatalf("stalled task should be uncertain; got %+v",stale.Events)
    }
}
