package crm
import (
 "context"
 "errors"
 "net/http"
 "net/http/httptest"
 "testing"
)
func TestCRMClient(t *testing.T){
 h:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/api/campaigns/schedule/execute"{t.Error(r.URL.Path)}
  if r.Header.Get("Authorization")!="exact-service-token"{t.Error("wrong authorization")}
  w.Header().Set("Content-Type","application/json")
  _,_=w.Write([]byte(`{"data":{"id":4721,"campaign":416,"status":{"name":"scheduled"}}}`))
 }))
 defer h.Close()
 p,err:=Build("48996425287","",nil);if err!=nil{t.Fatal(err)}
 id,err:=(Client{BaseURL:h.URL,Authorization:"exact-service-token"}).Schedule(context.Background(),p)
 if id!=4721||err!=nil{t.Fatal(id,err)}
 missing:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_,_=w.Write([]byte(`{"data":null}`))}))
 defer missing.Close()
 _,err=(Client{BaseURL:missing.URL,Authorization:"token"}).Schedule(context.Background(),p)
 if !errors.Is(err,ErrUnconfirmed){t.Fatal("ambiguous response must not auto retry",err)}
 _,err=(Client{BaseURL:h.URL,Authorization:""}).Schedule(context.Background(),p)
 if !errors.Is(err,ErrInvalidRequest){t.Fatal(err)}
}
