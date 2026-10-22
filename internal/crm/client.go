package crm

// The ONLY CRM integration is its EXISTING HTTP API. No CRM repository,
// schema, business logic, or infrastructure changes are necessary.
// The preview server does not wire this client to any outbound action.
import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "strings"
 "time"
)

var (
 ErrUnconfirmed = errors.New("crm: outcome unknown; never blindly replay POST")
 ErrInvalidRequest = errors.New("crm: invalid preflight")
 ErrRejected = errors.New("crm: request definitively rejected")
 ErrLookupUnavailable = errors.New("crm: schedule lookup unavailable or malformed")
)

type RejectedError struct{ StatusCode int }
func (e *RejectedError) Error() string{return fmt.Sprintf("crm: HTTP %d rejected",e.StatusCode)}
func (e *RejectedError) Is(target error)bool{return target==ErrRejected}

// Do not log the secret or bodies. A reusable *http.Client can be injected for
// offline httptest; a default shared transport is reused for real connections.
type Client struct {
 BaseURL string
 Authorization string
 HTTP *http.Client
}

type ScheduleReceipt struct {
 ID int `json:"id"`
 Campaign int `json:"campaign"`
 InstanceID int `json:"instance_id"`
 Status string `json:"status"`
}
type ScheduleMatch struct {
 ID int `json:"id"`
 Campaign int `json:"campaign"`
 InstanceID int `json:"instance_id"`
 Status string `json:"status"`
 CreatedAt string `json:"created_at,omitempty"`
}

func(c Client) preflight() (string,error){
 u,err:=url.Parse(c.BaseURL)
 if err!=nil || (u.Scheme!="https"&&u.Scheme!="http") || u.Host=="" ||
  u.User!=nil||u.RawQuery!=""||u.Fragment!=""||u.Path!=""||
  strings.TrimSpace(c.Authorization)=="" || c.Authorization!=strings.TrimSpace(c.Authorization){
  return "",ErrInvalidRequest
 }
 return c.BaseURL,nil
}
func(c Client) httpClient()*http.Client{
 if c.HTTP!=nil{return c.HTTP}
 return &http.Client{Timeout:15*time.Second}
}

// Decode the real Laravel response. AppServiceProvider calls
// JsonResource::withoutWrapping(), so the expected response is ROOT-level
// {"id":123, ...}; tolerate {"data":{...}} for legacy/test compatibility.
// 2xx with missing ID, changed campaign or wrong sending instance is UNCERTAIN.
func parseScheduledReceipt(b []byte,req Request)(ScheduleReceipt,error){
 var envelope struct{
  ID int `json:"id"`
  Campaign int `json:"campaign"`
  Instance struct{ID int `json:"id"`} `json:"instance"`
  Status struct{Name string `json:"name"`} `json:"status"`
  Data json.RawMessage `json:"data"`
 }
 if err:=json.Unmarshal(b,&envelope);err!=nil{return ScheduleReceipt{},ErrUnconfirmed}
 if len(envelope.Data)>0 && string(bytes.TrimSpace(envelope.Data))!="null"{
  if err:=json.Unmarshal(envelope.Data,&envelope);err!=nil{return ScheduleReceipt{},ErrUnconfirmed}
 }
 if req.Instance==nil || envelope.ID<=0 ||
  envelope.Campaign!=req.Campaign ||
  envelope.Instance.ID!=*req.Instance ||
  envelope.Status.Name=="" {
  return ScheduleReceipt{},ErrUnconfirmed
 }
 return ScheduleReceipt{ID:envelope.ID,Campaign:envelope.Campaign,
  InstanceID:envelope.Instance.ID,Status:envelope.Status.Name},nil
}
func(c Client) Schedule(ctx context.Context, req Request)(int,error){
 result,err:=c.ScheduleDetailed(ctx,req)
 return result.ID,err
}
func(c Client) ScheduleDetailed(ctx context.Context,req Request)(ScheduleReceipt,error){
 base,err:=c.preflight()
 if err!=nil||req.Campaign!=WithoutName&&req.Campaign!=WithName||
  req.Instance==nil||*req.Instance<=0||req.Type!=CampaignKind||
  req.UseLeadsSystem||len(req.Contacts)==0||strings.ContainsAny(req.Contacts,"\r\n"){
  return ScheduleReceipt{},ErrInvalidRequest
 }
 // Verify body structure exactly before any I/O, not just a non-empty contact.
 validated:=false
 switch req.Campaign {
 case WithoutName:
  validated=phoneRE.MatchString(req.Contacts)
 case WithName:
  parts:=strings.Split(req.Contacts,";")
  validated=len(parts)==2&&phoneRE.MatchString(parts[0])&&
   parts[1]!=""&&!strings.ContainsAny(parts[1],"\r\n")
 }
 if !validated{return ScheduleReceipt{},ErrInvalidRequest}
 b,err:=Encode(req);if err!=nil{return ScheduleReceipt{},ErrInvalidRequest}
 h,err:=http.NewRequestWithContext(ctx,http.MethodPost,base+"/api/campaigns/schedule/execute",bytes.NewReader(b))
 if err!=nil{return ScheduleReceipt{},ErrInvalidRequest}
 h.Header.Set("Authorization",c.Authorization) // RAW internal token; NOT Bearer
 h.Header.Set("Content-Type","application/json")
 res,err:=c.httpClient().Do(h)
 if err!=nil{return ScheduleReceipt{},ErrUnconfirmed}
 defer res.Body.Close()
 raw,err:=io.ReadAll(io.LimitReader(res.Body,65537))
 if err!=nil||len(raw)>65536{return ScheduleReceipt{},ErrUnconfirmed}
 // Guaranteed rejected at request-validation/authentication layer.
 if res.StatusCode==400||res.StatusCode==401||res.StatusCode==403||
  res.StatusCode==404||res.StatusCode==422{
  return ScheduleReceipt{},&RejectedError{StatusCode:res.StatusCode}
 }
 // Treat 408/409/429/5xx as uncertain. Do not retry this POST.
 if res.StatusCode<200||res.StatusCode>=300{return ScheduleReceipt{},ErrUnconfirmed}
 return parseScheduledReceipt(raw,req)
}

