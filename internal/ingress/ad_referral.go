package ingress

import (
	"encoding/json"
	"regexp"
	"strings"
)

// AdReference is sanitized evidence from Evolution Go. SourceID is a
// source ad/post reference, NOT a Meta Ad ID verified against Insights.
// Never retain ctwaClid, ad texts, URLs, images or raw provider payloads.
type AdReference struct {
	SourceID string `json:"ad_source_id,omitempty"`
	Evidence string `json:"ad_evidence,omitempty"`
}

var adSourceIDPattern = regexp.MustCompile(`^[0-9]{8,25}$`)

// parseAdReference understands Evolution Go 0.7.2 data.referral (protojson
// sourceID), and nested externalAdReply. Only sourceType=ad permits an ID.
func parseAdReference(referral, externalAdReply json.RawMessage) AdReference {
	for _, candidate := range []struct {
		raw json.RawMessage
		kind string
	}{{referral, "evolution_referral"}, {externalAdReply, "external_ad_reply"}} {
		if len(candidate.raw) == 0 || len(candidate.raw) > 8192 { continue }
		var data struct {
			SourceType string `json:"sourceType"`
			SnakeType string `json:"source_type"`
			SourceID string `json:"sourceID"`
			CamelID string `json:"sourceId"`
			SnakeID string `json:"source_id"`
		}
		if json.Unmarshal(candidate.raw, &data) != nil { continue }
		kind := strings.TrimSpace(data.SourceType)
		if kind == "" { kind = strings.TrimSpace(data.SnakeType) }
		if !strings.EqualFold(kind, "ad") { continue }
		id := strings.TrimSpace(data.SourceID)
		if id == "" { id = strings.TrimSpace(data.CamelID) }
		if id == "" { id = strings.TrimSpace(data.SnakeID) }
		if adSourceIDPattern.MatchString(id) {
			return AdReference{SourceID: id, Evidence: candidate.kind}
		}
		return AdReference{Evidence: candidate.kind}
	}
	return AdReference{}
}
