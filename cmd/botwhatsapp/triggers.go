package main

import (
 "bufio"
 "errors"
 "os"
 "strings"
 "github.com/hypeneural/botwhatsapp/internal/core"
)

// loadAdTriggers uses EITHER a UTF-8 file (one full phrase per line) OR the
// legacy pipe-separated env. A configured file replaces, not adds to, env.
// Changing the file requires a safe service restart; no remote fetch or regex.
func loadAdTriggers() ([]string,error) {
 var entries []string
 if p:=strings.TrimSpace(os.Getenv("BOT_ADS_TRIGGERS_FILE"));p!=""{
  f,err:=os.Open(p);if err!=nil{return nil,err}
  defer f.Close()
  scanner:=bufio.NewScanner(f)
  scanner.Buffer(make([]byte,4096),64<<10)
  for scanner.Scan(){
   line:=strings.TrimSpace(scanner.Text())
   if line==""||strings.HasPrefix(line,"#"){continue}
   entries=append(entries,line)
  }
  if err:=scanner.Err();err!=nil{return nil,err}
 }else{
  entries=strings.Split(os.Getenv("BOT_ADS_TRIGGERS"),"|")
 }
 result:=make([]string,0,len(entries))
 seen:=make(map[string]struct{})
 for _,raw:=range entries{
  normalized:=core.Normalize(raw)
  if normalized==""{continue}
  if len([]rune(normalized))>240{return nil,errors.New("ads trigger exceeds 240 characters")}
  if _,ok:=seen[normalized];ok{continue}
  seen[normalized]=struct{}{}
  result=append(result,raw)
  if len(result)>50{return nil,errors.New("ads trigger limit is 50 exact phrases")}
 }
 return result,nil
}
