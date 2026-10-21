package crm

// Read-only contract of CRM in another repo. This adapter can POST only when
// an authorized caller explicitly wires it; the preview server does not.
import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "net/url"
 "time"
)

var ErrUnconfirmed=errors.New("crm: scheduling outcome unknown; do not retry automatically")
var ErrInvalidRequest=errors.New("crm: invalid base URL, token or campaign payload")
type Client struct {BaseURL string;Authorization string;HTTP *http.Client}
func(c Client) Schedule(ctx context.Context,req Request)(int,error){
 base,err:=url.Parse(c.BaseURL)
 if err!=nil||base.Host==""||base.Scheme!="https"&&base.Scheme!="http"||base.User!=nil||base.RawQuery!=""||base.Fragment!=""||base.Path!=""||c.Authorization==""||req.Campaign!=WithName&&req.Campaign!=WithoutName||req.Contacts==""||req.Instance==nil||*req.Instance<=0 {
  return 0,ErrInvalidRequest
 }
 b,err:=Encode(req);if err!=nil{return 0,ErrInvalidRequest}
 request,err:=http.NewRequestWithContext(ctx,"POST",c.BaseURL+"/api/campaigns/schedule/execute",bytes.NewReader(b))
 if err!=nil{return 0,ErrInvalidRequest}
 request.Header.Set("Authorization",c.Authorization) // EXACT token; NO Bearer prefix
 request.Header.Set("Content-Type","application/json")
 client:=c.HTTP;if client==nil{client=&http.Client{Timeout:15*time.Second}}
 res,err:=client.Do(request);if err!=nil{return 0,ErrUnconfirmed}
 defer res.Body.Close()
 raw,err:=io.ReadAll(io.LimitReader(res.Body,65537))
 if err!=nil||len(raw)>65536||res.StatusCode<200||res.StatusCode>=300{return 0,ErrUnconfirmed}
 var reply struct{Data struct{ID int `json:"id"`;Campaign int `json:"campaign"`} `json:"data"`}
 if json.Unmarshal(raw,&reply)!=nil||reply.Data.ID<=0||reply.Data.Campaign!=req.Campaign{return 0,ErrUnconfirmed}
 return reply.Data.ID,nil
}
