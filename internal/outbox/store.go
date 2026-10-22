package outbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/instances"
)

var (
	ErrEventDuplicate = errors.New("outbox: duplicate event")
	ErrLeadClaimed    = errors.New("outbox: lead already claimed in current window")
	ErrTaskNotFound   = errors.New("outbox: task not found")
)

type Store struct {
	baseDir     string
	eventsDir   string
	leadsDir    string
	contactsDir string
	tasksDir    string
	tmpDir      string
	mu          sync.Mutex
}

func NewStore(baseDir string) (*Store, error) {
	if baseDir == "" {
		return nil, errors.New("outbox: base directory required")
	}

	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "events"),
		filepath.Join(baseDir, "leads"),
		filepath.Join(baseDir, "contacts"),
		filepath.Join(baseDir, "tasks"),
		filepath.Join(baseDir, "tmp"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("outbox: failed to create directory %s: %w", dir, err)
		}
	}

	return &Store{
		baseDir:     baseDir,
		eventsDir:   dirs[1],
		leadsDir:    dirs[2],
		contactsDir: dirs[3],
		tasksDir:    dirs[4],
		tmpDir:      dirs[5],
	}, nil
}

func (s *Store) atomicWriteFile(dir, filename string, data []byte) error {
	tmpFile, err := os.CreateTemp(s.tmpDir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		return err
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	targetPath := filepath.Join(dir, filename)
	if err := os.Rename(tmpName, targetPath); err != nil {
		return err
	}

	// fsync directory to guarantee directory entry persistence
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	return nil
}

// CheckAndRecordEvent returns true if the event was newly recorded, or false if it was already seen.
func (s *Store) CheckAndRecordEvent(instanceID, eventType, messageID string) (bool, error) {
	if instanceID == "" || eventType == "" || messageID == "" {
		return false, errors.New("outbox: invalid event identifiers")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	hash := sha256.Sum256([]byte(instanceID + "\x00" + eventType + "\x00" + messageID))
	key := hex.EncodeToString(hash[:])
	targetPath := filepath.Join(s.eventsDir, key+".lock")

	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	info := fmt.Sprintf("instance=%s\nevent=%s\nmessage=%s\ntime=%s\n",
		instanceID, eventType, messageID, time.Now().UTC().Format(time.RFC3339Nano))
	if _, err := io.WriteString(f, info); err != nil {
		return false, err
	}
	if err := f.Sync(); err != nil {
		return false, err
	}

	return true, nil
}

// ClaimLeadShared claims a lead across all instances of a tenant for a national phone and campaign,
// using an explicit timezone location (e.g. America/Sao_Paulo) for the daily window.
func (s *Store) ClaimLeadShared(tenant, phoneNational string, campaign int, when time.Time, loc *time.Location) (bool, error) {
	key, err := instances.LeadClaimKey(tenant, phoneNational, campaign, when, loc)
	if err != nil {
		return false, fmt.Errorf("outbox: invalid lead claim key: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	targetPath := filepath.Join(s.leadsDir, key+".lock")
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	info := fmt.Sprintf("tenant=%s\nphone=%s\ncampaign=%d\ntime=%s\n",
		tenant, phoneNational, campaign, when.In(loc).Format(time.RFC3339Nano))
	if _, err := io.WriteString(f, info); err != nil {
		return false, err
	}
	if err := f.Sync(); err != nil {
		return false, err
	}

	return true, nil
}

// ClaimLead claims a lead for a phone, campaign and time window (e.g. current day or window bucket).
// Returns true if claimed successfully, false if already claimed in this window.
func (s *Store) ClaimLead(phone, campaignID, window string) (bool, error) {
	if phone == "" || campaignID == "" || window == "" {
		return false, errors.New("outbox: invalid lead claim parameters")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	hash := sha256.Sum256([]byte(phone + "\x00" + campaignID + "\x00" + window))
	key := hex.EncodeToString(hash[:])
	targetPath := filepath.Join(s.leadsDir, key+".lock")

	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	info := fmt.Sprintf("phone=%s\ncampaign=%s\nwindow=%s\ntime=%s\n",
		phone, campaignID, window, time.Now().UTC().Format(time.RFC3339Nano))
	if _, err := io.WriteString(f, info); err != nil {
		return false, err
	}
	if err := f.Sync(); err != nil {
		return false, err
	}

	return true, nil
}

// GetContact retrieves the contact state by phone. If not found, returns empty state with PhoneE164 set.
func (s *Store) GetContact(phone string) (ContactState, error) {
	if phone == "" {
		return ContactState{}, errors.New("outbox: phone required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	hash := sha256.Sum256([]byte(phone))
	key := hex.EncodeToString(hash[:])
	path := filepath.Join(s.contactsDir, key+".json")

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ContactState{PhoneE164: phone}, nil
		}
		return ContactState{}, err
	}

	var state ContactState
	if err := json.Unmarshal(raw, &state); err != nil {
		// A corrupt policy is NOT a new contact: fail closed to preserve opt-outs.
		return ContactState{}, fmt.Errorf("outbox: corrupt contact state: %w", err)
	}
	if state.PhoneE164 != phone {
		return ContactState{}, errors.New("outbox: contact state identity mismatch")
	}
	return state, nil
}

// SaveContact updates contact state atomically.
func (s *Store) SaveContact(state ContactState) error {
	if state.PhoneE164 == "" {
		return errors.New("outbox: phone required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}

	hash := sha256.Sum256([]byte(state.PhoneE164))
	key := hex.EncodeToString(hash[:]) + ".json"

	return s.atomicWriteFile(s.contactsDir, key, raw)
}

// EnqueueTask persists a new task in pending state.
func (s *Store) EnqueueTask(task Task) error {
	if task.ID == "" {
		return errors.New("outbox: task ID required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	}
	task.State = TaskPending

	raw, err := json.Marshal(task)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.json", task.ID)
	return s.atomicWriteFile(s.tasksDir, filename, raw)
}

// UpdateTaskState updates a task's state atomically with fsync.
func (s *Store) UpdateTaskState(taskID string, updateFn func(t *Task) error) (Task, error) {
	if taskID == "" {
		return Task{}, errors.New("outbox: task ID required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", taskID)
	path := filepath.Join(s.tasksDir, filename)

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, err
	}

	var t Task
	if err := json.Unmarshal(raw, &t); err != nil {
		return Task{}, err
	}

	if err := updateFn(&t); err != nil {
		return Task{}, err
	}

	updatedRaw, err := json.Marshal(t)
	if err != nil {
		return Task{}, err
	}

	if err := s.atomicWriteFile(s.tasksDir, filename, updatedRaw); err != nil {
		return Task{}, err
	}

	return t, nil
}

// GetTask reads a task by ID.
func (s *Store) GetTask(taskID string) (Task, error) {
	if taskID == "" {
		return Task{}, errors.New("outbox: task ID required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", taskID)
	path := filepath.Join(s.tasksDir, filename)

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, err
	}

	var t Task
	if err := json.Unmarshal(raw, &t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// RecoverTasks scans tasks directory and separates tasks into pending (runnable)
// and tasks crashed in attempt_started (which are flagged as uncertain / pending reconciliation, NEVER retried automatically).
func (s *Store) RecoverTasks() (pending []Task, crashedPendingReconciliation []Task, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.tasksDir)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.tasksDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var t Task
		if err := json.Unmarshal(raw, &t); err != nil {
			continue
		}

		switch t.State {
		case TaskPending:
			pending = append(pending, t)
		case TaskAttemptStarted:
			// CRASH RECOVERY: task was interrupted while external call was in flight.
			// Transition to uncertain with crash reason to prevent blind duplicate retry.
			t.State = TaskUncertain
			t.Error = "crash_recovery: attempt was started but interrupted before completion; pending manual reconciliation"
			t.CompletedAt = time.Now().UTC()
			if updatedRaw, err := json.Marshal(t); err == nil {
				_ = s.atomicWriteFile(s.tasksDir, entry.Name(), updatedRaw)
			}
			crashedPendingReconciliation = append(crashedPendingReconciliation, t)
		}
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].CreatedAt.Before(pending[j].CreatedAt)
	})

	return pending, crashedPendingReconciliation, nil
}

// PruneOldTasks cleans up confirmed and suppressed tasks older than maxAge.
// If maxCount > 0 and the remaining tasks exceed maxCount, it purges the oldest
// completed tasks to protect disk capacity. Returns the count of deleted files.
func (s *Store) PruneOldTasks(maxAge time.Duration, maxCount int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.tasksDir)
	if err != nil {
		return 0, err
	}

	type fileInfo struct {
		name        string
		completedAt time.Time
		state       TaskState
	}

	var completed []fileInfo
	now := time.Now().UTC()
	deleted := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.tasksDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var t Task
		if err := json.Unmarshal(raw, &t); err != nil {
			continue
		}

		if t.State == TaskConfirmed || t.State == TaskSuppressed {
			if maxAge > 0 && !t.CompletedAt.IsZero() && now.Sub(t.CompletedAt) > maxAge {
				_ = os.Remove(path)
				deleted++
				continue
			}
			completed = append(completed, fileInfo{
				name:        entry.Name(),
				completedAt: t.CompletedAt,
				state:       t.State,
			})
		}
	}

	// Enforce maxCount capacity if configured
	if maxCount > 0 && len(completed) > maxCount {
		sort.Slice(completed, func(i, j int) bool {
			return completed[i].completedAt.Before(completed[j].completedAt)
		})
		toDelete := len(completed) - maxCount
		for i := 0; i < toDelete; i++ {
			_ = os.Remove(filepath.Join(s.tasksDir, completed[i].name))
			deleted++
		}
	}

	return deleted, nil
}
