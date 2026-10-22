package outbox

import (
 "strings"
 "unicode"
)

// firstName never executes raw PushName as instructions or in template markup.
// Accept a short first token of ordinary Unicode letters, e.g. "Tati🍀" -> "Tati".
func firstName(raw string) string {
 raw=strings.TrimSpace(raw)
 var b strings.Builder
 for _,r:=range raw {
  if !unicode.IsLetter(r) && !unicode.IsMark(r) {break}
  if b.Len()+len(string(r))>60 {return ""}
  b.WriteRune(r)
 }
 if b.Len()==0{return ""}
 return b.String()
}
// personalize replaces only the controlled greeting variable. Missing or
// unsafe names produce a well-formed general greeting, not stray commas.
func personalize(template,pushName string)string{
 name:=firstName(pushName)
 if name=="" {
  template=strings.ReplaceAll(template,"Oi, {{nome}}!","Oi!")
  template=strings.ReplaceAll(template,"Olá, {{nome}}!","Olá!")
 }
 template=strings.ReplaceAll(template,"{{nome}}",name)
 template=strings.ReplaceAll(template,"{{primeiro_nome}}",name)
 return template
}
