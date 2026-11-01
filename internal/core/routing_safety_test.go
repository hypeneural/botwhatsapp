package core

import "testing"

func TestSameBusinessPhoneNeverSchedulesEvenIfDirectionIsWrong(t *testing.T) {
	cfg := Config{AdsUUID: "ads", BusinessPhoneE164: "5548996425287",
		AdsTriggers: []string{"Quero conhecer os pacotes de Natal em Tijucas"}, AdsFallback: true}
	e := fixture("ads", "5548996425287@s.whatsapp.net",
		"Quero conhecer os pacotes de Natal em Tijucas", "echo-1")
	d := Select(e, cfg)
	if d.Action != Ignore || d.Reason != "own_business_number" || d.CRMPhone != "" {
		t.Fatalf("self-phone cannot be enrolled: %+v", d)
	}
	e.Data.Info.Chat = "5548999991000@s.whatsapp.net"
	if got := Select(e, cfg); got.Action != AdsProposal {
		t.Fatalf("actual external lead should still trigger exact ad: %+v", got)
	}
}

func TestVerifiedBusinessSenderBeatsIncorrectFromMeFlag(t *testing.T) {
	cfg := Config{AdsUUID: "ads", BusinessPhoneE164: "5548996425287",
		AdsTriggers: []string{"Quero conhecer os pacotes de Natal em Tijucas"}}
	e := fixture("ads", "5548999991000@s.whatsapp.net",
		"Quero conhecer os pacotes de Natal em Tijucas", "outbound-flag-wrong")
	e.Data.Info.Sender = "5548996425287@s.whatsapp.net"
	e.Data.Info.IsFromMe = false
	got := Select(e, cfg)
	if got.Action != OutboundObserved || got.PhoneE164 != "5548999991000" {
		t.Fatalf("self sender must never cause commercial proposal: %+v", got)
	}
}

func TestAdsPhoneNeverGreetsOrganicOrMenuEvenWithLegacyFlag(t *testing.T) {
	cfg := Config{AdsUUID: "ads", BusinessPhoneE164: "5548996425287",
		AdsTriggers: []string{"Quero conhecer os pacotes de Natal em Tijucas"}, AdsFallback: true}
	for _, text := range []string{"Oi", "Bom dia, sou a Maria", "Sim", "5", "Obrigado", "Qual o valor?"} {
		e := fixture("ads", "5548999991000@s.whatsapp.net", text, "msg-organic")
		got := Select(e, cfg)
		if got.Action != Ignore || got.Reason != "ads_unmatched_no_automation" {
			t.Fatalf("organic text %q cannot trigger greeting/menu: %+v", text, got)
		}
	}
}

func TestFromMeIsOnlyOutboundObservationNeverCampaign(t *testing.T) {
	cfg := Config{AdsUUID: "ads", BusinessPhoneE164: "5548996425287",
		AdsTriggers: []string{"Quero conhecer os pacotes de Natal em Tijucas"}}
	e := fixture("ads", "5548999991000@s.whatsapp.net",
		"Quero conhecer os pacotes de Natal em Tijucas", "own-to-customer")
	e.Data.Info.IsFromMe = true
	got := Select(e, cfg)
	if got.Action != OutboundObserved || got.PhoneE164 != "5548999991000" || got.CRMPhone != "48999991000" {
		t.Fatalf("own message should mark activity only: %+v", got)
	}
}

func TestPairedDeviceJIDUsesPhonePartWithoutInventingNinthDigit(t *testing.T) {
	cases := []struct {
		jid, phone string
		valid      bool
	}{
		{"554896425287:15@s.whatsapp.net", "554896425287", true},
		{"5548996425287:0@s.whatsapp.net", "5548996425287", true},
		{"554896425287@s.whatsapp.net", "554896425287", true},
		{"554896425287:1:15@s.whatsapp.net", "", false},
		{"554896425287:abc@s.whatsapp.net", "", false},
		{"554896425287:1234@s.whatsapp.net", "", false},
		{"554896425287@lid", "", false},
	}
	for _, tc := range cases {
		got, ok := JIDPhone(tc.jid)
		if ok != tc.valid || got != tc.phone {
			t.Errorf("invalid parsed-device JID: got=%q ok=%v", got, ok)
		}
	}
}

func TestOnlyExplicitOwnAliasesSuppressOldAndNewJIDForms(t *testing.T) {
	cfg := Config{AdsUUID: "ads", BusinessPhoneE164: "5548996425287",
		BusinessPhoneAliases: []string{"554896425287"},
		AdsTriggers:          []string{"Quero conhecer os pacotes de Natal em Tijucas"}}
	for _, jid := range []string{
		"554896425287:15@s.whatsapp.net",
		"554896425287@s.whatsapp.net",
		"5548996425287:2@s.whatsapp.net",
	} {
		e := fixture("ads", jid, "Quero conhecer os pacotes de Natal em Tijucas", "self-echo")
		result := Select(e, cfg)
		if result.Action != Ignore || result.Reason != "own_business_number" {
			t.Fatalf("verified own alias should not be a lead: %+v", result)
		}
	}
	e := fixture("ads", "5548999991234@s.whatsapp.net",
		"Quero conhecer os pacotes de Natal em Tijucas", "lead")
	e.Data.Info.Sender = "554896425287:15@s.whatsapp.net"
	result := Select(e, cfg)
	if result.Action != OutboundObserved {
		t.Fatalf("verified paired sender must suppress: %+v", result)
	}
	e.Data.Info.Sender = ""
	if res := Select(e, cfg); res.Action != AdsProposal {
		t.Fatalf("real customer JID should stay unchanged: %+v", res)
	}
}

func TestNoInferredNinthDigitAlias(t *testing.T) {
	cfg := Config{AdsUUID: "ads", BusinessPhoneE164: "5548996425287",
		AdsTriggers: []string{"Quero conhecer os pacotes de Natal em Tijucas"}}
	// Only a documented verified alias may bridge different BR WhatsApp JIDs.
	e := fixture("ads", "554896425287@s.whatsapp.net",
		"Quero conhecer os pacotes de Natal em Tijucas", "test-no-alias")
	d := Select(e, cfg)
	if d.Action != AdsProposal {
		t.Fatalf("without alias, no automatic equivalence may be invented: %+v", d)
	}
}
