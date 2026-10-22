package relay

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)
const evt1=`{"event":"Message","instanceId":"uuid-ads","data":{"Info":{"ID":"abc","Chat":"5548999999999@s.whatsapp.net"},"Message":{"conversation":"ola"}}}`
const evt2=`{"event":"Message","instanceId":"uuid-ads","data":{"Info":{"ID":"def","Chat":"5548999999999@s.whatsapp.net"},"Message":{"conversation":"ola"}}}`

func newRelayTest(t *testing.T,status,bot string,max int)(*Server,*Store,string){
	t.Helper()
	store,err:=NewStore(t.TempDir(),max)
	if err!=nil{t.Fatal(err)}
	secret:=strings.Repeat("Q",40)
	s,err:=New(Config{Enabled:true,ExpectedInstanceUUID:"uuid-ads",
		Secret:secret,StatusURL:status,BotURL:bot},store)
	if err!=nil{t.Fatal(err)}
	return s,store,secret
}
func sendIngress(t *testing.T,s *Server,path,body string)int{
	t.Helper()
	req:=httptest.NewRequest("POST",path,strings.NewReader(body))
	req.Header.Set("Content-Type","application/json")
	rec:=httptest.NewRecorder();s.ServeHTTP(rec,req)
	return rec.Code
}
func TestDurableFanoutAndRestartDedup(t *testing.T){
	var statusCalls,botCalls atomic.Int32
	status:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		statusCalls.Add(1)
		if r.Header.Get("X-Relay-Event-Key")==""{t.Error("missing delivery key")}
		var m map[string]any
		if json.NewDecoder(r.Body).Decode(&m)!=nil{t.Error("malformed forwarded body")}
		w.WriteHeader(204)
	}));defer status.Close()
	bot:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		botCalls.Add(1)
		w.WriteHeader(202)
	}));defer bot.Close()
	s,store,secret:=newRelayTest(t,status.URL,bot.URL,10)
	url:="/webhook/evolution/"+secret
	if c:=sendIngress(t,s,url,evt1);c!=202{t.Fatal(c)}
	if c:=sendIngress(t,s,url,evt1);c!=202{t.Fatal("duplicate was rejected",c)}
	key,_:=EventKey([]byte(evt1),"uuid-ads")
	if err:=s.Process(context.Background(),key);err!=nil{t.Fatal(err)}
	if err:=s.Process(context.Background(),key);err!=nil{t.Fatal(err)}
	if statusCalls.Load()!=1||botCalls.Load()!=1{t.Fatal("unexpected duplicate sends",statusCalls.Load(),botCalls.Load())}
	rec,err:=store.Get(key)
	if err!=nil||rec.CompletedAt.IsZero()||len(rec.Body)!=0{t.Fatal("completed record must scrub PII",err)}
	// Restart creates new store, the tombstone must block the same event.
	fresh,err:=NewStore(store.dir,10);if err!=nil{t.Fatal(err)}
	inserted,err:=fresh.Insert(key,[]byte(evt1))
	if inserted||err!=nil{t.Fatal("duplicate accepted across restart",err)}
	deleted,err:=fresh.PruneCompleted(time.Now().Add(time.Hour))
	if err!=nil||deleted!=1{t.Fatal(deleted,err)}
}
func TestPrimaryStatusFirstAndIndependentRetries(t *testing.T){
	var statusCalls,botCalls atomic.Int32
	statusFail:=atomic.Bool{};statusFail.Store(true)
	botFail:=atomic.Bool{};botFail.Store(true)
	status:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		statusCalls.Add(1);if statusFail.Load(){w.WriteHeader(503)}else{w.WriteHeader(200)}
	}));defer status.Close()
	bot:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		botCalls.Add(1);if botFail.Load(){w.WriteHeader(503)}else{w.WriteHeader(204)}
	}));defer bot.Close()
	s,store,secret:=newRelayTest(t,status.URL,bot.URL,10)
	if sendIngress(t,s,"/webhook/evolution/"+secret,evt1)!=202{t.Fatal("failed durable ingress")}
	key,_:=EventKey([]byte(evt1),"uuid-ads")
	if err:=s.Process(context.Background(),key);err!=nil{t.Fatal(err)}
	if statusCalls.Load()!=1||botCalls.Load()!=0{t.Fatal("bot must wait for original status engine")}
	// Backoff blocks rapid repeated retries.
	if err:=s.Process(context.Background(),key);err!=nil{t.Fatal(err)}
	if statusCalls.Load()!=1{t.Fatal("backoff ignored")}
	statusFail.Store(false)
	if err:=store.Postpone(key,time.Now().Add(-time.Second));err!=nil{t.Fatal(err)}
	if err:=s.Process(context.Background(),key);err!=nil{t.Fatal(err)}
	if statusCalls.Load()!=2||botCalls.Load()!=1{t.Fatal("did not attempt bot after status ACK")}
	rec,_:=store.Get(key)
	if !rec.StatusDone||rec.BotDone{t.Fatal("independent status ack lost",rec)}
	botFail.Store(false)
	if err:=store.Postpone(key,time.Now().Add(-time.Second));err!=nil{t.Fatal(err)}
	if err:=s.Process(context.Background(),key);err!=nil{t.Fatal(err)}
	if statusCalls.Load()!=2||botCalls.Load()!=2{t.Fatal("primary was resent after confirmed",statusCalls.Load(),botCalls.Load())}
}
func TestInvalidAndCapacityFailClosed(t *testing.T){
	status:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(204)}));defer status.Close()
	bot:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(204)}));defer bot.Close()
	s,store,secret:=newRelayTest(t,status.URL,bot.URL,1)
	if sendIngress(t,s,"/webhook/evolution/wrong",evt1)!=401{t.Fatal("unauthenticated")}
	if sendIngress(t,s,"/webhook/evolution/"+secret+"/more",evt1)!=401{t.Fatal("extra path accepted")}
	if sendIngress(t,s,"/webhook/evolution/"+secret,strings.Replace(evt1,"uuid-ads","another",1))!=400{t.Fatal("wrong instance accepted")}
	if sendIngress(t,s,"/webhook/evolution/"+secret,evt1)!=202{t.Fatal("not accepted")}
	if sendIngress(t,s,"/webhook/evolution/"+secret,evt2)!=503{t.Fatal("full spool must backpressure")}
	id,_:=EventKey([]byte(evt1),"uuid-ads")
	if inserted,err:=store.Insert(id,[]byte(evt1));inserted||err!=nil{t.Fatal("duplicate must be safe even when at capacity",err)}
	if _,err:=EventKey([]byte(evt1),"different");!errors.Is(err,ErrInvalid){t.Fatal(err)}
	if deleted,err:=store.PruneCompleted(time.Now().Add(time.Hour));err!=nil||deleted!=0{t.Fatal("pending event pruned",deleted,err)}
}
