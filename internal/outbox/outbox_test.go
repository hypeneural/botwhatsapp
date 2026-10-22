package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/crm"
	"github.com/hypeneural/botwhatsapp/internal/evolution"
	"github.com/hypeneural/botwhatsapp/internal/workflow"
)

type mockEvolutionSender struct {
	mu           sync.Mutex
	textCalls    int
	mediaCalls   int
	failAudio    bool
	textErr      error
	mediaErr     error
	sentTexts    []string
	sentMedia    []string
}

func (m *mockEvolutionSender) SendText(ctx context.Context, phone, text string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.textCalls++
	if m.textErr != nil {
		return "", m.textErr
	}
	m.sentTexts = append(m.sentTexts, text)
	return fmt.Sprintf("evo-msg-%d", m.textCalls), nil
}

func (m *mockEvolutionSender) SendMedia(ctx context.Context, phone, mediaType string, file []byte, caption string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mediaCalls++
	if mediaType == "audio" && m.failAudio {
		return "", evolution.ErrUnconfirmed
	}
	if m.mediaErr != nil {
		return "", m.mediaErr
	}
	m.sentMedia = append(m.sentMedia, mediaType)
	return fmt.Sprintf("evo-media-%d", m.mediaCalls), nil
}

type mockCRMScheduler struct {
	mu         sync.Mutex
	calls      int
	shouldFail bool
	failErr    error
	lastReq    crm.Request
}

func (m *mockCRMScheduler) Schedule(ctx context.Context, req crm.Request) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.lastReq = req
	if m.shouldFail {
		if m.failErr != nil {
			return 0, m.failErr
		}
		return 0, crm.ErrUnconfirmed
	}
	return 1000 + m.calls, nil
}

func setupTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store, dir
}

// 1. Crash/Restart Recovery:
// Interrupted attempt_started task must become uncertain on reboot and NEVER be retried.
func TestCrashRestartRecovery(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{}
	instanceID := 123
	cfg := Config{
		DataDir:                  dir,
		OperationalAuthorization: true,
		CRMInstanceID:            &instanceID,
	}

	_, err := NewWorker(cfg, store, evo, crmMock)
	if err != nil {
		t.Fatalf("failed to create worker: %v", err)
	}

	// Create a task and artificially transition to attempt_started (simulating crash during outbound HTTP)
	task := Task{
		ID:           "crash-test-task-1",
		EventKey:     "msg-crash-1",
		RecipientJID: "5548999999999@s.whatsapp.net",
		PhoneE164:    "5548999999999",
		CRMPhone:     "48999999999",
		Step:         workflow.SendAdText,
		PayloadText:  "Ola",
	}
	if err := store.EnqueueTask(task); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if _, err := store.UpdateTaskState(task.ID, func(t *Task) error {
		t.State = TaskAttemptStarted
		t.AttemptStartedAt = time.Now().UTC()
		return nil
	}); err != nil {
		t.Fatalf("failed to set attempt_started: %v", err)
	}

	// Simulate system reboot / restart with a new store instance on the same directory
	newStore, err := NewStore(dir)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}

	pending, crashed, err := newStore.RecoverTasks()
	if err != nil {
		t.Fatalf("recover failed: %v", err)
	}

	if len(pending) != 0 {
		t.Fatalf("expected 0 runnable pending tasks, got %d", len(pending))
	}
	if len(crashed) != 1 {
		t.Fatalf("expected 1 crashed task, got %d", len(crashed))
	}
	if crashed[0].State != TaskUncertain {
		t.Fatalf("expected crashed task to be marked uncertain, got %s", crashed[0].State)
	}

	// Verify DrainPending does not re-execute the task
	newWorker, _ := NewWorker(cfg, newStore, evo, crmMock)
	drained, err := newWorker.DrainPending(context.Background())
	if err != nil {
		t.Fatalf("drain failed: %v", err)
	}
	if drained != 0 {
		t.Fatalf("expected 0 tasks drained, got %d", drained)
	}
	if evo.textCalls != 0 {
		t.Fatalf("expected 0 text calls after crash recovery, got %d", evo.textCalls)
	}
}

