// Package ingress provides a safe, durable preview-only webhook intake.
// It NEVER sends a WhatsApp message or calls the CRM.
package ingress

import (
 "crypto/sha256"
 "crypto/subtle"
 "encoding/hex"
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "strings"
 "time"

 "github.com/hypeneural/botwhatsapp/internal/core"
)

type Settings struct {
 Enabled bool
 AdsSecret string
 GreetingSecret string
 Core core.Config
 SpoolDir string
 OnDecision func(core.Decision)
}
type Server struct {settings Settings}
func New(c Settings) (*Server,error){
 if c.SpoolDir=="" {return nil,errors.New("spool dir missing")}
 if c.Enabled && (len(c.AdsSecret)<32||len(c.GreetingSecret)<32||c.Core.AdsUUID==""||c.Core.GreetingUUID==""||c.Core.AdsUUID==c.Core.GreetingUUID) {
  return nil,errors.New("enable requires separate UUID mappings and two 32+ character webhook secrets")
 }
 if err:=os.MkdirAll(c.SpoolDir,0700);err!=nil{return nil,err}
 return &Server{settings:c},nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter,r *http.Request) {
 if r.URL.Path=="/healthz"&&r.Method=="GET" {w.WriteHeader(200);_,_=io.WriteString(w,"ok");return}
 if r.URL.Path=="/readyz"&&r.Method=="GET" {
  if !s.settings.Enabled {http.Error(w,"preview disabled",503);return}
  w.WriteHeader(200);_,_=io.WriteString(w,"ready");return
 }
 // NO request logging: URL contains a secret.
 if r.Method!="POST" {http.Error(w,"not found",404);return}
 var expected,uuid string
 switch {
 case strings.HasPrefix(r.URL.Path,"/webhook/ads/"):
  expected=s.settings.AdsSecret;uuid=s.settings.Core.AdsUUID
 case strings.HasPrefix(r.URL.Path,"/webhook/greeting/"):
  expected=s.settings.GreetingSecret;uuid=s.settings.Core.GreetingUUID
 default: http.Error(w,"not found",404);return
 }
 if !s.settings.Enabled {http.Error(w,"service disabled",503);return}
 secret:=r.URL.Path[strings.LastIndex(r.URL.Path,"/")+1:]
 if subtle.ConstantTimeCompare([]byte(secret),[]byte(expected))!=1 || secret=="" {
  http.Error(w,"unauthorized",401);return
 }
 r.Body=http.MaxBytesReader(w,r.Body,65536)
 body,err:=io.ReadAll(r.Body)
 if err!=nil {http.Error(w,"payload too large or unreadable",413);return}
 e,err:=core.Parse(body)
 if err!=nil {http.Error(w,"bad JSON",400);return}
 if e.InstanceID!=uuid {http.Error(w,"wrong instance",403);return}
 decision:=core.Select(e,s.settings.Core)
 if decision.Action==core.Ignore {w.WriteHeader(204);return}
 // Stable key for idempotency. Do not embed numbers/names in filenames.
 key:=sha256.Sum256([]byte(e.InstanceID+"\x00"+e.Event+"\x00"+e.Data.Info.ID))
 name:=hex.EncodeToString(key[:])+".json"
 // Atomic: only expose a fully written record. Hard link refuses overwrite.
 tmp,err:=os.CreateTemp(s.settings.SpoolDir,".tmp-*")
 if err!=nil {http.Error(w,"spool unavailable",503);return}
 defer os.Remove(tmp.Name())
 defer tmp.Close()
 if err=tmp.Chmod(0600);err!=nil{http.Error(w,"spool unavailable",503);return}
 record:=struct {
  Decision core.Decision `json:"decision"`
  ReceivedAt string `json:"received_at"`
  Status string `json:"status"`
 }{decision,time.Now().UTC().Format(time.RFC3339Nano),"planned_preview_only"}
 if err=json.NewEncoder(tmp).Encode(record);err!=nil{http.Error(w,"spool unavailable",503);return}
 if err=tmp.Sync();err!=nil{http.Error(w,"spool unavailable",503);return}
 if err=tmp.Close();err!=nil{http.Error(w,"spool unavailable",503);return}
 err=os.Link(tmp.Name(),filepath.Join(s.settings.SpoolDir,name))
 if os.IsExist(err) {w.WriteHeader(202);return}
 if err!=nil {http.Error(w,"spool unavailable",503);return}
 // Delivery of inbound webhook is accepted only after a durable record exists.
 if d,err:=os.Open(s.settings.SpoolDir);err==nil {_=d.Sync();_=d.Close()}
 if s.settings.OnDecision!=nil {go s.settings.OnDecision(decision)}
 w.WriteHeader(202)
}
