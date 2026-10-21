package workflow
import (
 "testing"
 "time"
 "github.com/hypeneural/botwhatsapp/internal/core"
)
func TestPlans(t *testing.T){
 ad:=Create(core.Decision{Action:core.AdsProposal})
 if len(ad.Steps)!=3||ad.Steps[2]!=ScheduleNatal{t.Fatal(ad)}
 gr:=Create(core.Decision{Action:core.Greeting})
 if len(gr.Steps)!=1||gr.Steps[0]!=SendGreetingImage{t.Fatal(gr)}
 h:=Create(core.Decision{Action:core.MenuOption,Option:"5"})
 if len(h.Steps)!=2||h.Steps[1]!=PauseHuman{t.Fatal(h)}
 if Reply("4")==""{t.Fatal("location missing")}
}
func TestCooldown(t *testing.T){
 now:=time.Date(2026,10,21,20,0,0,0,time.UTC)
 last:=now.Add(-10*time.Hour)
 if IsGreetingDue(last,now,24*time.Hour,false){t.Fatal("must not greet within cooldown")}
 if !IsGreetingDue(last,now,8*time.Hour,false){t.Fatal("expired cooldown must greet")}
 if IsGreetingDue(time.Time{},now,24*time.Hour,true){t.Fatal("human pause ignored")}
 if !IsGreetingDue(time.Time{},now,24*time.Hour,false){t.Fatal("new chat must greet")}
}
