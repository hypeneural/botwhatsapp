package outbox

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "time"
)

// UpdateContact serializes read/modify/write of policy and cooldown fields.
// Unlike GetContact + SaveContact, it cannot overwrite a concurrent PARAR or
// human takeover with stale in-memory contact data.
func (s *Store) UpdateContact(phone string, update func(*ContactState) error) (ContactState,error) {
 if phone=="" || update==nil {return ContactState{},errors.New("outbox: missing contact update argument")}
 s.mu.Lock()
 defer s.mu.Unlock()
 hash:=sha256.Sum256([]byte(phone))
 filename:=hex.EncodeToString(hash[:])+".json"
 path:=filepath.Join(s.contactsDir,filename)
 state:=ContactState{PhoneE164:phone}
 raw,err:=os.ReadFile(path)
 switch {
 case err==nil:
  if err:=json.Unmarshal(raw,&state);err!=nil{
   return ContactState{},fmt.Errorf("outbox: corrupt contact state: %w",err)
  }
  if state.PhoneE164!=phone{return ContactState{},errors.New("outbox: contact state identity mismatch")}
 case os.IsNotExist(err):
  // A genuinely new contact starts with no policy flags.
 default:
  return ContactState{},err
 }
 if err:=update(&state);err!=nil{return state,err}
 state.UpdatedAt=time.Now().UTC()
 b,err:=json.Marshal(state);if err!=nil{return ContactState{},err}
 if err:=s.atomicWriteFile(s.contactsDir,filename,b);err!=nil{return ContactState{},err}
 return state,nil
}
