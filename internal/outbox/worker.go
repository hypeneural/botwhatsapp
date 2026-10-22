package outbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/crm"
	"github.com/hypeneural/botwhatsapp/internal/instances"
	"github.com/hypeneural/botwhatsapp/internal/workflow"
)

type Worker struct {
	cfg       Config
	store     *Store
	evolution EvolutionSender
	crm       CRMScheduler
	taskCh    chan Task
	mu        sync.Mutex
}

func NewWorker(cfg Config, store *Store, evolution EvolutionSender, crmScheduler CRMScheduler) (*Worker, error) {
	if store == nil {
		return nil, fmt.Errorf("outbox: store required")
	}
	if cfg.GreetingCooldown <= 0 {
		cfg.GreetingCooldown = 24 * time.Hour
	}
	if cfg.AdWelcomeText == "" {
		cfg.AdWelcomeText = "Oii! Tudo bem? 🎄❤️ Que alegria receber seu contato! Vamos enviar a proposta de Natal 2026 por outro WhatsApp oficial do Estúdio Evidência."
	}
	if cfg.GreetingCaption == "" {
		cfg.GreetingCaption = "Olá! Seja bem-vindo ao Estúdio Evidência. Escolha uma das opções abaixo para continuarmos:\n1. Natal 2026\n2. Agendamentos\n3. Fotos e pedidos\n4. Localização\n5. Falar com atendente humano"
	}
	if cfg.Tenant == "" {
		cfg.Tenant = "estudio-evydencia"
	}
	if cfg.Location == nil {
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			loc = time.FixedZone("BRT", -3*3600)
		}
		cfg.Location = loc
	}
	if cfg.TaskRetention <= 0 {
		cfg.TaskRetention = 7 * 24 * time.Hour
	}
	if cfg.MaxTasksCount <= 0 {
		cfg.MaxTasksCount = 10000
	}

	w := &Worker{
		cfg:       cfg,
		store:     store,
		evolution: evolution,
		crm:       crmScheduler,
		taskCh:    make(chan Task, 256),
	}

	return w, nil
}

// IngestDecision accepts a core.Decision, checks deduplication, opt-out, cooldown and enqueues tasks.
func (w *Worker) IngestDecision(d core.Decision) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 1. If reason is opt-out suppression, record it immediately and stop.
	if d.Reason == "opt_out_requires_persistent_suppression" {
		if d.PhoneE164 != "" {
			c, err := w.store.GetContact(d.PhoneE164)
			if err == nil {
				c.OptOut = true
				c.OptOutAt = time.Now().UTC()
				if err := w.store.SaveContact(c); err != nil {return err}
			} else { return err }
		}
		return nil
	}

	if d.Action == core.Ignore {
		return nil
	}

	// Check contact policy before publishing a durable event marker.
	// The marker is committed LAST, only after all workflow tasks are durable.
	// 3. Check contact suppression (opt-out and human paused)
	contact, err := w.store.GetContact(d.PhoneE164)
	if err != nil {
		return fmt.Errorf("outbox: contact lookup failed: %w", err)
	}

	if contact.OptOut {
		// Contact has opted out: suppress all automated workflows
		return nil
	}

	if contact.HumanPaused {
		// Human paused: suppress automated responses
		return nil
	}

	// 4. Greeting cooldown check
	if d.Action == core.Greeting {
		if !workflow.IsGreetingDue(contact.LastGreetingAt, time.Now().UTC(), w.cfg.GreetingCooldown, contact.HumanPaused) {
			// Cooldown not elapsed: suppress greeting
			return nil
		}
	}

	// 5. Create workflow plan and enqueue tasks
	plan := workflow.Create(d)
	if len(plan.Steps) == 0 {
		return nil
	}

	eventKey, err := instances.EventKey(d.InstanceID, "Message", d.MessageID)
	if err != nil {return err}
	tasks := make([]Task, 0, len(plan.Steps))
	for idx, step := range plan.Steps {
		taskID := fmt.Sprintf("%s-%02d-%s", eventKey, idx, step)
		task := Task{
			ID:           taskID,
			EventKey:     eventKey,
			InstanceID:   d.InstanceID,
			RecipientJID: d.ReplyJID,
			PhoneE164:    d.PhoneE164,
			CRMPhone:     d.CRMPhone,
			ContactName:  d.Name,
			Step:         step,
			State:        TaskPending,
			CreatedAt:    time.Now().UTC(),
		}

		switch step {
		case workflow.SendAdText:
			task.PayloadText = w.cfg.AdWelcomeText
		case workflow.SendGreetingImage:
			task.PayloadCaption = w.cfg.GreetingCaption
		case workflow.SendMenuReply:
			task.PayloadText = plan.Message
		}

		tasks = append(tasks, task)
	}
	// Commit all tasks and the dedup marker in one serialized disk operation.
	// On failure, the marker is not committed so the webhook can retry safely.
	isNew, err := w.store.CommitEventTasks(d.InstanceID, "Message", d.MessageID, tasks)
	if err != nil { return fmt.Errorf("outbox: cannot commit event tasks: %w", err) }
	if !isNew { return ErrEventDuplicate }
	return nil
}

