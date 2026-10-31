package main

import "testing"

func TestAdsPhoneExplicitAndChangeable(t *testing.T) {
    cases := []struct {
        name, value string
        ok bool
    }{
        {"current", "5548996425287", true},
        {"current with spaces", " 5548996425287 ", true},
        {"other valid local", "5548999999999", true},
        {"historical number not implicitly blocked by code", "554832200440", true},
        {"missing", "", false},
        {"no DDI", "48999999999", false},
        {"too short", "5287", false},
        {"foreign country code", "13475551234", false},
        {"bad characters", "55+48999999999", false},
        {"has suffix", "5548999999999@s.whatsapp.net", false},
    }
    for _, tc := range cases {
        t.Run(tc.name,func(t *testing.T) {
            phone, err := adsPhoneFromEnv(func(key string) string {
                if key != "BOT_ADS_PHONE" {t.Fatalf("wrong key %s",key)}
                return tc.value
            })
            if (err == nil) != tc.ok {t.Fatalf("accepted=%t want %t",err==nil,tc.ok)}
            if tc.ok && phone == "" {t.Fatal("valid configured number was lost")}
        })
    }
}

func TestGreetingRoleCannotShareAdsOrUseInvalidNumber(t *testing.T) {
    for _, tc := range []struct {
        name, ads, greeting string
        ok bool
    }{
        {"separate", "5548996425287", "5548999999999", true},
        {"missing", "5548996425287", "", false},
        {"same number", "5548996425287", "5548996425287", false},
        {"invalid", "5548996425287", "5287", false},
    } {
        t.Run(tc.name,func(t *testing.T) {
            _,err := greetingPhoneFromEnv(func(k string) string{return tc.greeting},tc.ads)
            if (err==nil)!=tc.ok {t.Fatalf("accepted=%t want %t",err==nil,tc.ok)}
        })
    }
}