// 2. 100 Concurrent Duplicates:
// 100 goroutines sending the exact same webhook event simultaneously; exactly 1 must succeed.
func Test100ConcurrentDuplicates(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{}
	instanceID := 123
	cfg := Config{
		DataDir:                  dir,
		OperationalAuthorization: true,
		CRMInstanceID:            &instanceID,
	}

	w, _ := NewWorker(cfg, store, evo, crmMock)

	decision := core.Decision{
		Action:     core.AdsProposal,
		InstanceID: "ads-uuid-1",
		MessageID:  "concurrent-msg-1",
		PhoneE164:  "5548999999999",
		ReplyJID:   "5548999999999@s.whatsapp.net",
		CRMPhone:   "48999999999",
		Name:       "Cliente Concorrente",
	}

	var acceptedCount int32
	var duplicateCount int32
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := w.IngestDecision(decision)
			if err == nil {
				atomic.AddInt32(&acceptedCount, 1)
			} else if errors.Is(err, ErrEventDuplicate) {
				atomic.AddInt32(&duplicateCount, 1)
			}
		}()
	}

	wg.Wait()

	if acceptedCount != 1 {
		t.Fatalf("expected exactly 1 accepted event, got %d", acceptedCount)
	}
	if duplicateCount != 99 {
		t.Fatalf("expected 99 duplicate rejections, got %d", duplicateCount)
	}
}

// 3. Lead Claim Deduplication:
// Same contact sends 2 separate messages triggering AdsProposal.
// First message claims the lead; second message has its CRM step suppressed to prevent duplicate proposals.
func TestLeadClaimDeduplication(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{}
	instanceID := 42
	cfg := Config{
		DataDir:                  dir,
		OperationalAuthorization: true,
		CRMInstanceID:            &instanceID,
	}

	w, _ := NewWorker(cfg, store, evo, crmMock)

	d1 := core.Decision{
		Action:     core.AdsProposal,
		InstanceID: "ads-uuid",
		MessageID:  "msg-1",
		PhoneE164:  "5548999999999",
		ReplyJID:   "5548999999999@s.whatsapp.net",
		CRMPhone:   "48999999999",
		Name:       "Ana Maria",
	}
	d2 := core.Decision{
		Action:     core.AdsProposal,
		InstanceID: "ads-uuid",
		MessageID:  "msg-2", // different message ID
		PhoneE164:  "5548999999999",
		ReplyJID:   "5548999999999@s.whatsapp.net",
		CRMPhone:   "48999999999",
		Name:       "Ana Maria",
	}

	if err := w.IngestDecision(d1); err != nil {
		t.Fatalf("ingest d1 failed: %v", err)
	}
	if err := w.IngestDecision(d2); err != nil {
		t.Fatalf("ingest d2 failed: %v", err)
	}

	// Drain all tasks
	_, err := w.DrainPending(context.Background())
	if err != nil {
		t.Fatalf("drain failed: %v", err)
	}

	// CRM should only have been scheduled ONCE
	if crmMock.calls != 1 {
		t.Fatalf("expected exactly 1 CRM schedule call, got %d", crmMock.calls)
	}
	if crmMock.lastReq.Campaign != crm.WithoutName {
		t.Fatalf("expected campaign 416 (without name), got %d", crmMock.lastReq.Campaign)
	}
	if crmMock.lastReq.Contacts != "48999999999" {
		t.Fatalf("expected contacts without name '48999999999', got %s", crmMock.lastReq.Contacts)
	}
}

