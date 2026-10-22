// Package relay provides a separate, durable webhook fan-out component.
// It is NOT a replacement for the existing status-engine or Evolution Go.
// No existing webhook/port is modified by importing this package.
package relay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	ErrFull = errors.New("relay: spool capacity reached")
	ErrInvalid = errors.New("relay: invalid webhook envelope")
)

type Record struct {
	Key string `json:"key"`
	Body json.RawMessage `json:"body,omitempty"`
	StatusDone bool `json:"status_done"`
	BotDone bool `json:"bot_done"`
	CreatedAt time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	NextAttempt time.Time `json:"next_attempt,omitempty"`
}

type Store struct {
	mu sync.Mutex
	dir string
	max int
	count int
	maxBytes int64
	usedBytes int64
}

func NewStore(dir string, max int) (*Store,error) {
	return NewStoreWithLimits(dir,max,128<<20)
}
// NewStoreWithLimits enforces BOTH a maximum event count and a maximum spool
// disk budget. No unacknowledged event is silently purged to make room.
func NewStoreWithLimits(dir string,max int,maxBytes int64)(*Store,error){
	if dir=="" || max<=0 || maxBytes<4096 {return nil,errors.New("relay: invalid spool directory/capacity")}
	if err:=os.MkdirAll(dir,0700);err!=nil{return nil,err}
	entries,err:=os.ReadDir(dir)
	if err!=nil{return nil,err}
	n:=0
	var bytesUsed int64
	for _,e:=range entries {
		if !e.IsDir()&&strings.HasSuffix(e.Name(),".json"){
			info,err:=e.Info()
			if err!=nil{return nil,err}
			n++;bytesUsed+=info.Size()
		}
	}
	if n>max||bytesUsed>maxBytes{return nil,ErrFull}
	return &Store{dir:dir,max:max,count:n,maxBytes:maxBytes,usedBytes:bytesUsed},nil
}

func EventKey(body []byte, expectedUUID string)(string,error){
	var info struct{
		InstanceID string `json:"instanceId"`
		Event string `json:"event"`
		Data struct { Info struct{ ID string `json:"ID"` } `json:"Info"` } `json:"data"`
	}
	if err:=json.Unmarshal(body,&info);err!=nil||info.InstanceID!=expectedUUID||info.Event=="" {
		return "",ErrInvalid
	}
	var content []byte
	if info.Data.Info.ID!="" {
		content=[]byte(info.InstanceID+"\x00"+info.Event+"\x00"+info.Data.Info.ID)
	}else{
		// Events without message IDs must not collide with other event types.
		content=append([]byte(info.InstanceID+"\x00"+info.Event+"\x00"),body...)
	}
	sum:=sha256.Sum256(content)
	return hex.EncodeToString(sum[:]),nil
}

func (s *Store) path(key string)(string,error){
	if len(key)!=64{return "",ErrInvalid}
	for _,r:=range key {
		if r<'0'||r>'9'&&r<'a'||r>'f' {return "",ErrInvalid}
	}
	return filepath.Join(s.dir,key+".json"),nil
}
func (s *Store) write(rec Record)error{
	path,err:=s.path(rec.Key)
	if err!=nil{return err}
	raw,err:=json.Marshal(rec)
	if err!=nil{return err}
	f,err:=os.CreateTemp(s.dir,".relay-tmp-")
	if err!=nil{return err}
	defer os.Remove(f.Name())
	if err=f.Chmod(0600);err!=nil{f.Close();return err}
	if _,err=f.Write(raw);err!=nil{f.Close();return err}
	if err=f.Sync();err!=nil{f.Close();return err}
	if err=f.Close();err!=nil{return err}
	var priorSize int64
	if old,err:=os.Stat(path);err==nil{priorSize=old.Size()}else if !os.IsNotExist(err){return err}
	if err=os.Rename(f.Name(),path);err!=nil{return err}
	s.usedBytes+=int64(len(raw))-priorSize
	if runtime.GOOS!="windows" {
		d,err:=os.Open(s.dir)
		if err!=nil{return err}
		err=d.Sync();closeErr:=d.Close()
		if err!=nil{return err}
		if closeErr!=nil{return closeErr}
	}
	return nil
}

