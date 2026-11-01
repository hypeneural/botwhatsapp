package ingress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPruneHistoryPreservesOperationalOutbox(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"tasks", "events", "contacts", "leads"} {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "preserve.lock"), []byte("important"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	old := strings.Repeat("a", 64) + ".json"
	newer := strings.Repeat("b", 64) + ".json"
	for _, name := range []string{old, newer} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("history"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	older := time.Now().Add(-100 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(root, old), older, older); err != nil {
		t.Fatal(err)
	}
	deleted, err := PruneHistory(root, 90*24*time.Hour, 30000)
	if err != nil || deleted != 1 {
		t.Fatalf("deleted=%d, err=%v", deleted, err)
	}
	if _, err := os.Stat(filepath.Join(root, newer)); err != nil {
		t.Fatal("new journal lost", err)
	}
	for _, dir := range []string{"tasks", "events", "contacts", "leads"} {
		if _, err := os.Stat(filepath.Join(root, dir, "preserve.lock")); err != nil {
			t.Fatalf("deleted durable state %s: %v", dir, err)
		}
	}
}
