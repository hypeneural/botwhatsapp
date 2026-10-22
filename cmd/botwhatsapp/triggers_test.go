package main

import (
 "os"
 "path/filepath"
 "reflect"
 "testing"
)
func TestLoadAdTriggersFileOverridesEnv(t *testing.T){
 p:=filepath.Join(t.TempDir(),"triggers.txt")
 content:="# Comments\nQuero conhecer os pacotes do Primeiro Natal em Tijucas\n  \nquero conhecer os pacotes do Primeiro Natal em Tijucas\nQuero receber o orçamento de Natal em Tijucas\n"
 if err:=os.WriteFile(p,[]byte(content),0600);err!=nil{t.Fatal(err)}
 t.Setenv("BOT_ADS_TRIGGERS_FILE",p)
 t.Setenv("BOT_ADS_TRIGGERS","gatilho legado|natal")
 actual,err:=loadAdTriggers()
 if err!=nil{t.Fatal(err)}
 want:=[]string{"Quero conhecer os pacotes do Primeiro Natal em Tijucas","Quero receber o orçamento de Natal em Tijucas"}
 if !reflect.DeepEqual(actual,want){t.Fatalf("got %v want %v",actual,want)}
}
func TestLoadAdTriggersEnvMultiple(t *testing.T){
 t.Setenv("BOT_ADS_TRIGGERS_FILE","")
 t.Setenv("BOT_ADS_TRIGGERS","Primeiro Natal|Natal + Ano Novo|Primeiro Natal")
 actual,err:=loadAdTriggers()
 if err!=nil||len(actual)!=2{t.Fatalf("%v %v",actual,err)}
}
func TestLoadAdTriggersMissingFileFails(t *testing.T){
 t.Setenv("BOT_ADS_TRIGGERS_FILE",filepath.Join(t.TempDir(),"missing.txt"))
 if _,err:=loadAdTriggers();err==nil{t.Fatal("should refuse missing file")}
}
