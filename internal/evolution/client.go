// Package evolution is a minimal v0.7.2 HTTP adapter, NOT wired to the preview server.
// No automatic retry: a network failure after POST has unknown delivery status.
package evolution
import (
 "bytes"
 "context"
 "encoding/base64"
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "net/url"
 "strings"
 "time"
)
var ErrUnconfirmed=errors.New("evolution: outbound result unconfirmed, do not retry automatically")
var ErrInvalid=errors.New("evolution: invalid request")
type Client struct {BaseURL string;Token string;HTTP *http.Client}
func (c Client) send(ctx context.Context,path string,payload any)(string,error){
 base,err:=url.Parse(c.BaseURL)
 if err!=nil||base.Scheme!="http"&&base.Scheme!="https"||base.Host==""||base.User!=nil||base.RawQuery!=""||base.Fragment!=""||base.Path!=""||c.Token==""{
  return "",ErrInvalid
 }
 b,err:=json.Marshal(payload);if err!=nil||len(b)>16<<20{return "",ErrInvalid}
 client:=c.HTTP;if client==nil{client=&http.Client{Timeout:12*time.Second}}
 req,err:=http.NewRequestWithContext(ctx,"POST",c.BaseURL+path,bytes.NewReader(b))
 if err!=nil{return "",ErrInvalid}
 req.Header.Set("apikey",c.Token)
 req.Header.Set("Content-Type","application/json")
 res,err:=client.Do(req);if err!=nil{return "",ErrUnconfirmed}
 defer res.Body.Close()
 raw,err:=io.ReadAll(io.LimitReader(res.Body,65537))
 if err!=nil||len(raw)>65536||res.StatusCode<200||res.StatusCode>=300{return "",ErrUnconfirmed}
 var response struct{
  Data struct{Info struct{ID string `json:"ID"`} `json:"Info"`} `json:"data"`
 }
 if json.Unmarshal(raw,&response)!=nil||response.Data.Info.ID=="" {return "",ErrUnconfirmed}
 return response.Data.Info.ID,nil
}
func (c Client) SendText(ctx context.Context,phone,text string)(string,error){
 if phone==""||strings.TrimSpace(text)==""{return "",ErrInvalid}
 return c.send(ctx,"/send/text",map[string]string{"number":phone,"text":text})
}
func (c Client) SendMedia(ctx context.Context,phone,mediaType string,file []byte,caption string)(string,error){
 if phone==""||(mediaType!="image"&&mediaType!="audio")||len(file)==0||len(file)>8<<20{return "",ErrInvalid}
 return c.send(ctx,"/send/media",map[string]string{"number":phone,"type":mediaType,"url":base64.StdEncoding.EncodeToString(file),"caption":caption})
}
