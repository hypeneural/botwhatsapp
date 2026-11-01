package main

import (
	"errors"
	"strings"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

// configuredBusinessPhone validates an explicit Brazilian E.164 number. The
// phone is *not* a routing authority by itself: the Antigravity preflight must
// verify it belongs to the configured Evolution instance UUID and API key
// before changing any webhook or enabling outbound.
func configuredBusinessPhone(getenv func(string) string, key string) (string, error) {
	phone := strings.TrimSpace(getenv(key))
	if phone == "" {
		return "", errors.New(key + " is required for an enabled instance")
	}
	if parsed, ok := core.JIDPhone(phone + "@s.whatsapp.net"); !ok || parsed != phone {
		return "", errors.New(key + " must be a Brazilian E.164 number with DDI 55 and DDD")
	}
	return phone, nil
}

func adsPhoneFromEnv(getenv func(string) string) (string, error) {
	return configuredBusinessPhone(getenv, "BOT_ADS_PHONE")
}

// Greeting is a separate role and must never share the Ads line.
func greetingPhoneFromEnv(getenv func(string) string, adsPhone string) (string, error) {
	phone, err := configuredBusinessPhone(getenv, "BOT_GREETING_PHONE")
	if err != nil {
		return "", err
	}
	if phone == adsPhone && adsPhone != "" {
		return "", errors.New("greeting role cannot share the configured Ads number")
	}
	return phone, nil
}
