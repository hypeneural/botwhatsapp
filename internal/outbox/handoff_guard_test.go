package outbox

import (
	"context"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

func TestQueuedGreetingsRecheckCooldownAtSendTime(t *testing.T) {
	store,_:=setupTestStore(t)
	sender:=&mockEvolutionSender{}
	worker,err:=NewWorker(Config{
		GreetingImageFixture:[]byte("synthetic image only in mock"),
		GreetingCooldown:24*time.Hour,
	},store,sender,nil)
	if err!=nil{t.Fatal(err)}
	d:=core.Decision{
		Action:core.Greeting, InstanceID:"greeting", MessageID:"g1",
		PhoneE164:"5548999999999", ReplyJID:"5548999999999@s.whatsapp.net",
	}
	if err:=worker.IngestDecision(d);err!=nil{t.Fatal(err)}
	d.MessageID="g2"
	if err:=worker.IngestDecision(d);err!=nil{t.Fatal(err)}
	_,err=worker.DrainPending(context.Background())
	if err!=nil{t.Fatal(err)}
	if sender.mediaCalls!=1{t.Fatalf("expected only one image despite queued greetings; got %d",sender.mediaCalls)}
}

func TestHumanHandoffStopsEarlierPendingMessages(t *testing.T){
	store,_:=setupTestStore(t)
	sender:=&mockEvolutionSender{}
	worker,err:=NewWorker(Config{
		GreetingImageFixture:[]byte("synthetic image only in mock"),
	},store,sender,nil)
	if err!=nil{t.Fatal(err)}
	d:=core.Decision{
		Action:core.Greeting, InstanceID:"greeting", MessageID:"greet-before-human",
		PhoneE164:"5548999999999", ReplyJID:"5548999999999@s.whatsapp.net",
	}
	if err:=worker.IngestDecision(d);err!=nil{t.Fatal(err)}
	d.Action=core.MenuOption
	d.Option="5"
	d.MessageID="pause-now"
	if err:=worker.IngestDecision(d);err!=nil{t.Fatal(err)}
	c,err:=store.GetContact(d.PhoneE164)
	if err!=nil||!c.HumanPaused{t.Fatal("human pause was not persisted at ingress",err)}
	_,err=worker.DrainPending(context.Background())
	if err!=nil{t.Fatal(err)}
	if sender.mediaCalls!=0{t.Fatalf("queued greeting reached a human-paused customer: %d",sender.mediaCalls)}
	if sender.textCalls!=1{t.Fatalf("human handoff acknowledgment not sent once: %d",sender.textCalls)}
}
