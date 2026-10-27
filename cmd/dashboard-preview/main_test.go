package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/dashboard"
)

func TestSyntheticPreviewIsReadOnlyAndHasStatuses(t *testing.T) {
	dir:=t.TempDir()
	if err:=makePreviewData(dir,time.Now().UTC());err!=nil {t.Fatal(err)}
	data,err:=dashboard.NewService(dir).GetEvents(50)
	if err!=nil {t.Fatal(err)}
	if len(data.Events)!=5||data.Stats.MetaAds!=3||data.Stats.Normal!=2||
		data.Stats.CRMAccepted!=1||data.Stats.CRMRejected!=1||data.Stats.Uncertain!=1 {
		t.Fatalf("preview regression: total=%d stats=%+v",len(data.Events),data.Stats)
	}
	h:=dashboard.NewHandler(dir,"demo-local")
	// Only the dashboard GET paths are exposed in preview; no webhook/CRM.
	mux:=http.NewServeMux()
	mux.Handle("/painel",h)
	mux.Handle("/api/dashboard/",h)
	for _,path:=range []string{"/webhook/ads/key","/api/campaigns/schedule/execute"} {
		req:=httptest.NewRequest(http.MethodPost,path,strings.NewReader("{}"))
		w:=httptest.NewRecorder()
		mux.ServeHTTP(w,req)
		if w.Code!=http.StatusNotFound {t.Fatalf("preview exposed write route %s: %d",path,w.Code)}
	}
}