// ProcessTask executes a single task with strict atomic state transitions:
// pending -> attempt_started (fsync) -> confirmed | uncertain
func (w *Worker) ProcessTask(ctx context.Context, taskID string) error {
	task, err := w.store.GetTask(taskID)
	if err != nil {
		return err
	}

	if task.State != TaskPending {
		// Never re-process tasks that are already attempt_started, confirmed, or uncertain.
		return nil
	}

	// Check contact opt-out once more before starting attempt
	contact, err := w.store.GetContact(task.PhoneE164)
	if err != nil { return fmt.Errorf("outbox: cannot read contact policy: %w", err) }
	if contact.OptOut || (contact.HumanPaused && task.Step != workflow.PauseHuman) {
		_, err = w.store.UpdateTaskState(task.ID, func(t *Task) error {
			t.State = TaskSuppressed
			t.Error = "suppressed: contact policy (opt-out or human handoff)"
			t.CompletedAt = time.Now().UTC()
			return nil
		})
		return err
	}

	// Do not transmit fake/missing media; suppress BEFORE attempt_started.
	if task.Step == workflow.SendAdAudio && len(w.cfg.AdAudioFixture) == 0 {
		return w.recordSuppressed(task.ID, "audio asset missing: no outbound request attempted")
	}
	if task.Step == workflow.SendGreetingImage && len(w.cfg.GreetingImageFixture) == 0 {
		return w.recordSuppressed(task.ID, "greeting image asset missing: no outbound request attempted")
	}
	// Transition ATOMICALLY to attempt_started with fsync before issuing outbound call
	task, err = w.store.UpdateTaskState(task.ID, func(t *Task) error {
		t.State = TaskAttemptStarted
		t.AttemptStartedAt = time.Now().UTC()
		return nil
	})
	if err != nil {
		return fmt.Errorf("outbox: failed to record attempt_started: %w", err)
	}

	// Sender routing is strict when multiple instances are configured.
	// Missing sender means suppress/fail closed, never use another account's key.
	sender := w.evolution
	if len(w.cfg.Senders) != 0 {
		sender = w.cfg.Senders[task.InstanceID]
	}
	// Execute specific step effect
	switch task.Step {
	case workflow.SendAdText:
		if sender == nil {
			return w.recordUncertain(task.ID, "evolution sender not configured")
		}
		extID, err := sender.SendText(ctx, task.RecipientJID, task.PayloadText)
		if err != nil {
			return w.recordUncertain(task.ID, err.Error())
		}
		return w.recordConfirmed(task.ID, extID)

	case workflow.SendAdAudio:
		if sender == nil {
			return w.recordUncertain(task.ID, "evolution sender not configured")
		}
		// SendMedia with type="audio" produces AudioMessage with PTT: true in Evolution Go 0.7.2
		extID, err := sender.SendMedia(ctx, task.RecipientJID, "audio", w.cfg.AdAudioFixture, "")
		if err != nil {
			// Note: Failure or timeout in audio does NOT block subsequent steps (like CRM)
			return w.recordUncertain(task.ID, err.Error())
		}
		return w.recordConfirmed(task.ID, extID)

	case workflow.ScheduleNatal:
		return w.executeScheduleNatal(ctx, task)

	case workflow.SendGreetingImage:
		if sender == nil {
			return w.recordUncertain(task.ID, "evolution sender not configured")
		}
		extID, err := sender.SendMedia(ctx, task.RecipientJID, "image", w.cfg.GreetingImageFixture, task.PayloadCaption)
		if err != nil {
			return w.recordUncertain(task.ID, err.Error())
		}
		// Record successful greeting timestamp on contact
		contact.LastGreetingAt = time.Now().UTC()
		if err := w.store.SaveContact(contact); err != nil {
			// Media was already sent; do not retry on contact state persistence failure.
			return w.recordUncertain(task.ID, "greeting sent; contact cooldown persistence failed")
		}
		return w.recordConfirmed(task.ID, extID)

	case workflow.SendMenuReply:
		if sender == nil {
			return w.recordUncertain(task.ID, "evolution sender not configured")
		}
		extID, err := sender.SendText(ctx, task.RecipientJID, task.PayloadText)
		if err != nil {
			return w.recordUncertain(task.ID, err.Error())
		}
		return w.recordConfirmed(task.ID, extID)

	case workflow.PauseHuman:
		contact.HumanPaused = true
		contact.HumanPausedAt = time.Now().UTC()
		if err := w.store.SaveContact(contact); err != nil {
			return w.recordUncertain(task.ID, "human pause persistence failed")
		}
		return w.recordConfirmed(task.ID, "internal-pause-applied")

	default:
		return w.recordUncertain(task.ID, fmt.Sprintf("unsupported step: %s", task.Step))
	}
}

