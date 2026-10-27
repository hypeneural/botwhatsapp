// Command dashboard-preview serves ONLY synthetic records on loopback.
// It never touches a running bot, CRM, Evolution, production files or network.
package main

import (
	"crypto/sha256"
	"context"
	"errors"
	"os/signal"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/dashboard"
	"github.com/hypeneural/botwhatsapp/internal/outbox"
	"github.com/hypeneural/botwhatsapp/internal/workflow"
)

type previewCase struct {
	phone, name, message, source, errorText string
	action core.Action
	state outbox.TaskState
	received time.Time
}

func makePreviewData(dir string, now time.Time) error {
	taskDir:=filepath.Join(dir,"tasks")
	contactDir:=filepath.Join(dir,"contacts")
	if err:=os.MkdirAll(taskDir,0700);err!=nil{return err}
	if err:=os.MkdirAll(contactDir,0700);err!=nil{return err}
	cases:=[]previewCase{
		{"5548999000001","Maria (exemplo)","Quero conhecer os pacotes de Natal em Tijucas","meta_ads","",core.AdsProposal,outbox.TaskConfirmed,now.Add(-3*time.Minute)},
		{"5548999000002","Laura (exemplo)","Quero conhecer os pacotes de Natal em Tijucas","meta_ads","crm: HTTP 422 rejected",core.AdsProposal,outbox.TaskSuppressed,now.Add(-19*time.Minute)},
		{"5548999000003","Clara (exemplo)","Quero conhecer os pacotes de Natal em Tijucas","meta_ads","crm: HTTP 503 uncertain",core.AdsProposal,outbox.TaskUncertain,now.Add(-42*time.Minute)},
		{"5548999000004","Família (exemplo)","Bom dia, vocês abrem hoje?","normal","",core.Greeting,outbox.TaskConfirmed,now.Add(-78*time.Minute)},
		{"5548999000005","Contato (exemplo)","Vocês fazem fotos de Ano Novo?","normal","",core.Ignore,"",now.Add(-112*time.Minute)},
	}
	for i,c:=range cases{
		// Valid SHA-256 key follows the real outbox filename scheme.
		sha:=sha256.Sum256([]byte(fmt.Sprintf("sample-%d",i)))
		key:=hex.EncodeToString(sha[:])
		crmPhone:=strings.TrimPrefix(c.phone,"55")
		decision:=core.Decision{Action:c.action,InstanceID:"demo-uuid",MessageID:fmt.Sprintf("demo-%d",i),
			PhoneE164:c.phone,CRMPhone:crmPhone,Name:c.name}
		inbound:=map[string]interface{}{
			"message_type":"texto","text":c.message,"source":c.source,
			"trigger_match":c.action==core.AdsProposal,
			"trigger_text":func()string{if c.action==core.AdsProposal{return c.message};return ""}(),
			"has_ad_context":c.source=="meta_ads",
		}
		event:=map[string]interface{}{"decision":decision,"inbound":inbound,
			"received_at":c.received.UTC().Format(time.RFC3339Nano),
			"event_key":key,"status":"received"}
		if err:=writeJSON(filepath.Join(dir,key+".json"),event);err!=nil{return err}
		contact:=outbox.ContactState{PhoneE164:c.phone,UpdatedAt:c.received}
		if c.action==core.AdsProposal {contact.LastAdsFlowAt=c.received}
		if err:=writeJSON(filepath.Join(contactDir,key+".json"),contact);err!=nil{return err}
		if c.action==core.Ignore {continue}
		addTask:=func(index int,step workflow.Step,state outbox.TaskState,externalID,errorText string)error{
			task:=outbox.Task{ID:fmt.Sprintf("%s-%02d-%s",key,index,step),
				EventKey:key,PhoneE164:c.phone,ContactName:c.name,
				Step:step,State:state,ExternalID:externalID,Error:errorText,CreatedAt:c.received,
				AttemptStartedAt:c.received.Add(time.Duration(index)*time.Second)}
			if state!=outbox.TaskPending {task.CompletedAt=task.AttemptStartedAt.Add(time.Second)}
			return writeJSON(filepath.Join(taskDir,task.ID+".json"),task)
		}
		if c.action==core.Greeting {
			if err:=addTask(0,workflow.SendGreetingText,outbox.TaskConfirmed,"evolution-demo-menu","");err!=nil{return err}
			continue
		}
		if err:=addTask(0,workflow.SendAdText,outbox.TaskConfirmed,"evolution-demo-text","");err!=nil{return err}
		if err:=addTask(1,workflow.SendAdAudio,outbox.TaskConfirmed,"evolution-demo-audio","");err!=nil{return err}
		if err:=addTask(2,workflow.SendAdDocument,outbox.TaskConfirmed,"evolution-demo-pdf","");err!=nil{return err}
		receipt:=""
		if c.state==outbox.TaskConfirmed {receipt="proposal-dispatch-11111111-2222-4333-8444-555555555555"}
		if err:=addTask(3,workflow.ScheduleNatal,c.state,receipt,c.errorText);err!=nil{return err}
	}
	return nil
}

func writeJSON(path string, value interface{}) error {
	data,err:=json.Marshal(value)
	if err!=nil{return err}
	return os.WriteFile(path,data,0600)
}

func main(){
	// This command deliberately has no flags for public interfaces or real data.
	dir,err:=os.MkdirTemp("","evydencia-dashboard-demo-*")
	if err!=nil{log.Fatal(err)}
	defer os.RemoveAll(dir)
	if err:=makePreviewData(dir,time.Now().UTC());err!=nil{log.Fatal(err)}
	const demoPassword="demo-local"
	l,err:=net.Listen("tcp","127.0.0.1:0")
	if err!=nil{log.Fatal(err)}
	defer l.Close()
	mux:=http.NewServeMux()
	handler:=dashboard.NewHandler(dir,demoPassword)
	mux.Handle("/painel",handler)
	mux.Handle("/api/dashboard/",handler)
	fmt.Println("==== PREVIEW LOCAL COM DADOS 100% FICTÍCIOS ====")
	fmt.Println("Endereço: http://"+l.Addr().String()+"/painel")
	fmt.Println("Usuário: demo | Senha: "+demoPassword)
	fmt.Println("Somente loopback, sem envios externos. Encerrar: Ctrl+C")
	srv:=&http.Server{Handler:mux,ReadHeaderTimeout:5*time.Second}
	stop:=make(chan os.Signal,1)
	signal.Notify(stop,os.Interrupt)
	defer signal.Stop(stop)
	serveErr:=make(chan error,1)
	go func(){serveErr<-srv.Serve(l)}()
	select {
	case <-stop:
		ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	case err:=<-serveErr:
		if err!=nil && !errors.Is(err,http.ErrServerClosed) {
			log.Printf("preview local finalizado por erro: %v",err)
		}
	}

}
