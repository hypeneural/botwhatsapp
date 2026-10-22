package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hypeneural/botwhatsapp/internal/core"
	"github.com/hypeneural/botwhatsapp/internal/crm"
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

func main() {
	enabled := os.Getenv("BOT_ENABLED") == "true"
	outboundEnabled := os.Getenv("BOT_OUTBOUND_ENABLED") == "true"
	dataDir := env("BOT_DATA_DIR", "./data/spool")

	triggers := []string{"Quero a proposta de Natal 2026", "proposta de natal", "natal 2026"}
	if s := strings.TrimSpace(os.Getenv("BOT_ADS_TRIGGERS")); s != "" {
		triggers = nil
		for _, x := range strings.Split(s, "|") {
			if x = strings.TrimSpace(x); x != "" {
				triggers = append(triggers, x)
			}
		}
	}

	adsUUID := os.Getenv("BOT_ADS_INSTANCE_UUID")
	greetingUUID := os.Getenv("BOT_GREETING_INSTANCE_UUID")
	adsSecret := os.Getenv("BOT_ADS_WEBHOOK_SECRET")
	greetingSecret := os.Getenv("BOT_GREETING_WEBHOOK_SECRET")

	// Multi-instance Registry configuration
	var reg *instances.Registry
	secretsMap := make(map[string]string)
	if adsUUID != "" && greetingUUID != "" {
		defs := []instances.Definition{
			{
				Slug:              "ads",
				InstanceUUID:      adsUUID,
				BusinessPhoneE164: env("BOT_ADS_PHONE", "554832200440"),
				Role:              instances.Ads,
				Enabled:           true,
				TokenEnv:          "EVOLUTION_ADS_APIKEY",
				WebhookSecretEnv:  "BOT_ADS_WEBHOOK_SECRET",
				AdsTriggers:       triggers,
				CRMAllowed:        true,
			},
			{
				Slug:              "greeting",
				InstanceUUID:      greetingUUID,
				BusinessPhoneE164: env("BOT_GREETING_PHONE", "5548996425287"),
				Role:              instances.Greeting,
				Enabled:           true,
				TokenEnv:          "EVOLUTION_GREETING_APIKEY",
				WebhookSecretEnv:  "BOT_GREETING_WEBHOOK_SECRET",
			},
		}
		var err error
		reg, err = instances.New(defs)
		if err != nil {
			log.Printf("warning: multi-instance registry init: %v (falling back to legacy routing)", err)
		} else {
			secretsMap["ads"] = adsSecret
			secretsMap["greeting"] = greetingSecret
		}
	}

	coreCfg := core.Config{
		AdsUUID:      adsUUID,
		GreetingUUID: greetingUUID,
		AdsTriggers:  triggers,
		AdsFallback:  os.Getenv("BOT_ADS_FALLBACK_MENU") == "true",
	}

	var worker *outbox.Worker
	if outboundEnabled {
		store, err := outbox.NewStore(dataDir)
		if err != nil {
			log.Fatalf("outbox store init error: %v", err)
		}

		evoClient := evolution.Client{
			BaseURL: env("EVOLUTION_GO_BASE_URL", "https://wa-api.etijucas.com.br"),
			Token:   os.Getenv("EVOLUTION_ADS_APIKEY"),
		}

		crmClient := crm.Client{
			BaseURL:       env("EVIDENCIA_CRM_BASE_URL", "https://evydencia.com"),
			Authorization: os.Getenv("EVIDENCIA_CRM_AUTHORIZATION"),
		}

		var crmInstanceID *int
		if idStr := os.Getenv("EVIDENCIA_CRM_OUTBOUND_MESSAGE_INSTANCE_ID"); idStr != "" {
			if id, err := strconv.Atoi(idStr); err == nil && id > 0 {
				crmInstanceID = &id
			}
		}

		opAuth := os.Getenv("BOT_OPERATIONAL_AUTHORIZATION") == "true"

		var audioFixture []byte
		audioPath := env("BOT_ADS_AUDIO_FILE", "testdata/test_audio.ogg")
		if b, err := os.ReadFile(audioPath); err == nil && len(b) > 0 {
			audioFixture = b
		}

		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			loc = time.FixedZone("BRT", -3*3600)
		}

		outboxCfg := outbox.Config{
			DataDir:                  dataDir,
			GreetingCooldown:         24 * time.Hour,
			CRMInstanceID:            crmInstanceID,
			OperationalAuthorization: opAuth,
			AdAudioFixture:           audioFixture,
			AdWelcomeText:            env("BOT_ADS_RESPONSE_TEXT", "Oii! Tudo bem? 🎄❤️ Que alegria receber seu contato! Vamos enviar a proposta de Natal 2026 por outro WhatsApp oficial do Estúdio Evidência."),
			Tenant:                   env("BOT_TENANT", "estudio-evydencia"),
			Location:                 loc,
			TaskRetention:            7 * 24 * time.Hour,
			MaxTasksCount:            10000,
		}

		worker, err = outbox.NewWorker(outboxCfg, store, evoClient, crmClient)
		if err != nil {
			log.Fatalf("outbox worker init error: %v", err)
		}

		// Run background consumer loop
		go func() {
			log.Println("durable outbox worker started in background")
			for {
				if _, err := worker.DrainPending(context.Background()); err != nil {
					log.Printf("worker drain error: %v", err)
				}
				time.Sleep(500 * time.Millisecond)
			}
		}()
	}

	var onDecision func(core.Decision)
	if worker != nil {
		onDecision = func(d core.Decision) {
			if err := worker.IngestDecision(d); err != nil {
				log.Printf("outbox ingest decision error: %v", err)
			}
		}
	}

	ingressSettings := ingress.Settings{
		Enabled:        enabled,
		AdsSecret:      adsSecret,
		GreetingSecret: greetingSecret,
		SpoolDir:       dataDir,
		Core:           coreCfg,
		OnDecision:     onDecision,
	}
	if reg != nil && len(secretsMap["ads"]) >= 32 && len(secretsMap["greeting"]) >= 32 {
		ingressSettings.Registry = reg
		ingressSettings.Secrets = secretsMap
	}

	h, err := ingress.New(ingressSettings)
	if err != nil {
		log.Fatalf("configuration invalid: %v", err)
	}

	addr := env("BOT_LISTEN_ADDR", "0.0.0.0:8788")
	log.Printf("webhook intake listening on %s; enabled=%t outbound=%t", addr, enabled, outboundEnabled)
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8192,
	}
	log.Fatal(srv.ListenAndServe())
}
