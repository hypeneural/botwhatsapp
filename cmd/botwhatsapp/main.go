package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/crm"
	"github.com/hypeneural/botwhatsapp/internal/dashboard"
	"github.com/hypeneural/botwhatsapp/internal/evolution"
	"github.com/hypeneural/botwhatsapp/internal/ingress"
	"github.com/hypeneural/botwhatsapp/internal/instances"
	"github.com/hypeneural/botwhatsapp/internal/outbox"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func readAsset(envKey string) []byte {
	path := strings.TrimSpace(os.Getenv(envKey))
	if path == "" {
		return nil // No fixture is ever substituted for a missing real file.
	}
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("configured media file for %s is unavailable", envKey)
	}
	limit := 8 << 20
	if envKey == "BOT_ADS_DOCUMENT_FILE" {
		limit = 25 << 20
	}
	if len(b) == 0 || len(b) > limit {
		log.Fatalf("configured media file for %s has an invalid size", envKey)
	}
	return b
}

func main() {
	enabled := os.Getenv("BOT_ENABLED") == "true"
	outboundEnabled := os.Getenv("BOT_OUTBOUND_ENABLED") == "true"
	opAuth := os.Getenv("BOT_OPERATIONAL_AUTHORIZATION") == "true"
	if outboundEnabled && (!enabled || !opAuth) {
		log.Fatal("outbound must have BOT_ENABLED=true and BOT_OPERATIONAL_AUTHORIZATION=true")
	}
	dataDir := env("BOT_DATA_DIR", "./data/spool")
	adsUUID := strings.TrimSpace(os.Getenv("BOT_ADS_INSTANCE_UUID"))
	greetingUUID := strings.TrimSpace(os.Getenv("BOT_GREETING_INSTANCE_UUID"))

	triggers, err := loadAdTriggers()
	if err != nil {
		log.Fatalf("invalid ad trigger configuration: %T", err)
	}

	defs := make([]instances.Definition, 0, 2)
	secrets := make(map[string]string)
	senders := make(map[string]outbox.EvolutionSender)
	evoBaseURL := strings.TrimSpace(os.Getenv("EVOLUTION_GO_BASE_URL"))
	if outboundEnabled && evoBaseURL == "" {
		log.Fatal("EVOLUTION_GO_BASE_URL must be explicitly configured for outbound")
	}
	if adsUUID != "" {
		defs = append(defs, instances.Definition{
			Slug: "ads", InstanceUUID: adsUUID,
			BusinessPhoneE164: env("BOT_ADS_PHONE", "554832200440"),
			Role:              instances.Ads, Enabled: true, TokenEnv: "EVOLUTION_ADS_APIKEY",
			WebhookSecretEnv: "BOT_ADS_WEBHOOK_SECRET",
			AdsTriggers:      triggers, AdsFallback: env("BOT_ADS_FALLBACK_MENU", "true") == "true", CRMAllowed: true,
		})
		secrets["ads"] = os.Getenv("BOT_ADS_WEBHOOK_SECRET")
		if outboundEnabled {
			token := os.Getenv("EVOLUTION_ADS_APIKEY")
			if token == "" {
				log.Fatal("outbound ads instance requires EVOLUTION_ADS_APIKEY")
			}
			senders[adsUUID] = evolution.Client{BaseURL: evoBaseURL, Token: token}
		}
	}
	if greetingUUID != "" {
		defs = append(defs, instances.Definition{
			Slug: "greeting", InstanceUUID: greetingUUID,
			BusinessPhoneE164: env("BOT_GREETING_PHONE", "5548996425287"),
			Role:              instances.Greeting, Enabled: true, TokenEnv: "EVOLUTION_GREETING_APIKEY",
			WebhookSecretEnv: "BOT_GREETING_WEBHOOK_SECRET",
		})
		secrets["greeting"] = os.Getenv("BOT_GREETING_WEBHOOK_SECRET")
		if outboundEnabled {
			token := os.Getenv("EVOLUTION_GREETING_APIKEY")
			if token == "" {
				log.Fatal("outbound greeting instance requires EVOLUTION_GREETING_APIKEY")
			}
			senders[greetingUUID] = evolution.Client{BaseURL: evoBaseURL, Token: token}
		}
	}
	if outboundEnabled && adsUUID != "" && greetingUUID != "" &&
		os.Getenv("EVOLUTION_ADS_APIKEY") == os.Getenv("EVOLUTION_GREETING_APIKEY") {
		log.Fatal("outbound accounts require distinct per-instance API keys")
	}
	var reg *instances.Registry
	if len(defs) > 0 {
		var err error
		reg, err = instances.New(defs)
		if err != nil {
			// Never fall back to legacy routing on a failed registry.
			log.Fatalf("invalid multi-instance registry: %v", err)
		}
	}
	if enabled && reg == nil {
		log.Fatal("BOT_ENABLED requires at least one explicitly mapped Evolution Go instance")
	}
	for slug, secret := range secrets {
		if enabled && len(secret) < 32 {
			log.Fatalf("webhook for role %s requires a 32+ character secret", slug)
		}
	}
	coreCfg := core.Config{
		AdsUUID: adsUUID, GreetingUUID: greetingUUID, AdsTriggers: triggers,
		AdsFallback: env("BOT_ADS_FALLBACK_MENU", "true") == "true",
	}

	var worker *outbox.Worker
	if outboundEnabled {
		store, err := outbox.NewStore(dataDir)
		if err != nil {
			log.Fatal("cannot initialize durable outbox: ", err)
		}
		var crmInstanceID *int
		if id, err := strconv.Atoi(os.Getenv("EVIDENCIA_CRM_OUTBOUND_MESSAGE_INSTANCE_ID")); err == nil && id > 0 {
			crmInstanceID = &id
		}
		if adsUUID != "" && crmInstanceID == nil {
			log.Fatal("ads CRM dispatch requires explicit positive EVIDENCIA_CRM_OUTBOUND_MESSAGE_INSTANCE_ID")
		}
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			log.Fatal("America/Sao_Paulo timezone is required for shared CRM claim")
		}
		if adsUUID != "" && strings.TrimSpace(os.Getenv("EVIDENCIA_CRM_AUTHORIZATION")) == "" {
			log.Fatal("ads campaign requires EVIDENCIA_CRM_AUTHORIZATION")
		}
		crmClient := crm.Client{
			BaseURL:       env("EVIDENCIA_CRM_BASE_URL", "https://evydencia.com"),
			Authorization: os.Getenv("EVIDENCIA_CRM_AUTHORIZATION"),
		}
		audio := readAsset("BOT_ADS_AUDIO_FILE")
		image := readAsset("BOT_GREETING_IMAGE_FILE")
		adImage := readAsset("BOT_ADS_IMAGE_FILE")
		adVideo := readAsset("BOT_ADS_VIDEO_FILE")
		adDoc := readAsset("BOT_ADS_DOCUMENT_FILE")
		if len(adDoc) > 0 && !strings.HasPrefix(string(adDoc), "%PDF-") {
			log.Fatal("BOT_ADS_DOCUMENT_FILE must be a valid PDF (starting with %PDF-)")
		}
		// Validate configured files before allowing live media. Explicitly
		// missing files remain suppressed by the outbox; never send a fake.
		if len(audio) > 0 && (!strings.HasPrefix(string(audio), "OggS") || !strings.Contains(string(audio[:min(len(audio), 128)]), "OpusHead")) {
			log.Fatal("BOT_ADS_AUDIO_FILE must be OGG Opus/WhatsApp PTT")
		}
		if len(image) > 0 && !(strings.HasPrefix(string(image), "\x89PNG\r\n\x1a\n") ||
			(len(image) >= 3 && image[0] == 0xff && image[1] == 0xd8 && image[2] == 0xff)) {
			log.Fatal("BOT_GREETING_IMAGE_FILE must be PNG or JPEG")
		}
		if len(adImage) > 0 && !(strings.HasPrefix(string(adImage), "\x89PNG\r\n\x1a\n") ||
			(len(adImage) >= 3 && adImage[0] == 0xff && adImage[1] == 0xd8 && adImage[2] == 0xff)) {
			log.Fatal("BOT_ADS_IMAGE_FILE must be PNG or JPEG")
		}
		if len(adVideo) > 0 && !(len(adVideo) >= 12 && string(adVideo[4:8]) == "ftyp") {
			log.Fatal("BOT_ADS_VIDEO_FILE must be MP4 (up to 8MiB)")
		}
		adsCooldownHours, err := strconv.Atoi(env("BOT_ADS_FLOW_COOLDOWN_HOURS", "24"))
		if err != nil || adsCooldownHours < 1 || adsCooldownHours > 168 {
			log.Fatal("BOT_ADS_FLOW_COOLDOWN_HOURS must be between 1 and 168")
		}
		adWelcome := env("BOT_ADS_RESPONSE_TEXT", "Oi, tudo bem?\nSeja bem vindo ao estúdio de fotos de natal Evydência\nVeja mais fotos no nosso instagram.com/estudioevydenciaa")
		if path := os.Getenv("BOT_ADS_RESPONSE_TEXT_FILE"); path != "" {
			raw, err := os.ReadFile(path)
			if err != nil || strings.TrimSpace(string(raw)) == "" {
				log.Fatal("invalid ad response text file")
			}
			adWelcome = string(raw)
		}
		adClosing := env("BOT_ADS_CLOSING_TEXT", "O nosso outro WhatsApp vai entrar em contato com mais informações também, mas qualquer dúvida pode falar.")
		if path := os.Getenv("BOT_ADS_CLOSING_TEXT_FILE"); path != "" {
			raw, err := os.ReadFile(path)
			if err != nil || strings.TrimSpace(string(raw)) == "" {
				log.Fatal("invalid ad closing text file")
			}
			adClosing = string(raw)
		}
		cfg := outbox.Config{
			DataDir: dataDir, GreetingCooldown: 24 * time.Hour, AdsFlowCooldown: time.Duration(adsCooldownHours) * time.Hour,
			CRMInstanceID:            crmInstanceID,
			OperationalAuthorization: opAuth,
			AdReactionEmoji:          env("BOT_ADS_REACTION_EMOJI", "🎄"),
			AdAudioFixture:           audio,
			AdDocumentFixture:        adDoc,
			AdDocumentFilename:       env("BOT_ADS_DOCUMENT_FILENAME", "Proposta_Natal.pdf"),
			AdDocumentURL:            env("BOT_ADS_DOCUMENT_URL", "http://bot:8788/static/Proposta_Natal.pdf"),
			AdDocumentCaption:        env("BOT_ADS_DOCUMENT_CAPTION", "Proposta de Natal 2026 - Estúdio Evydência"),
			GreetingImageFixture:     image,
			AdImageFixture:           adImage,
			AdVideoFixture:           adVideo,
			AdWelcomeText:            adWelcome,
			AdClosingText:            adClosing,
			Tenant:                   env("BOT_TENANT", "estudio-evydencia"),
			Location:                 loc,
			TaskRetention:            7 * 24 * time.Hour,
			MaxTasksCount:            10000,
			Senders:                  senders,
		}
		if path := os.Getenv("BOT_ADS_IMAGE_CAPTION_FILE"); path != "" {
			raw, err := os.ReadFile(path)
			if err != nil || strings.TrimSpace(string(raw)) == "" {
				log.Fatal("invalid ad image caption file")
			}
			cfg.AdImageCaption = string(raw)
		} else {
			cfg.AdImageCaption = env("BOT_ADS_IMAGE_CAPTION", "🎄 Natal + Ano Novo no Estúdio Evidência, em Tijucas! Veja nossos pacotes: https://evydencia.com/catalogo/natal")
		}
		if path := os.Getenv("BOT_ADS_VIDEO_CAPTION_FILE"); path != "" {
			raw, err := os.ReadFile(path)
			if err != nil || strings.TrimSpace(string(raw)) == "" {
				log.Fatal("invalid ad video caption file")
			}
			cfg.AdVideoCaption = string(raw)
		} else {
			cfg.AdVideoCaption = os.Getenv("BOT_ADS_VIDEO_CAPTION")
		}
		if path := os.Getenv("BOT_GREETING_CAPTION_FILE"); path != "" {
			raw, err := os.ReadFile(path)
			if err != nil || strings.TrimSpace(string(raw)) == "" {
				log.Fatal("BOT_GREETING_CAPTION_FILE must contain text")
			}
			cfg.GreetingCaption = string(raw)
		}
		worker, err = outbox.NewWorker(cfg, store, nil, crmClient)
		if err != nil {
			log.Fatal("outbox worker init: ", err)
		}
	}

	var onDecision func(core.Decision) error
	if worker != nil {
		onDecision = func(d core.Decision) error {
			err := worker.IngestDecision(d)
			if errors.Is(err, outbox.ErrEventDuplicate) {
				return nil
			}
			if err != nil {
				log.Printf("durable decision ingestion unavailable: %T", err)
			}
			return err
		}
	}
	// Preview mode never enqueues production tasks. In outbound mode it must
	// synchronously persist tasks BEFORE the webhook returns HTTP 202.
	maxBodyBytes := int64(32 << 20)
	if s := os.Getenv("BOT_MAX_BODY_BYTES"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil && v >= 1<<20 && v <= 100<<20 {
			maxBodyBytes = v
		}
	}
	dashHandler := dashboard.NewHandler(dataDir, env("BOT_DASHBOARD_TOKEN", ""))
	h, err := ingress.New(ingress.Settings{
		Enabled: enabled, AdsSecret: secrets["ads"],
		GreetingSecret: secrets["greeting"],
		SpoolDir:       dataDir, Core: coreCfg, OnDecision: onDecision,
		Registry: reg, Secrets: secrets,
		StaticDir:        env("BOT_ASSETS_DIR", "./assets"),
		MaxBodyBytes:     maxBodyBytes,
		DashboardHandler: dashHandler,
	})
	if err != nil {
		log.Fatalf("invalid webhook configuration: %v", err)
	}

	if worker != nil {
		// One bounded runner: prompt notifications for new tasks, a slow
		// recovery sweep for dropped notifications/restarts, and separate
		// maintenance. Never scan every task 2 times per second.
		go func() {
			ctx := context.Background()
			if _, err := worker.DrainPending(ctx); err != nil {
				log.Printf("initial outbox recovery unavailable: %T", err)
			}
			recovery := time.NewTicker(60 * time.Second)
			maintenance := time.NewTicker(6 * time.Hour)
			defer recovery.Stop()
			defer maintenance.Stop()
			for {
				select {
				case task := <-worker.PendingTasks():
					if err := worker.ProcessTask(ctx, task.ID); err != nil {
						log.Printf("outbox task failed: %T", err)
					}
				case <-recovery.C:
					if _, err := worker.DrainPending(ctx); err != nil {
						log.Printf("outbox recovery unavailable: %T", err)
					}
				case <-maintenance.C:
					if _, err := worker.PruneCompleted(); err != nil {
						log.Printf("outbox maintenance unavailable: %T", err)
					}
				}
			}
		}()
	}
	addr := env("BOT_LISTEN_ADDR", "127.0.0.1:8788")
	log.Printf("bot started: enabled=%t outbound=%t listen=%s", enabled, outboundEnabled, addr)
	srv := &http.Server{
		Addr: addr, Handler: h,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second,
		MaxHeaderBytes: 8192,
	}
	log.Fatal(srv.ListenAndServe())
}
