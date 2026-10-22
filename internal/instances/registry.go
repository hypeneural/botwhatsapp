// Package instances is a pure configuration registry for multiple Evolution Go
// instances. It performs no I/O and never connects or changes WhatsApp sessions.
package instances

import (
 "crypto/sha256"
 "crypto/subtle"
 "encoding/hex"
 "errors"
 "fmt"
 "regexp"
 "strings"
 "time"

 "github.com/hypeneural/botwhatsapp/internal/core"
)

type Role string
const (
 Ads Role = "ads"
 Greeting Role = "greeting"
 Observe Role = "observe"
)
var (
 ErrInvalid = errors.New("instances: invalid configuration")
 ErrUnknown = errors.New("instances: unknown or disabled instance")
 ErrInvalidClaim = errors.New("instances: invalid lead claim")
 slugRE = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,39}$`)
 phoneRE = regexp.MustCompile(`^55[0-9]{10,11}$`)
 nationalRE = regexp.MustCompile(`^[0-9]{10,11}// Package instances is a pure configuration registry for multiple Evolution Go
// instances. It performs no I/O and never connects or changes WhatsApp sessions.
package instances

import (
 "crypto/sha256"
 "crypto/subtle"
 "encoding/hex"
 "errors"
 "fmt"
 "regexp"
 "strings"
 "time"

 "github.com/hypeneural/botwhatsapp/internal/core"
)