// 4. Audio Failure Does Not Block CRM:
// If audio fails with timeout or error, CRM Natal step must still proceed and succeed.
func TestAudioFailureDoesNotBlockCRM(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{failAudio: true}
	crmMock := &mockCRMScheduler{}
	instanceID := 15
	cfg := Config{
		DataDir:                  dir,
		OperationalAuthorization: true,
		CRMInstanceID:            &instanceID,
		AdAudioFixture:           []byte("synthetic-audio-fixture"),
	}

	w, _ := NewWorker(cfg, store, evo, crmMock)

	decision := core.Decision{
		Action:     core.AdsProposal,
		InstanceID: "ads-uuid",
		MessageID:  "msg-audio-fail",
		PhoneE164:  "5548988887777",
		ReplyJID:   "5548988887777@s.whatsapp.net",
		CRMPhone:   "48988887777",
	}

	if err := w.IngestDecision(decision); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	_, err := w.DrainPending(context.Background())
	if err != nil {
		t.Fatalf("drain failed: %v", err)
	}

	if evo.textCalls != 1 {
		t.Fatalf("expected 1 text call, got %d", evo.textCalls)
	}
	if evo.mediaCalls != 1 {
		t.Fatalf("expected 1 audio media call attempt, got %d", evo.mediaCalls)
	}
	// CRM call MUST have been executed despite audio failure!
	if crmMock.calls != 1 {
		t.Fatalf("expected CRM call to succeed despite audio failure, got %d calls", crmMock.calls)
	}
	if crmMock.lastReq.Campaign != crm.WithoutName {
		t.Fatalf("expected campaign 416 (without name), got %d", crmMock.lastReq.Campaign)
	}
}

// 5. Timeout Uncertain Outcome:
// If CRM Schedule fails with timeout, task is marked uncertain and NEVER retried blindly.
func TestTimeoutUncertainOutcome(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{shouldFail: true, failErr: crm.ErrUnconfirmed}
	instanceID := 99
	cfg := Config{
		DataDir:                  dir,
		OperationalAuthorization: true,
		CRMInstanceID:            &instanceID,
	}

	w, _ := NewWorker(cfg, store, evo, crmMock)

	decision := core.Decision{
		Action:     core.AdsProposal,
		InstanceID: "ads-uuid",
		MessageID:  "msg-timeout",
		PhoneE164:  "5548977776666",
		ReplyJID:   "5548977776666@s.whatsapp.net",
		CRMPhone:   "48977776666",
	}

	if err := w.IngestDecision(decision); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	_, _ = w.DrainPending(context.Background())

	if crmMock.calls != 1 {
		t.Fatalf("expected 1 CRM call attempt, got %d", crmMock.calls)
	}

	// Try draining again: must NOT re-attempt!
	drained, _ := w.DrainPending(context.Background())
	if drained != 0 {
		t.Fatalf("expected 0 tasks drained on second pass, got %d", drained)
	}
	if crmMock.calls != 1 {
		t.Fatalf("CRM call was retried blindly! Got %d calls", crmMock.calls)
	}
}

// 6. Opt-Out Persistent Suppression:
// Contact sends "parar". Contact state is persisted as OptOut=true. Subsequent messages are dropped.
func TestOptOutPersistentSuppression(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{}
	cfg := Config{DataDir: dir}

	w, _ := NewWorker(cfg, store, evo, crmMock)

	phone := "5548911112222"

	// 1. Ingest opt-out message
	optOutDecision := core.Decision{
		Action:     core.Ignore,
		InstanceID: "ads-uuid",
		MessageID:  "msg-optout",
		PhoneE164:  phone,
		Reason:     "opt_out_requires_persistent_suppression",
	}
	if err := w.IngestDecision(optOutDecision); err != nil {
		t.Fatalf("opt out ingest failed: %v", err)
	}

	// Check contact record
	contact, err := store.GetContact(phone)
	if err != nil || !contact.OptOut {
		t.Fatalf("expected contact to have OptOut=true, got %v (err=%v)", contact.OptOut, err)
	}

	// 2. Contact tries to send another message
	subsequentDecision := core.Decision{
		Action:     core.Greeting,
		InstanceID: "greeting-uuid",
		MessageID:  "msg-after-optout",
		PhoneE164:  phone,
		ReplyJID:   phone + "@s.whatsapp.net",
		CRMPhone:   "48911112222",
	}
	if err := w.IngestDecision(subsequentDecision); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Worker should have enqueued 0 tasks
	pending, _, _ := store.RecoverTasks()
	if len(pending) != 0 {
		t.Fatalf("expected 0 tasks for opted-out contact, got %d", len(pending))
	}
}

