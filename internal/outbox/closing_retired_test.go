package outbox

import (
    "context"
    "strings"
    "testing"

    "github.com/hypeneural/botwhatsapp/internal/workflow"
)

// Old spools can contain a text stage created by a previously deployed
// image. It must not send an obsolete handoff message or prevent CRM work.
func TestQueuedLegacyClosingNeverSendsAndCRMStillRuns(t *testing.T) {
    store, _ := setupTestStore(t)
    evo := &mockEvolutionSender{}
    crmClient := &mockCRMScheduler{}
    instance := 1
    w, err := NewWorker(Config{OperationalAuthorization: true,
        CRMInstanceID: &instance, Senders: map[string]EvolutionSender{"ads-old":evo},
    }, store, evo, crmClient)
    if err != nil { t.Fatal(err) }
    phone := "5548999991234"
    old := Task{ID:"old-closing", InstanceID:"ads-old", PhoneE164:phone,
        RecipientJID: phone+"@s.whatsapp.net", CRMPhone:"48999991234",
        Step:workflow.SendAdClosingText,
        PayloadText:"Outro WhatsApp entrará em contato"}
    next := Task{ID:"old-following-crm",PreviousTaskID:old.ID,
        InstanceID:"ads-old",PhoneE164:phone,CRMPhone:"48999991234",
        RecipientJID:phone+"@s.whatsapp.net",Step:workflow.ScheduleNatal}
    if err:=store.EnqueueTask(old);err!=nil {t.Fatal(err)}
    if err:=store.EnqueueTask(next);err!=nil {t.Fatal(err)}
    if err:=w.ProcessTask(context.Background(),old.ID);err!=nil {t.Fatal(err)}
    oldState,err:=store.GetTask(old.ID)
    if err!=nil || oldState.State!=TaskSuppressed ||
        !strings.Contains(oldState.Error,"retired"){
        t.Fatalf("legacy outbound step still active: %+v %v",oldState,err)
    }
    if err:=w.ProcessTask(context.Background(),next.ID);err!=nil {t.Fatal(err)}
    if evo.textCalls!=0 {t.Fatalf("legacy text sent %d times",evo.textCalls)}
    if crmClient.calls!=1 {t.Fatalf("CRM must remain eligible, got %d",crmClient.calls)}
}