type Role string
const (
 Ads Role = "ads"
 Greeting Role = "greeting"
 Observe Role = "observe"
)
var (
 ErrInvalid = errors.New("instances: invalid configuration")
 ErrUnknown = errors.New("instances: unknown or disabled instance")
 ErrInvalidClaim = errors.New("instances: invalid lead claim")
 slugRE = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,39}$`)
 phoneRE = regexp.MustCompile(`^55[0-9]{10,11}$`)
)
 envRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,79}// Package instances is a pure configuration registry for multiple Evolution Go
// instances. It performs no I/O and never connects or changes WhatsApp sessions.
package instances

import (
 "crypto/sha256"
 "crypto/subtle"
 "encoding/hex"
 "errors"
 "fmt"
 "regexp"
 "strings"
 "time"

 "github.com/hypeneural/botwhatsapp/internal/core"
)

type Role string
const (
 Ads Role = "ads"
 Greeting Role = "greeting"
 Observe Role = "observe"
)
var (
 ErrInvalid = errors.New("instances: invalid configuration")
 ErrUnknown = errors.New("instances: unknown or disabled instance")
 ErrInvalidClaim = errors.New("instances: invalid lead claim")
 slugRE = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,39}$`)
 phoneRE = regexp.MustCompile(`^55[0-9]{10,11}$`)
)
)
type Definition struct {
 Slug string
 InstanceUUID string
 BusinessPhoneE164 string
 Role Role
 Enabled bool
 TokenEnv string
 WebhookSecretEnv string
 AdsTriggers []string
 CRMAllowed bool
}
type Registry struct {
 byID map[string]Definition
 bySlug map[string]Definition
 byPhone map[string]Definition
}
func New(defs []Definition)(*Registry,error){
 if len(defs)==0{return nil,fmt.Errorf("%w: empty registry",ErrInvalid)}
 r:=&Registry{byID:make(map[string]Definition,len(defs)),
  bySlug:make(map[string]Definition,len(defs)),byPhone:make(map[string]Definition,len(defs))}
 for _,d:=range defs{
  if !slugRE.MatchString(d.Slug) ||
   (d.Role!=Ads&&d.Role!=Greeting&&d.Role!=Observe) ||
   (d.CRMAllowed&&d.Role!=Ads){
   return nil,fmt.Errorf("%w: slug or role invalid",ErrInvalid)
  }
  if d.InstanceUUID==""||d.BusinessPhoneE164=="" {
   // Inventory placeholders are allowed only when an instance is disabled.
   if d.Enabled {return nil,fmt.Errorf("%w: enabled instance missing identity",ErrInvalid)}
  }else if !phoneRE.MatchString(d.BusinessPhoneE164) || len(d.InstanceUUID)>128 ||
   strings.TrimSpace(d.InstanceUUID)!=d.InstanceUUID{
   return nil,fmt.Errorf("%w: invalid phone or UUID",ErrInvalid)
  }
  if d.Enabled && (!envRE.MatchString(d.TokenEnv) || !envRE.MatchString(d.WebhookSecretEnv) ||
   d.TokenEnv==d.WebhookSecretEnv) {
   return nil,fmt.Errorf("%w: enabled instance needs distinct ENV variable names, not secret values",ErrInvalid)
  }
  if d.TokenEnv!="" && !envRE.MatchString(d.TokenEnv) ||
   d.WebhookSecretEnv!="" && !envRE.MatchString(d.WebhookSecretEnv){
   return nil,fmt.Errorf("%w: invalid ENV variable identifier",ErrInvalid)
  }
  if d.Role==Ads && d.Enabled && len(d.AdsTriggers)==0{
   return nil,fmt.Errorf("%w: ads role requires explicit triggers",ErrInvalid)
  }
  if d.Role!=Ads && len(d.AdsTriggers)>0{
   return nil,fmt.Errorf("%w: non-ads role cannot hold ads triggers",ErrInvalid)
  }
  if _,exists:=r.bySlug[d.Slug];exists{return nil,fmt.Errorf("%w: duplicate slug",ErrInvalid)}
  if d.InstanceUUID!="" {
   if _,exists:=r.byID[d.InstanceUUID];exists{return nil,fmt.Errorf("%w: duplicate UUID",ErrInvalid)}
   r.byID[d.InstanceUUID]=d
  }
  if d.BusinessPhoneE164!="" {
   if _,exists:=r.byPhone[d.BusinessPhoneE164];exists{return nil,fmt.Errorf("%w: duplicate business phone",ErrInvalid)}
   r.byPhone[d.BusinessPhoneE164]=d
  }
  r.bySlug[d.Slug]=d
 }
 return r,nil
}
func (r *Registry) ResolveUUID(id string)(Definition,error){
 if r==nil{return Definition{},ErrUnknown}
 d,ok:=r.byID[id]
 if !ok||!d.Enabled{return Definition{},ErrUnknown}
 return d,nil
}
func (r *Registry) ResolveSlug(slug string)(Definition,error){
 if r==nil{return Definition{},ErrUnknown}
 d,ok:=r.bySlug[slug]
 if !ok||!d.Enabled{return Definition{},ErrUnknown}
 return d,nil
}
// Decide routes any number of independently configured instances while
// reusing the existing tested core behavior. No outbound HTTP is performed.
func (r *Registry) Decide(e core.Envelope) (core.Decision,error){
 d,err:=r.ResolveUUID(e.InstanceID);if err!=nil{return core.Decision{},err}
 var cfg core.Config
 switch d.Role{
 case Ads:
  cfg.AdsUUID=d.InstanceUUID
  cfg.AdsTriggers=d.AdsTriggers
 case Greeting:
  cfg.GreetingUUID=d.InstanceUUID
 default:
  return core.Decision{Action:core.Ignore,InstanceID:e.InstanceID,
   MessageID:e.Data.Info.ID,Reason:"observe_only"},nil
 }
 result:=core.Select(e,cfg)
 return result,nil
}
// VerifyRoute enforces an EXACT webhook path and secret for each instance;
// it must not be called with fields from the untrusted JSON body.
func (r *Registry) VerifyRoute(path,slug,configuredSecret string) (Definition,error){
 d,err:=r.ResolveSlug(slug)
 if err!=nil||len(configuredSecret)<32{return Definition{},ErrUnknown}
 prefix:="/webhook/"+slug+"/"
 if !strings.HasPrefix(path,prefix){return Definition{},ErrUnknown}
 candidate:=strings.TrimPrefix(path,prefix)
 if candidate==""||strings.Contains(candidate,"/")||strings.Contains(candidate,"\\")||
  len(candidate)>256{return Definition{},ErrUnknown}
 a:=sha256.Sum256([]byte(candidate))
 b:=sha256.Sum256([]byte(configuredSecret))
 if subtle.ConstantTimeCompare(a[:],b[:])!=1{return Definition{},ErrUnknown}
 return d,nil
}
// EventKey deduplicates transport delivery per instance, event and message.
func EventKey(instanceUUID,event,messageID string)(string,error){
 if instanceUUID==""||event==""||messageID=="" {return "",ErrInvalidClaim}
 h:=sha256.Sum256([]byte(instanceUUID+"\x00"+event+"\x00"+messageID))
 return hex.EncodeToString(h[:]),nil
}
// LeadClaimKey is SHARED by every instance of the same brand/tenant. Not
// including the Evolution instance prevents cross-instance duplicate CRM sends.
// The default window is one local calendar day, in an explicit timezone.
func LeadClaimKey(tenant,phoneNational string,campaign int,when time.Time,location *time.Location)(string,error){
 if tenant==""||len(tenant)>100||strings.ContainsRune(tenant,'\x00')||
  !nationalRE.MatchString(phoneNational)||campaign<=0||location==nil||
  when.IsZero(){return "",ErrInvalidClaim}
 day:=when.In(location).Format("2006-01-02")
 h:=sha256.Sum256([]byte(tenant+"\x00"+phoneNational+"\x00"+
  fmt.Sprint(campaign)+"\x00"+day))
 return hex.EncodeToString(h[:]),nil
}
