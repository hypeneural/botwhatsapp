package evolution

import (
 "context"
 "encoding/base64"
 "encoding/json"
 "errors"
 "net/http"
 "net/http/httptest"
 "testing"
)
func TestStatusReadOnlyAndOptionalID(t *testing.T){
 calls:=0
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  calls++
  if r.Header.Get("apikey")!="instance-token"{t.Error("missing instance token")}
  if r.URL.Path=="/instance/status"{
   if r.Method!="GET"{t.Error("status must be GET")}
   _,_=w.Write([]byte(`{"message":"success","data":{"Connected":true,"LoggedIn":true,"Name":"bot-greeting"}}`))
   return
  }
  var data map[string]string
  if err:=json.NewDecoder(r.Body).Decode(&data);err!=nil{t.Error(err)}
  if data["id"]!="known-effect-id"{t.Error("missing send correlation ID")}
  if r.URL.Path=="/send/media"{
   if data["type"]!="audio"||data["caption"]!=""{t.Error("audio type mismatch")}
   media,err:=base64.StdEncoding.DecodeString(data["url"])
   if err!=nil||string(media)!="recording"{t.Error("bad base64")}
  }
  _,_=w.Write([]byte(`{"message":"success","data":{"Info":{"ID":"wa-out-1"}}}`))
 }))
 defer server.Close()
 c:=Client{BaseURL:server.URL,Token:"instance-token"}
 status,err:=c.Status(context.Background())
 if err!=nil||!status.Connected||!status.LoggedIn{t.Fatal(status,err)}
 if _,err=c.SendTextWithID(context.Background(),"5548999999999","hello","known-effect-id");err!=nil{t.Fatal(err)}
 if _,err=c.SendMediaWithID(context.Background(),"5548999999999","audio",[]byte("recording"),"","known-effect-id");err!=nil{t.Fatal(err)}
 if calls!=3{t.Fatal(calls)}
}
func TestStatusUnavailable(t *testing.T){
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.WriteHeader(http.StatusServiceUnavailable)
 }))
 defer server.Close()
 _,err:=(Client{BaseURL:server.URL,Token:"token"}).Status(context.Background())
 if !errors.Is(err,ErrStatusUnavailable){t.Fatal(err)}
}