func (s *Store) Insert(key string,body []byte)(bool,error){
	s.mu.Lock();defer s.mu.Unlock()
	path,err:=s.path(key)
	if err!=nil{return false,err}
	if _,err:=os.Stat(path);err==nil{return false,nil}else if !os.IsNotExist(err){return false,err}
	// Reserve headroom for status flags/timestamps as the event progresses.
	// Store.write keeps usedBytes accurate as records shrink after delivery.
	if s.count>=s.max||s.usedBytes+int64(len(body))+2048>s.maxBytes{return false,ErrFull}
	rec:=Record{Key:key,Body:append([]byte(nil),body...),CreatedAt:time.Now().UTC()}
	if err:=s.write(rec);err!=nil{return false,err}
	s.count++
	return true,nil
}
func (s *Store) Get(key string)(Record,error){
	s.mu.Lock();defer s.mu.Unlock()
	path,err:=s.path(key);if err!=nil{return Record{},err}
	raw,err:=os.ReadFile(path);if err!=nil{return Record{},err}
	var rec Record
	if err:=json.Unmarshal(raw,&rec);err!=nil{return Record{},fmt.Errorf("relay: corrupt record: %w",err)}
	if rec.Key!=key {return Record{},errors.New("relay: record identity mismatch")}
	return rec,nil
}
// UpdateSink persists each sink's ACK separately. An ACKed sink is never
// deliberately re-sent after reboot; if the ACK's fsync fails, the outcome is
// uncertain, and downstream consumers MUST themselves deduplicate events.
func (s *Store) UpdateSink(key string,statusSink bool,now time.Time) (Record,error){
	s.mu.Lock();defer s.mu.Unlock()
	path,err:=s.path(key);if err!=nil{return Record{},err}
	b,err:=os.ReadFile(path);if err!=nil{return Record{},err}
	var r Record;if err=json.Unmarshal(b,&r);err!=nil||r.Key!=key{return Record{},errors.New("relay: corrupt record")}
	if statusSink {r.StatusDone=true}else{r.BotDone=true}
	r.NextAttempt=time.Time{}
	if r.StatusDone&&r.BotDone {r.Body=nil;r.CompletedAt=now.UTC()}
	if err:=s.write(r);err!=nil{return Record{},err}
	return r,nil
}
func (s *Store) Postpone(key string,until time.Time)error{
	s.mu.Lock();defer s.mu.Unlock()
	p,err:=s.path(key);if err!=nil{return err}
	b,err:=os.ReadFile(p);if err!=nil{return err}
	var r Record;if json.Unmarshal(b,&r)!=nil||r.Key!=key{return errors.New("relay: corrupt record")}
	if r.CompletedAt.IsZero(){r.NextAttempt=until.UTC()}
	return s.write(r)
}
func(s *Store) Pending(now time.Time)([]string,error){
	s.mu.Lock();defer s.mu.Unlock()
	entries,err:=os.ReadDir(s.dir);if err!=nil{return nil,err}
	keys:=make([]string,0)
	for _,e:=range entries{
		if e.IsDir()||!strings.HasSuffix(e.Name(),".json"){continue}
		b,err:=os.ReadFile(filepath.Join(s.dir,e.Name()))
		if err!=nil{return nil,err}
		var r Record
		if json.Unmarshal(b,&r)!=nil||r.Key+".json"!=e.Name(){return nil,errors.New("relay: corrupt spool")}
		if r.CompletedAt.IsZero()&&(r.NextAttempt.IsZero()||!now.Before(r.NextAttempt)){keys=append(keys,r.Key)}
	}
	return keys,nil
}
// Prune only completed records with scrubbed bodies, retaining dedup tombstones
// long enough for upstream retries. Pending records are NEVER silently dropped.
func(s *Store) PruneCompleted(before time.Time)(int,error){
	s.mu.Lock();defer s.mu.Unlock()
	entries,err:=os.ReadDir(s.dir);if err!=nil{return 0,err}
	n:=0
	for _,e:=range entries{
		if e.IsDir()||!strings.HasSuffix(e.Name(),".json"){continue}
		path:=filepath.Join(s.dir,e.Name())
		b,err:=os.ReadFile(path);if err!=nil{return n,err}
		var r Record
		if err:=json.Unmarshal(b,&r);err!=nil{return n,err}
		if !r.CompletedAt.IsZero()&&r.CompletedAt.Before(before)&&len(r.Body)==0{
			info,err:=os.Stat(path);if err!=nil{return n,err}
			if err:=os.Remove(path);err!=nil{return n,err}
			s.count--;s.usedBytes-=info.Size();n++
		}
	}
	return n,nil
}
