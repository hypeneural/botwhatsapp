package relay

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxWebhookBytes = 1 << 20
type Config struct{
	Enabled bool
	ExpectedInstanceUUID string
	Secret string
	StatusURL string
	BotURL string
}
type Server struct{
	Cfg Config
	Store *Store
	HTTP *http.Client
	queue chan string
}
func validSink(s string)bool{
	u,e:=url.Parse(s)
	return e==nil&&(u.Scheme=="http"||u.Scheme=="https")&&u.Host!=""&&u.User==nil&&u.Fragment==""&&u.RawQuery==""
}
func New(cfg Config,store *Store)(*Server,error){
	if store==nil {return nil,errors.New("relay: durable store is required")}
	if cfg.Enabled {
		if len(cfg.Secret)<32||cfg.ExpectedInstanceUUID==""||!validSink(cfg.StatusURL)||!validSink(cfg.BotURL)||cfg.StatusURL==cfg.BotURL{
			return nil,errors.New("relay: invalid production configuration")
		}
	}
	return &Server{
		Cfg:cfg,Store:store,queue:make(chan string,1024),
		HTTP:&http.Client{Timeout:7*time.Second,
			CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}},
	},nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter,r *http.Request){
	switch r.URL.Path{
	case "/healthz":
		if r.Method!=http.MethodGet{w.WriteHeader(http.StatusMethodNotAllowed);return}
		w.WriteHeader(http.StatusOK);return
	case "/readyz":
		if r.Method!=http.MethodGet{w.WriteHeader(http.StatusMethodNotAllowed);return}
		if !s.Cfg.Enabled {http.Error(w,"relay disabled",503);return}
		w.WriteHeader(http.StatusOK);return
	}
	if r.Method!=http.MethodPost {http.NotFound(w,r);return}
	if !s.Cfg.Enabled {http.Error(w,"relay disabled",503);return}
	// Credentials are a URL capability only inside the restricted Docker network.
	// Never log the URL, secret, the payload or errors containing either.
	secretPart:=strings.TrimPrefix(r.URL.Path,"/webhook/evolution/")
	if secretPart==r.URL.Path||secretPart==""||strings.Contains(secretPart,"/")||len(secretPart)>256{
		http.Error(w,"unauthorized",401);return
	}
	got,want:=sha256.Sum256([]byte(secretPart)),sha256.Sum256([]byte(s.Cfg.Secret))
	if subtle.ConstantTimeCompare(got[:],want[:])!=1{
		http.Error(w,"unauthorized",401);return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"),"application/json"){
		http.Error(w,"expected JSON",415);return
	}
	r.Body=http.MaxBytesReader(w,r.Body,maxWebhookBytes)
	body,err:=io.ReadAll(r.Body)
	if err!=nil{http.Error(w,"invalid payload size",413);return}
	key,err:=EventKey(body,s.Cfg.ExpectedInstanceUUID)
	if err!=nil {http.Error(w,"invalid event",400);return}
	inserted,err:=s.Store.Insert(key,body)
	if err!=nil {
		http.Error(w,"durable relay temporarily unavailable",503);return
	}
	if inserted {
		select{case s.queue<-key:default:}
	}
	w.WriteHeader(http.StatusAccepted)
}
func(s *Server) post(ctx context.Context,target string,body []byte,key string)bool{
	request,err:=http.NewRequestWithContext(ctx,http.MethodPost,target,bytes.NewReader(body))
	if err!=nil{return false}
	request.Header.Set("Content-Type","application/json")
	request.Header.Set("X-Relay-Event-Key",key)
	resp,err:=s.HTTP.Do(request)
	if err!=nil{return false}
	defer resp.Body.Close()
	_,_=io.Copy(io.Discard,io.LimitReader(resp.Body,4096))
	return resp.StatusCode>=200&&resp.StatusCode<300
}
// Process enforces STATUS FIRST before BOT. Primary status failures hold
// the secondary bot until the original consumer has accepted the event.
// A successful 2xx is only a downstream ACK, not a WhatsApp delivery receipt.
func(s *Server) Process(ctx context.Context,key string)error{
	rec,err:=s.Store.Get(key);if err!=nil{return err}
	if !rec.CompletedAt.IsZero(){return nil}
	if !rec.NextAttempt.IsZero()&&time.Now().Before(rec.NextAttempt){return nil}
	if len(rec.Body)==0{return errors.New("relay: incomplete record without payload")}
	if !rec.StatusDone {
		if !s.post(ctx,s.Cfg.StatusURL,rec.Body,key){
			return s.Store.Postpone(key,time.Now().Add(15*time.Second))
		}
		rec,err=s.Store.UpdateSink(key,true,time.Now())
		if err!=nil{return err}
	}
	if !rec.BotDone{
		if !s.post(ctx,s.Cfg.BotURL,rec.Body,key){
			return s.Store.Postpone(key,time.Now().Add(15*time.Second))
		}
		_,err=s.Store.UpdateSink(key,false,time.Now())
		return err
	}
	return nil
}
func(s *Server) Drain(ctx context.Context)error{
	keys,err:=s.Store.Pending(time.Now())
	if err!=nil{return err}
	for _,key:=range keys{
		if ctx.Err()!=nil{return ctx.Err()}
		if err=s.Process(ctx,key);err!=nil{return err}
	}
	return nil
}
func(s *Server) Run(ctx context.Context,onError func(error)){
	if err:=s.Drain(ctx);err!=nil{onError(err)}
	recovery:=time.NewTicker(30*time.Second)
	maintenance:=time.NewTicker(12*time.Hour)
	defer recovery.Stop();defer maintenance.Stop()
	for {
		select{
		case <-ctx.Done():return
		case key:=<-s.queue:
			if err:=s.Process(ctx,key);err!=nil{onError(err)}
		case <-recovery.C:
			if err:=s.Drain(ctx);err!=nil{onError(err)}
		case <-maintenance.C:
			// Tombstones live 7 days. Never prune unacknowledged events.
			if _,err:=s.Store.PruneCompleted(time.Now().Add(-7*24*time.Hour));err!=nil{onError(err)}
		}
	}
}
