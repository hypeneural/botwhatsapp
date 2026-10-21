// Main is a safe phase-1 preview server. No outbound clients are wired.
package main
import (
 "log"
 "net/http"
 "os"
 "strings"
 "time"

 "github.com/hypeneural/botwhatsapp/internal/core"
 "github.com/hypeneural/botwhatsapp/internal/ingress"
)
func env(k,def string)string{if v:=os.Getenv(k);v!=""{return v};return def}
func main(){
 enabled:=os.Getenv("BOT_ENABLED")=="true"
 triggers:=[]string{}
 if s:=strings.TrimSpace(os.Getenv("BOT_ADS_TRIGGERS"));s!=""{
  for _,x:=range strings.Split(s,"|"){if x=strings.TrimSpace(x);x!=""{triggers=append(triggers,x)}}
 }
 h,err:=ingress.New(ingress.Settings{
  Enabled:enabled,
  AdsSecret:os.Getenv("BOT_ADS_WEBHOOK_SECRET"),GreetingSecret:os.Getenv("BOT_GREETING_WEBHOOK_SECRET"),
  SpoolDir:env("BOT_DATA_DIR","./data/spool"),
  Core:core.Config{AdsUUID:os.Getenv("BOT_ADS_INSTANCE_UUID"),GreetingUUID:os.Getenv("BOT_GREETING_INSTANCE_UUID"),
   AdsTriggers:triggers,AdsFallback:os.Getenv("BOT_ADS_FALLBACK_MENU")=="true"},
 })
 if err!=nil{log.Fatal("configuration invalid: ",err)}
 addr:=env("BOT_LISTEN_ADDR","127.0.0.1:8788")
 log.Printf("preview-only webhook intake listening on %s; enabled=%t (NO outbound sends)",addr,enabled)
 srv:=&http.Server{Addr:addr,Handler:h,ReadHeaderTimeout:5*time.Second,ReadTimeout:10*time.Second,
  WriteTimeout:10*time.Second,IdleTimeout:30*time.Second,MaxHeaderBytes:8192}
 log.Fatal(srv.ListenAndServe())
}
