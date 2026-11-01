package outbox

import (
	"context"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

func TestHumanInitiatedConversationSuppressesGreetingForFiveDays(t *testing.T) {
	store, _ := setupTestStore(t)
	evo := &mockEvolutionSender{}
	worker, err := NewWorker(Config{
		GreetingCaption: "Saudacao de teste", GreetingCooldown: 24*time.Hour,
	}, store, evo, nil)
	if err != nil { t.Fatal(err) }
	phone := "5548999991000"
	outbound := core.Decision{
		Action:core.OutboundObserved, InstanceID:"greeting", MessageID:"own-1",
		PhoneE164:phone, ReplyJID:phone+"@s.whatsapp.net",
	}
	if err := worker.IngestDecision(outbound); err != nil { t.Fatal(err) }
	state, err := store.GetContact(phone)
	if err != nil || state.LastOutboundAt.IsZero() {t.Fatalf("outbound not recorded: %+v %v",state,err)}
	reply := core.Decision{
		Action:core.Greeting, InstanceID:"greeting", MessageID:"customer-reply",
		PhoneE164:phone, ReplyJID:phone+"@s.whatsapp.net",
	}
	if err := worker.IngestDecision(reply); err != nil {t.Fatal(err)}
	pending, _, err := store.RecoverTasks()
	if err != nil || len(pending) != 0 {t.Fatalf("should not greet recent human conversation: %v %+v",err,pending)}
	if _,err := worker.DrainPending(context.Background());err!=nil{t.Fatal(err)}
	if evo.textCalls!=0||evo.mediaCalls!=0 {t.Fatal("studio-initiated conversation was auto-greeted")}
}

func TestOrdinaryInboundActivityPreventsDuplicateGreeting(t *testing.T) {
	store,_ := setupTestStore(t)
	worker,err := NewWorker(Config{},store,&mockEvolutionSender{},nil)
	if err!=nil {t.Fatal(err)}
	phone := "5548999991001"
	if err:=worker.IngestDecision(core.Decision{Action:core.Ignore,Reason:"ads_unmatched_no_automation",
		PhoneE164:phone,InstanceID:"ads",MessageID:"organic"});err!=nil {t.Fatal(err)}
	if err:=worker.IngestDecision(core.Decision{Action:core.Greeting,
		PhoneE164:phone,InstanceID:"greeting",MessageID:"reply"});err!=nil {t.Fatal(err)}
	got,_,err:=store.RecoverTasks()
	if err!=nil||len(got)!=0{t.Fatalf("recent organic inbound should suppress greet: %v %+v",err,got)}
}

func TestQueuedGreetingCancelledWhenStudioStartsConversation(t *testing.T) {
	store,_:=setupTestStore(t)
	evo:=&mockEvolutionSender{}
	w,err:=NewWorker(Config{GreetingCaption:"Saudacao",GreetingCooldown:24*time.Hour},store,evo,nil)
	if err!=nil {t.Fatal(err)}
	phone:="5548999991002"
	d:=core.Decision{Action:core.Greeting,InstanceID:"greeting",MessageID:"in1",
		PhoneE164:phone,ReplyJID:phone+"@s.whatsapp.net"}
	if err:=w.IngestDecision(d);err!=nil{t.Fatal(err)}
	queued,_,err:=store.RecoverTasks()
	if err!=nil||len(queued)!=1{t.Fatalf("greeting task should be queued: %+v %v",queued,err)}
	if err:=w.IngestDecision(core.Decision{Action:core.OutboundObserved,InstanceID:"greeting",
		MessageID:"own-after",PhoneE164:phone});err!=nil{t.Fatal(err)}
	if _,err:=w.DrainPending(context.Background());err!=nil{t.Fatal(err)}
	if evo.textCalls!=0||evo.mediaCalls!=0{t.Fatal("in-flight manual conversation must cancel queued greeting")}
}

func TestFiveDayActivityBoundaryStillAllowsFreshGreeting(t *testing.T) {
	store,_:=setupTestStore(t)
	evo:=&mockEvolutionSender{}
	w,err:=NewWorker(Config{GreetingCaption:"Saudacao",GreetingCooldown:24*time.Hour},store,evo,nil)
	if err!=nil {t.Fatal(err)}
	phone:="5548999991003"
	_,err=store.UpdateContact(phone,func(c *ContactState)error{
		c.LastInboundAt=time.Now().Add(-121*time.Hour)
		c.LastOutboundAt=time.Now().Add(-121*time.Hour)
		c.LastActivityAt=time.Now().Add(-121*time.Hour)
		return nil
	})
	if err!=nil {t.Fatal(err)}
	if err:=w.IngestDecision(core.Decision{Action:core.Greeting,InstanceID:"greeting",
		MessageID:"new-after-5d",PhoneE164:phone,ReplyJID:phone+"@s.whatsapp.net"});err!=nil{t.Fatal(err)}
	tasks,_,err:=store.RecoverTasks()
	if err!=nil||len(tasks)!=1{t.Fatalf("first fresh inbound should get greeting: %+v %v",tasks,err)}
}