// 7. Human Handoff and Greeting Cooldown:
// Option 5 pauses bot for human. Greeting cooldown prevents spam.
func TestHumanHandoffAndCooldown(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{}
	cfg := Config{
		DataDir:          dir,
		GreetingCooldown: 24 * time.Hour,
		GreetingImageFixture: []byte("synthetic-image-fixture"),
	}

	w, _ := NewWorker(cfg, store, evo, crmMock)
	phone := "5548933334444"
	replyJID := phone + "@s.whatsapp.net"

	// 1. Initial Greeting
	dGreet1 := core.Decision{
		Action:     core.Greeting,
		InstanceID: "greeting-uuid",
		MessageID:  "msg-g1",
		PhoneE164:  phone,
		ReplyJID:   replyJID,
		CRMPhone:   "48933334444",
	}
	if err := w.IngestDecision(dGreet1); err != nil {
		t.Fatalf("g1 failed: %v", err)
	}
	_, _ = w.DrainPending(context.Background())

	if evo.mediaCalls != 1 {
		t.Fatalf("expected 1 greeting image call, got %d", evo.mediaCalls)
	}

	// 2. Second greeting immediately after: should be suppressed by 24h cooldown
	dGreet2 := core.Decision{
		Action:     core.Greeting,
		InstanceID: "greeting-uuid",
		MessageID:  "msg-g2",
		PhoneE164:  phone,
		ReplyJID:   replyJID,
		CRMPhone:   "48933334444",
	}
	if err := w.IngestDecision(dGreet2); err != nil {
		t.Fatalf("g2 ingest failed: %v", err)
	}
	_, _ = w.DrainPending(context.Background())
	if evo.mediaCalls != 1 {
		t.Fatalf("expected greeting to be suppressed by cooldown, but got %d calls", evo.mediaCalls)
	}

	// 3. User sends option 5 (Human Handoff)
	dMenu5 := core.Decision{
		Action:     core.MenuOption,
		InstanceID: "greeting-uuid",
		MessageID:  "msg-m5",
		PhoneE164:  phone,
		ReplyJID:   replyJID,
		Option:     "5",
	}
	if err := w.IngestDecision(dMenu5); err != nil {
		t.Fatalf("menu5 failed: %v", err)
	}
	_, _ = w.DrainPending(context.Background())

	// Contact must now be human_paused
	c, _ := store.GetContact(phone)
	if !c.HumanPaused {
		t.Fatalf("expected contact to have HumanPaused=true")
	}

	// 4. Any subsequent greeting is blocked by human pause
	dGreet3 := core.Decision{
		Action:     core.Greeting,
		InstanceID: "greeting-uuid",
		MessageID:  "msg-g3",
		PhoneE164:  phone,
		ReplyJID:   replyJID,
	}
	if err := w.IngestDecision(dGreet3); err != nil {
		t.Fatalf("g3 failed: %v", err)
	}
	_, _ = w.DrainPending(context.Background())
	if evo.mediaCalls != 1 {
		t.Fatalf("greeting was sent to human-paused contact! Got %d calls", evo.mediaCalls)
	}
}

// 8. LID Without Phone Mapping:
// Fails closed and produces no outbound effects.
func TestLIDWithoutPhoneMapping(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{}
	cfg := Config{DataDir: dir}

	w, _ := NewWorker(cfg, store, evo, crmMock)

	rawLID := []byte(`{
		"event": "Message",
		"instanceId": "ads-uuid",
		"data": {
			"Info": {
				"ID": "lid-msg-1",
				"Chat": "123456789@lid",
				"Sender": "123456789@lid",
				"IsFromMe": false
			},
			"Message": {
				"conversation": "Quero a proposta de Natal 2026"
			}
		}
	}`)

	env, err := core.Parse(rawLID)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	coreCfg := core.Config{AdsUUID: "ads-uuid", AdsTriggers: []string{"Quero a proposta de Natal 2026"}}
	d := core.Select(env, coreCfg)

	if d.Action != core.Ignore {
		t.Fatalf("expected action Ignore for unresolved LID, got %s", d.Action)
	}

	if err := w.IngestDecision(d); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	pending, _, _ := store.RecoverTasks()
	if len(pending) != 0 {
		t.Fatalf("expected 0 tasks enqueued for LID without phone, got %d", len(pending))
	}
}

