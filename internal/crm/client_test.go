package crm

import (
 "context"
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "sync/atomic"
 "testing"
)
func TestScheduleRootUnwrappedResponse(t *testing.T){
 var calls atomic.Int32
 h:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  calls.Add(1)
  if r.URL.Path!="/api/campaigns/schedule/execute" || r.Method!="POST"{t.Error(r.URL.Path)}
  if r.Header.Get("Authorization")!="exact-service-token"{t.Error("wrong raw authorization")}
  b,_:=io.ReadAll(r.Body)
  var p Request
  if err:=json.Unmarshal(b,&p);err!=nil{t.Error(err)}
  if p.Campaign!=416 || p.Contacts!="48996425287" || p.Instance==nil || *p.Instance!=42 ||
   p.Type!="campaign" || p.UseLeadsSystem{t.Errorf("bad request %+v",p)}
  w.Header().Set("Content-Type","application/json")
  // Source: CRM .scribe/examples/campaigns/execute-success.json is ROOT-level
  _,_=w.Write([]byte(`{"id":4721,"campaign":416,"instance":{"id":42,"name":"Oficial"},"status":{"id":1,"name":"scheduled"}}`))
 }))
 defer h.Close()
 mid:=42
 p,_:=Build("48996425287","Any sender push name",&mid)
 result,err:=(Client{BaseURL:h.URL,Authorization:"exact-service-token"}).ScheduleDetailed(context.Background(),p)
 if err!=nil || result.ID!=4721 || result.Campaign!=416 || result.InstanceID!=42 || result.Status!="scheduled"{t.Fatal(result,err)}
 if calls.Load()!=1{t.Fatal(calls.Load())}
}
func TestScheduleWrappedCompatAndWrongSender(t *testing.T){
 mid:=42
 p,_:=BuildWithoutName("48996425287",&mid)
 for _, tc:=range []struct{body string;wantErr bool}{
  {`{"data":{"id":1,"campaign":416,"instance":{"id":42},"status":{"name":"scheduled"}}}`,false},
  {`{"data":null}`,true},
  {`{"id":1,"campaign":415,"instance":{"id":42},"status":{"name":"scheduled"}}`,true},
  {`{"id":1,"campaign":416,"instance":{"id":9},"status":{"name":"scheduled"}}`,true},
  {`{"id":1,"campaign":416,"instance":{"id":42}}`,true},
 }{
  srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,_ *http.Request){_,_=w.Write([]byte(tc.body))}))
  _,err:=(Client{BaseURL:srv.URL,Authorization:"token"}).Schedule(context.Background(),p)
  srv.Close()
  if tc.wantErr && !errors.Is(err,ErrUnconfirmed){t.Errorf("expected uncertain for %s: %v",tc.body,err)}
  if !tc.wantErr && err!=nil{t.Errorf("unexpected %s: %v",tc.body,err)}
 }
}
func TestRejectedIsNotBlindRetry(t *testing.T){
 mid:=42
 p,_:=BuildWithoutName("48996425287",&mid)
 for _, status:=range []int{400,401,403,404,422,500,429}{
  srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,_ *http.Request){w.WriteHeader(status);_,_=w.Write([]byte("private error"))}))
  _,err:=(Client{BaseURL:srv.URL,Authorization:"token"}).Schedule(context.Background(),p)
  srv.Close()
  if status<500 && status!=429 && !errors.Is(err,ErrRejected){t.Errorf("status %d: %v",status,err)}
  if (status>=500||status==429) && !errors.Is(err,ErrUnconfirmed){t.Errorf("uncertain status %d: %v",status,err)}
  if strings.Contains(err.Error(),"private error"){t.Fatal("error leaked upstream body")}
 }
 p.Instance=nil
 _,err:=(Client{BaseURL:"https://evydencia.com",Authorization:"token"}).Schedule(context.Background(),p)
 if !errors.Is(err,ErrInvalidRequest){t.Fatal("must never send with favorite instance",err)}
}
func TestSearchReadOnlyUnwrappedAndWrapper(t *testing.T){
 for _, wrapped:=range []bool{false,true}{
  srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
   if r.Method!="GET"||r.URL.Path!="/api/campaigns/schedule/search"{t.Error(r.Method,r.URL.Path)}
   if r.URL.Query().Get("campaign.id")!="416"||r.URL.Query().Get("contacts.phone")!="48996425287"{t.Error(r.URL.RawQuery)}
   if r.Header.Get("Authorization")!="raw-token"{t.Error("auth")}
   data:=`[{"id":9,"campaign":416,"instance":{"id":42},"status":{"name":"scheduled"},"contacts":[{"whatsapp":"48996425287"}]},{"id":10,"campaign":415,"contacts":[{"whatsapp":"48996425287"}]}]`
   if wrapped{data=`{"data":`+data+`}`}
   _,_=w.Write([]byte(data))
  }))
  result,err:=(Client{BaseURL:srv.URL,Authorization:"raw-token"}).Search(context.Background(),416,"48996425287")
  srv.Close()
  if err!=nil||len(result)!=1||result[0].ID!=9||result[0].InstanceID!=42{t.Fatal(wrapped,result,err)}
 }
}
