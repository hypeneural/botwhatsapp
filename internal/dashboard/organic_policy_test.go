package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

func TestUnmatchedCTWAWithAdContextRequestsHumanWithoutBotTasks(t *testing.T) {
	dir:=t.TempDir()
	now:=time.Now().UTC()
	for _,tc:=range []struct{key,source,status string;meta,attention bool}{
		{strings.Repeat("a",64),"meta_ads","atendimento_manual",true,true},
		{strings.Repeat("b",64),"normal","conversa_organica",false,false},
	} {
		row:=map[string]interface{}{
			"event_key":tc.key,"received_at":now.Format(time.RFC3339Nano),
			"decision":core.Decision{Action:core.Ignore,Reason:"ads_unmatched_no_automation",
				InstanceID:"ads",MessageID:tc.key[:8],PhoneE164:"5548999991004"},
			"inbound":map[string]interface{}{"source":tc.source,"message_type":"texto",
				"has_ad_context":tc.meta,"trigger_match":false,"text":"Oi, posso falar?"},
			"status":"received",
		}
		b,err:=json.Marshal(row)
		if err!=nil{t.Fatal(err)}
		if err:=os.WriteFile(filepath.Join(dir,tc.key+".json"),b,0600);err!=nil{t.Fatal(err)}
	}
	result,err:=NewService(dir).GetEvents(10)
	if err!=nil{t.Fatal(err)}
	if len(result.Events)!=2{t.Fatalf("events=%d",len(result.Events))}
	for _,event:=range result.Events{
		if event.HasAdContext {
			if event.Status!="atendimento_manual"||!event.NeedsAttention||len(event.Steps)!=0{
				t.Fatalf("ad context without trigger must be human-only: %+v",event)
			}
		} else if event.Status!="conversa_organica"||event.NeedsAttention{
			t.Fatalf("ordinary chat must not be a bot failure: %+v",event)
		}
	}
}