// 9. Shared Lead Claim Across Multi-Instances:
// If customer contacts instance A and later instance B in the same local calendar day,
// the lead claim is shared and blocks duplicate campaign dispatch.
func TestSharedLeadClaimAcrossInstances(t *testing.T) {
	store, dir := setupTestStore(t)
	evo := &mockEvolutionSender{}
	crmMock := &mockCRMScheduler{}
	instanceID := 1
	loc := time.FixedZone("BRT", -3*3600)
	cfg := Config{
		DataDir:                  dir,
		OperationalAuthorization: true,
		CRMInstanceID:            &instanceID,
		Tenant:                   "estudio-evydencia",
		Location:                 loc,
	}

	w, _ := NewWorker(cfg, store, evo, crmMock)

	// Inbound message from instance A (ads)
	d1 := core.Decision{
		Action:     core.AdsProposal,
		InstanceID: "uuid-instance-a",
		MessageID:  "msg-inst-a",
		PhoneE164:  "5548996553954",
		CRMPhone:   "48996553954",
		ReplyJID:   "5548996553954@s.whatsapp.net",
	}

	// Inbound message from instance B (greeting) for the same customer on the same day
	d2 := core.Decision{
		Action:     core.AdsProposal,
		InstanceID: "uuid-instance-b",
		MessageID:  "msg-inst-b",
		PhoneE164:  "5548996553954",
		CRMPhone:   "48996553954",
		ReplyJID:   "5548996553954@s.whatsapp.net",
	}

	if err := w.IngestDecision(d1); err != nil {
		t.Fatalf("ingest d1 failed: %v", err)
	}
	if err := w.IngestDecision(d2); err != nil {
		t.Fatalf("ingest d2 failed: %v", err)
	}

	// Drain tasks
	_, err := w.DrainPending(context.Background())
	if err != nil {
		t.Fatalf("drain failed: %v", err)
	}

	// CRM must be scheduled exactly once despite two different incoming instances
	if crmMock.calls != 1 {
		t.Fatalf("expected exactly 1 CRM schedule call, got %d", crmMock.calls)
	}
}

// 10. Prune Old Tasks and Capacity Limits:
// Confirmed tasks older than retention or exceeding capacity must be pruned safely.
func TestPruneOldTasksAndDiskCapacity(t *testing.T) {
	store, _ := setupTestStore(t)

	// Create 5 confirmed tasks with older completed timestamps
	now := time.Now().UTC()
	for i := 1; i <= 5; i++ {
		taskID := fmt.Sprintf("task-prune-%d", i)
		tObj := Task{
			ID:          taskID,
			Step:        workflow.SendAdText,
			State:       TaskConfirmed,
			CreatedAt:   now.Add(-48 * time.Hour),
			CompletedAt: now.Add(-time.Duration(30-i) * time.Hour),
		}
		raw, _ := json.Marshal(tObj)
		_ = store.atomicWriteFile(store.tasksDir, taskID+".json", raw)
	}

	// Prune tasks older than 24h
	deleted, err := store.PruneOldTasks(24*time.Hour, 10)
	if err != nil {
		t.Fatalf("prune failed: %v", err)
	}
	if deleted != 5 {
		t.Fatalf("expected 5 tasks pruned by age, got %d", deleted)
	}

	// Create 10 tasks and test capacity limit of 3
	for i := 1; i <= 10; i++ {
		taskID := fmt.Sprintf("task-cap-%d", i)
		tObj := Task{
			ID:          taskID,
			Step:        workflow.SendAdText,
			State:       TaskConfirmed,
			CreatedAt:   now,
			CompletedAt: now.Add(time.Duration(i) * time.Minute),
		}
		raw, _ := json.Marshal(tObj)
		_ = store.atomicWriteFile(store.tasksDir, taskID+".json", raw)
	}

	deletedCap, err := store.PruneOldTasks(0, 3)
	if err != nil {
		t.Fatalf("prune capacity failed: %v", err)
	}
	if deletedCap != 7 {
		t.Fatalf("expected 7 tasks pruned to enforce capacity of 3, got %d", deletedCap)
	}
}
