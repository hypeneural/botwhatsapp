package outbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CommitEventTasks writes every task as a durable unit and creates the event
// dedup marker LAST. The worker's scan uses the same Store mutex, so a
// partially staged event is never observable by the worker in this process.
// This design requires ONE bot process per Store directory (no shared-volume
// multi-replica deployment).
func (s *Store) CommitEventTasks(instanceUUID, event, messageID string, tasks []Task) (bool, error) {
	if instanceUUID == "" || event == "" || messageID == "" || len(tasks) == 0 {
		return false, errors.New("outbox: invalid event batch")
	}
	sum := sha256.Sum256([]byte(instanceUUID + "\x00" + event + "\x00" + messageID))
	key := hex.EncodeToString(sum[:])
	marker := filepath.Join(s.eventsDir, key+".lock")

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(marker); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	type staged struct{ tmp, target string }
	stage := make([]staged, 0, len(tasks))
	published := make([]string, 0, len(tasks))
	defer func() {
		for _, item := range stage {
			_ = os.Remove(item.tmp)
		}
	}()
	rollback := func() {
		for _, p := range published {
			_ = os.Remove(p)
		}
		if d, err := os.Open(s.tasksDir); err == nil {
			_ = d.Sync()
			_ = d.Close()
		}
	}
	for _, task := range tasks {
		if task.ID == "" || task.ID != filepath.Base(task.ID) ||
			strings.ContainsAny(task.ID, "/\\") || task.InstanceID != instanceUUID ||
			task.EventKey != key {
			return false, errors.New("outbox: invalid task identity in batch")
		}
		if task.CreatedAt.IsZero() {
			task.CreatedAt = time.Now().UTC()
		}
		task.State = TaskPending
		raw, err := json.Marshal(task)
		if err != nil {
			return false, err
		}
		tmp, err := os.CreateTemp(s.tmpDir, ".batch-*")
		if err != nil {
			return false, err
		}
		stage = append(stage, staged{tmp: tmp.Name(), target: filepath.Join(s.tasksDir, task.ID+".json")})
		if err := tmp.Chmod(0600); err != nil {
			_ = tmp.Close()
			return false, err
		}
		if _, err := tmp.Write(raw); err != nil {
			_ = tmp.Close()
			return false, err
		}
		if err := tmp.Sync(); err != nil {
			_ = tmp.Close()
			return false, err
		}
		if err := tmp.Close(); err != nil {
			return false, err
		}
	}
	for _, item := range stage {
		// Hard-link is exclusive: never overwrite a prior task or its final state.
		if err := os.Link(item.tmp, item.target); err != nil {
			rollback()
			return false, err
		}
		published = append(published, item.target)
	}
	if dir, err := os.Open(s.tasksDir); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}

	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		rollback()
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	_, writeErr := fmt.Fprintf(f, "event_key=%s\ncreated_at=%s\n", key, time.Now().UTC().Format(time.RFC3339Nano))
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		// The marker may exist, but all tasks were persisted first. Preserve
		// it rather than risking a repeated external effect.
		if writeErr != nil {
			return false, writeErr
		}
		return false, closeErr
	}
	if dir, err := os.Open(s.eventsDir); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return true, nil
}
