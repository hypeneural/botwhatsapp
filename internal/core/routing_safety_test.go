package core

import "testing"

func TestSameBusinessPhoneNeverSchedulesEvenIfDirectionIsWrong(t *testing.T) {
	cfg:=Config{AdsUUID:"ads",BusinessPhoneE164:"5548996425287",
		AdsTriggers:[]string{"Quero conhecer os pacotes de Natal em Tijucas"}, AdsFallback:true}
	e:=fixture("ads","5548996425287@s.whatsapp.net",
		"Quero conhecer os pacotes de Natal em Tijucas","echo-1")
	d:=Select(e,cfg)
	if d.Action!=Ignore||d.Reason!="own_business_number"||d.CRMPhone!="" {
		t.Fatalf("self-phone cannot be enrolled: %+v",d)
	}
	e.Data.Info.Chat="5548999991000@s.whatsapp.net"
	if got:=Select(e,cfg);got.Action!=AdsProposal{
		t.Fatalf("actual external lead should still trigger exact ad: %+v",got)
	}
}

func TestVerifiedBusinessSenderBeatsIncorrectFromMeFlag(t *testing.T) {
	cfg:=Config{AdsUUID:"ads",BusinessPhoneE164:"5548996425287",
		AdsTriggers:[]string{"Quero conhecer os pacotes de Natal em Tijucas"}}
	e:=fixture("ads","5548999991000@s.whatsapp.net",
		"Quero conhecer os pacotes de Natal em Tijucas","outbound-flag-wrong")
	e.Data.Info.Sender="5548996425287@s.whatsapp.net"
	e.Data.Info.IsFromMe=false
	got:=Select(e,cfg)
	if got.Action!=OutboundObserved||got.PhoneE164!="5548999991000"{
		t.Fatalf("self sender must never cause commercial proposal: %+v",got)
	}
}

func TestAdsPhoneNeverGreetsOrganicOrMenuEvenWithLegacyFlag(t *testing.T) {
	cfg:=Config{AdsUUID:"ads",BusinessPhoneE164:"5548996425287",
		AdsTriggers:[]string{"Quero conhecer os pacotes de Natal em Tijucas"}, AdsFallback:true}
	for _,text:=range []string{"Oi","Bom dia, sou a Maria","Sim","5","Obrigado","Qual o valor?"} {
		e:=fixture("ads","5548999991000@s.whatsapp.net",text,"msg-organic")
		got:=Select(e,cfg)
		if got.Action!=Ignore||got.Reason!="ads_unmatched_no_automation"{
			t.Fatalf("organic text %q cannot trigger greeting/menu: %+v",text,got)
		}
	}
}

func TestFromMeIsOnlyOutboundObservationNeverCampaign(t *testing.T) {
	cfg:=Config{AdsUUID:"ads",BusinessPhoneE164:"5548996425287",
		AdsTriggers:[]string{"Quero conhecer os pacotes de Natal em Tijucas"}}
	e:=fixture("ads","5548999991000@s.whatsapp.net",
		"Quero conhecer os pacotes de Natal em Tijucas","own-to-customer")
	e.Data.Info.IsFromMe=true
	got:=Select(e,cfg)
	if got.Action!=OutboundObserved || got.PhoneE164!="5548999991000" || got.CRMPhone!="48999991000" {
		t.Fatalf("own message should mark activity only: %+v",got)
	}
}
