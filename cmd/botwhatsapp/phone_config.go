package main

import (
	"errors"
	"strings"
)

const activeAdsPhoneE164 = "5548996425287"

// adsPhoneFromEnv fail-closes when an ads role is configured with no explicitly
// approved number. It NEVER calls the network or sends a message.
func adsPhoneFromEnv(getenv func(string) string) (string, error) {
	p := strings.TrimSpace(getenv("BOT_ADS_PHONE"))
	if p == "" {
		return "", errors.New("BOT_ADS_PHONE must be explicitly configured for enabled ads role")
	}
	if p != activeAdsPhoneE164 {
		return "", errors.New("BOT_ADS_PHONE differs from approved ads-number contract (ending 5287)")
	}
	return p, nil
}

// greetingPhoneFromEnv cannot silently reuse the ads role's number.
// The ads role already handles the greeting/menu fallback on unmatched texts.
func greetingPhoneFromEnv(getenv func(string) string, adsPhone string) (string, error) {
	p := strings.TrimSpace(getenv("BOT_GREETING_PHONE"))
	if p == "" {
		return "", errors.New("BOT_GREETING_PHONE must be explicitly configured for enabled greeting role")
	}
	if p == adsPhone && adsPhone != "" {
		return "", errors.New("greeting role may not duplicate ads phone; use ads fallback greeting on 5287")
	}
	return p, nil
}
