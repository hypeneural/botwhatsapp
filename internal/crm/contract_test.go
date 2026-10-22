package crm

import (
 "errors"
 "testing"
)
func TestDefaultAlwaysWithoutName(t *testing.T){
 mid:=42
 for _, name:=range []string{"","Mariana","Ruim;\nnome"} {
  req,err:=Build("48996425287",name,&mid)
  if err!=nil || req.Campaign!=WithoutName || req.Contacts!="48996425287" ||
   req.Type!=CampaignKind || req.UseLeadsSystem || req.Instance==nil || *req.Instance!=42 {
   t.Fatalf("unexpected default: %+v err=%v",req,err)
  }
 }
 if _,err:=Build("5548996425287","",&mid);!errors.Is(err,ErrInvalidContact){t.Fatal("DDI must be rejected",err)}
 if _,err:=Build("4832200440","",nil);err!=nil{t.Fatal("build is pure and may preview unknown instance",err)}
}
func TestOptInWithNameOnly(t *testing.T){
 mid:=3
 req,err:=BuildWithName("48996425287","Mariana",&mid)
 if err!=nil || req.Campaign!=WithName || req.Contacts!="48996425287;Mariana"{t.Fatal(req,err)}
 if _,err:=BuildWithName("48996425287","Ruim;\nnome",&mid);!errors.Is(err,ErrInvalidName){t.Fatal(err)}
}
