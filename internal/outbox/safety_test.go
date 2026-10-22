package outbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/instances"
	"github.com/hypeneural/botwhatsapp/internal/workflow"
)

func TestNeverSendMissingMediaFixtures(t *testing.T){
	store,_:=setupTestStore(t)
	evo:=&mockEvolutionSender{}
	crm:=&mockCRMScheduler{}
	mid:=1
	w,err:=NewWorker(Config{CRMInstanceID:&mid,OperationalAuthorization:true},
		store,evo,crm)
	if err!=nil{t.Fatal(err)}
	d:=core.Decision{Action:core.AdsProposal,InstanceID:"ads",MessageID:"no-media",
		PhoneE164:"5548999999999",CRMPhone:"48999999999",
		ReplyJID:"5548999999999@s.whatsapp.net"}
	if err:=w.IngestDecision(d);err!=nil{t.Fatal(err)}
	_,err=w.DrainPending(context.Background())
	if err!=nil{t.Fatal(err)}
	if evo.mediaCalls!=0{t.Fatalf("sent a fake audio: %d",evo.mediaCalls)}
	if crm.calls!=1{t.Fatalf("missing audio blocked CRM: %d",crm.calls)}
	d=core.Decision{Action:core.Greeting,InstanceID:"greet",MessageID:"no-image",
		PhoneE164:"5548999998888",ReplyJID:"5548999998888@s.whatsapp.net"}
	if err:=w.IngestDecision(d);err!=nil{t.Fatal(err)}
	_,err=w.DrainPending(context.Background())
	if err!=nil{t.Fatal(err)}
	if evo.mediaCalls!=0{t.Fatalf("sent fake greeting image: %d",evo.mediaCalls)}
}
func TestSenderRoutingStrictByInstance(t *testing.T){
	store,_:=setupTestStore(t)
	ads:=&mockEvolutionSender{}
	greet:=&mockEvolutionSender{}
	w,err:=NewWorker(Config{
		Senders:map[string]EvolutionSender{"ads":ads,"greeting":greet},
		GreetingImageFixture:[]byte("fake image used only by mock"),
	},store,nil,nil)
	if err!=nil{t.Fatal(err)}
	a:=core.Decision{Action:core.AdsProposal,InstanceID:"ads",MessageID:"isolated-ads",
		PhoneE164:"5548999999999",CRMPhone:"48999999999",
		ReplyJID:"5548999999999@s.whatsapp.net"}
	g:=core.Decision{Action:core.Greeting,InstanceID:"greeting",MessageID:"isolated-greet",
		PhoneE164:"5548999998888",ReplyJID:"5548999998888@s.whatsapp.net"}
	if err:=w.IngestDecision(a);err!=nil{t.Fatal(err)}
	if err:=w.IngestDecision(g);err!=nil{t.Fatal(err)}
	_,err=w.DrainPending(context.Background());if err!=nil{t.Fatal(err)}
	if ads.textCalls!=1||ads.mediaCalls!=0||greet.textCalls!=0||greet.mediaCalls!=1{
		t.Fatalf("wrong account sender used: ads text=%d/media=%d greet text=%d/media=%d",
			ads.textCalls,ads.mediaCalls,greet.textCalls,greet.mediaCalls)
	}
}
func TestCorruptContactStateFailsClosed(t *testing.T){
	store,_:=setupTestStore(t)
	phone:="5548999999999"
	c:=ContactState{PhoneE164:phone,OptOut:true}
	if err:=store.SaveContact(c);err!=nil{t.Fatal(err)}
	entries,err:=os.ReadDir(store.contactsDir)
	if err!=nil||len(entries)!=1{t.Fatal(err)}
	path:=filepath.Join(store.contactsDir,entries[0].Name())
	if err:=os.WriteFile(path,[]byte("{not json"),0600);err!=nil{t.Fatal(err)}
	if _,err:=store.GetContact(phone);err==nil{t.Fatal("corrupt opt-out reset to new contact")}
	w,_:=NewWorker(Config{},store,&mockEvolutionSender{},nil)
	d:=core.Decision{Action:core.Greeting,InstanceID:"greet",MessageID:"policy-error",
		PhoneE164:phone,ReplyJID:phone+"@s.whatsapp.net"}
	if err:=w.IngestDecision(d);err==nil{t.Fatal("corrupt policy allowed greeting ingestion")}
}
func TestEventBatchRollbackAndRetry(t *testing.T){
	store,_:=setupTestStore(t)
	eventKey,err:=instances.EventKey("ads","Message","safe-msg")
	if err!=nil{t.Fatal(err)}
	tasks:=[]Task{
		{ID:eventKey+"-00-text",EventKey:eventKey,InstanceID:"ads",Step:workflow.SendAdText,
			CreatedAt:time.Now()},
		{ID:"../invalid",EventKey:eventKey,InstanceID:"ads",Step:workflow.SendAdAudio},
	}
	ok,err:=store.CommitEventTasks("ads","Message","safe-msg",tasks)
	if ok||err==nil{t.Fatal("unsafe task batch committed")}
	entries,err:=os.ReadDir(store.tasksDir)
	if err!=nil||len(entries)!=0{t.Fatal("partial task batch visible")}
	tasks[1].ID=eventKey+"-01-audio"
	ok,err=store.CommitEventTasks("ads","Message","safe-msg",tasks)
	if err!=nil||!ok{t.Fatal("retry should recover",err)}
	ok,err=store.CommitEventTasks("ads","Message","safe-msg",tasks)
	if err!=nil||ok{t.Fatal("duplicate event should be blocked",err)}
	pending,_,err:=store.RecoverTasks()
	if err!=nil||len(pending)!=2{t.Fatalf("batch incomplete: %d %v",len(pending),err)}
	if !strings.HasPrefix(pending[0].ID,eventKey){t.Fatal("event IDs must be deterministic")}
}
