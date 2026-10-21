package crm
import "testing"
func TestBuild(t *testing.T){
 w,err:=Build("48996425287","Mariana",nil)
 if err!=nil||w.Campaign!=415||w.Contacts!="48996425287;Mariana"{t.Fatal(w,err)}
 n,err:=Build("4832200440","",nil)
 if err!=nil||n.Campaign!=416||n.Contacts!="4832200440"{t.Fatal(n,err)}
 n,err=Build("48996425287","Ruim;\nnome",nil)
 if err!=nil||n.Campaign!=416 {t.Fatal(n,err)}
 if _,err=Build("5548996425287","nome",nil);err==nil{t.Fatal("DDI must be rejected")}
}
