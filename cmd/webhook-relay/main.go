// Command webhook-relay is a separate sidecar candidate. It does not alter
// Evolution Go, Caddy, ports, or any existing container on startup.
// Until RELAY_ENABLED=true and private endpoints are configured it rejects
// all webhook ingress. No startup causes an outbound request.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/relay"
)

func getenv(k,def string) string {
	if x:=os.Getenv(k);x!=""{return x}
	return def
}
func main(){
	enabled:=os.Getenv("RELAY_ENABLED")=="true"
	max:=20000
	if s:=os.Getenv("RELAY_MAX_EVENTS");s!=""{
		v,e:=strconv.Atoi(s)
		if e!=nil||v<1||v>100000 {log.Fatal("invalid RELAY_MAX_EVENTS")}
		max=v
	}
	store,err:=relay.NewStore(getenv("RELAY_DATA_DIR","./data/relay"),max)
	if err!=nil{log.Fatal("relay store init failed: ",err)}
	cfg:=relay.Config{
		Enabled:enabled,
		ExpectedInstanceUUID:os.Getenv("RELAY_EXPECTED_INSTANCE_UUID"),
		Secret:os.Getenv("RELAY_INGRESS_SECRET"),
		StatusURL:os.Getenv("RELAY_STATUS_URL"),
		BotURL:os.Getenv("RELAY_BOT_URL"),
	}
	svc,err:=relay.New(cfg,store)
	if err!=nil{log.Fatal("relay configuration invalid: ",err)}
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM)
	defer stop()
	if enabled {
		go svc.Run(ctx,func(err error){
			// Never print error bodies, headers, token-bearing URLs or PII.
			log.Printf("relay delivery/storage requires attention: %T",err)
		})
	}
	addr:=getenv("RELAY_LISTEN_ADDR","127.0.0.1:8789")
	server:=&http.Server{Addr:addr,Handler:svc,
		ReadHeaderTimeout:5*time.Second,ReadTimeout:10*time.Second,
		WriteTimeout:10*time.Second,IdleTimeout:30*time.Second,MaxHeaderBytes:8192,
	}
	go func(){<-ctx.Done();shutdownCtx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel();_ = server.Shutdown(shutdownCtx)}()
	log.Printf("webhook-relay enabled=%t; no existing status-engine ports are modified",enabled)
	if err:=server.ListenAndServe();err!=nil&&err!=http.ErrServerClosed{log.Fatalf("relay listen failed: %T",err)}
}
