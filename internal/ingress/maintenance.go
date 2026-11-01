package ingress

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type historyEntry struct {
	name string
	mod  time.Time
}

// PruneHistory bounds *only* the informational root ingress journal. It never
// deletes outbox tasks, event dedup markers, contact policies, lead claims or
// uncertain executions. Run during low-frequency maintenance, not on ingress.
func PruneHistory(dir string, retention time.Duration, maxRecords int) (int, error) {
	if dir == "" || retention <= 0 || maxRecords < 1 {
		return 0, nil
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	rows := make([]historyEntry, 0)
	for _, item := range files {
		name := item.Name()
		if item.IsDir() || len(name) != 69 || !strings.HasSuffix(name, ".json") {
			continue
		}
		valid := true
		for _, ch := range name[:64] {
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		info, err := item.Info()
		if err != nil {
			continue
		}
		rows = append(rows, historyEntry{name: name, mod: info.ModTime()})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].mod.After(rows[j].mod) })
	cutoff := time.Now().Add(-retention)
	removed := 0
	for i, row := range rows {
		if i < maxRecords && !row.mod.Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, row.name)); err != nil && !os.IsNotExist(err) {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
