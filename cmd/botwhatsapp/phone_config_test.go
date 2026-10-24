package main

import "testing"

func TestAdsPhoneExplicit5287Only(t *testing.T) {
    cases := []struct{
        name string
        value string
        ok bool
    }{
        {"valid", "5548996425287", true},
        {"whitespace", " 5548996425287 ", true},
        {"missing", "", false},
        {"legacy", "554832200440", false},
        {"different", "5548999999999", false},
        {"short", "5287", false},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            got, err := adsPhoneFromEnv(func(k string) string {
                if k != "BOT_ADS_PHONE" { t.Fatal("unexpected env key") }
                return c.value
            })
            if c.ok && (err != nil || got != activeAdsPhoneE164) {
                t.Fatalf("expected approved phone (redacted), got error %v", err)
            }
            if !c.ok && err == nil {
                t.Fatal("unapproved ads phone was accepted")
            }
        })
    }
}

func TestGreetingRoleMustHaveDistinctExplicitPhone(t *testing.T) {
    for _, tc := range []struct{
        label, value, ads string
        ok bool
    }{
        {"missing", "", activeAdsPhoneE164, false},
        {"same", activeAdsPhoneE164, activeAdsPhoneE164, false},
        {"separate", "5548999999999", activeAdsPhoneE164, true},
    } {
        t.Run(tc.label,func(t *testing.T){
            _, err := greetingPhoneFromEnv(func(k string) string { return tc.value }, tc.ads)
            if (err == nil) != tc.ok {t.Fatalf("expected ok=%v, got err=%v", tc.ok,err)}
        })
    }
}