// Search is a READ-ONLY aid for operators resolving uncertain delivery.
// An empty search result is NOT proof that a timed-out POST was not accepted.
// Its results must NEVER authorize an automatic second POST.
func(c Client) Search(ctx context.Context,campaign int,phoneNational string)([]ScheduleMatch,error){
 base,err:=c.preflight()
 if err!=nil|| (campaign!=WithoutName&&campaign!=WithName) ||
  !phoneRE.MatchString(phoneNational){
  return nil,ErrInvalidRequest
 }
 q:=url.Values{}
 q.Set("campaign.id",fmt.Sprint(campaign))
 q.Set("contacts.phone",phoneNational)
 h,err:=http.NewRequestWithContext(ctx,http.MethodGet,base+"/api/campaigns/schedule/search?"+q.Encode(),nil)
 if err!=nil{return nil,ErrInvalidRequest}
 h.Header.Set("Authorization",c.Authorization)
 h.Header.Set("Accept","application/json")
 res,err:=c.httpClient().Do(h)
 if err!=nil{return nil,ErrLookupUnavailable}
 defer res.Body.Close()
 raw,err:=io.ReadAll(io.LimitReader(res.Body,262145))
 if err!=nil||len(raw)>262144||res.StatusCode!=200{return nil,ErrLookupUnavailable}
 var rows []json.RawMessage
 if err:=json.Unmarshal(raw,&rows);err!=nil{
  var outer struct{ Data []json.RawMessage `json:"data"` }
  if json.Unmarshal(raw,&outer)!=nil||outer.Data==nil{return nil,ErrLookupUnavailable}
  rows=outer.Data
 }
 matches:=make([]ScheduleMatch,0,len(rows))
 for _,item:=range rows{
  var entry struct{
   ID int `json:"id"`
   Campaign int `json:"campaign"`
   Instance struct{ID int `json:"id"`} `json:"instance"`
   Status struct{Name string `json:"name"`} `json:"status"`
   CreatedAt string `json:"created_at"`
   Contacts []struct{Whatsapp string `json:"whatsapp"`} `json:"contacts"`
  }
  if json.Unmarshal(item,&entry)!=nil||entry.ID<=0 {return nil,ErrLookupUnavailable}
  if entry.Campaign!=campaign {continue}
  matchPhone:=false
  for _,person:=range entry.Contacts{
   if person.Whatsapp==phoneNational {matchPhone=true;break}
  }
  if !matchPhone {continue}
  matches=append(matches,ScheduleMatch{
   ID:entry.ID,Campaign:entry.Campaign,InstanceID:entry.Instance.ID,
   Status:entry.Status.Name,CreatedAt:entry.CreatedAt,
  })
 }
 return matches,nil
}