func (w *Worker) executeScheduleNatal(ctx context.Context, task Task) error {
	// 1. Check operational authorization
	if !w.cfg.OperationalAuthorization {
		return w.recordSuppressed(task.ID, "crm: operational authorization required before dispatch")
	}

	// 2. Check explicit CRM instance ID
	if w.cfg.CRMInstanceID == nil || *w.cfg.CRMInstanceID <= 0 {
		return w.recordSuppressed(task.ID, "crm: explicit numeric message_instances.id (>0) required")
	}

	// 3. Check CRM phone format (10-11 digits, no DDI 55)
	if task.CRMPhone == "" {
		return w.recordSuppressed(task.ID, "crm: invalid phone format (must be DDD + 8/9 digits without 55)")
	}

	// 4. Deduplicate lead claim: shared across all instances of the tenant for campaign 416 in local BRT day
	claimed, err := w.store.ClaimLeadShared(w.cfg.Tenant, task.CRMPhone, crm.WithoutName, time.Now(), w.cfg.Location)
	if err != nil {
		return w.recordUncertain(task.ID, fmt.Sprintf("lead claim check error: %v", err))
	}
	if !claimed {
		return w.recordSuppressed(task.ID, "lead already claimed in current window; duplicate campaign dispatch skipped")
	}

	// 5. Build CRM request (Default campaign is strictly 416 - Natal 2026 without name)
	req, err := crm.BuildWithoutName(task.CRMPhone, w.cfg.CRMInstanceID)
	if err != nil {
		return w.recordSuppressed(task.ID, fmt.Sprintf("crm: build request rejected: %v", err))
	}

	if w.crm == nil {
		return w.recordUncertain(task.ID, "crm scheduler client not configured")
	}

	// 6. Execute CRM Schedule call
	scheduledID, err := w.crm.Schedule(ctx, req)
	if err != nil {
		// Timeout or unconfirmed outcome MUST transition to uncertain. Never auto-replay.
		return w.recordUncertain(task.ID, err.Error())
	}

	return w.recordConfirmed(task.ID, fmt.Sprintf("crm-schedule-%d", scheduledID))
}

func (w *Worker) recordConfirmed(taskID string, externalID string) error {
	_, err := w.store.UpdateTaskState(taskID, func(t *Task) error {
		t.State = TaskConfirmed
		t.ExternalID = externalID
		t.CompletedAt = time.Now().UTC()
		return nil
	})
	return err
}

func (w *Worker) recordUncertain(taskID string, reason string) error {
	_, err := w.store.UpdateTaskState(taskID, func(t *Task) error {
		t.State = TaskUncertain
		t.Error = reason
		t.CompletedAt = time.Now().UTC()
		return nil
	})
	return err
}

func (w *Worker) recordSuppressed(taskID string, reason string) error {
	_, err := w.store.UpdateTaskState(taskID, func(t *Task) error {
		t.State = TaskSuppressed
		t.Error = reason
		t.CompletedAt = time.Now().UTC()
		return nil
	})
	return err
}

// DrainPending processes all currently pending tasks in the store sequentially.
func (w *Worker) DrainPending(ctx context.Context) (int, error) {
	pending, _, err := w.store.RecoverTasks()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, t := range pending {
		if err := ctx.Err(); err != nil {
			return count, err
		}
		if err := w.ProcessTask(ctx, t.ID); err != nil {
			// Record error and continue to avoid worker halt
			count++
			continue
		}
		count++
	}

	return count, nil
}
