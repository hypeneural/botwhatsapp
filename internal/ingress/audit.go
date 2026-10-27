package ingress

import (
	"encoding/json"
	"strings"
	"unicode"

	"github.com/hypeneural/botwhatsapp/internal/core"
)

// InboundAudit intentionally persists only a short inbound text excerpt and
// categorical ad attribution. Never retain Evolution webhook payloads, tokens,
// image metadata, media content or request paths in the operational history.
type InboundAudit struct {
	MessageType   string `json:"message_type"`
	Text          string `json:"text,omitempty"`
	Source        string `json:"source"`
	HasAdContext  bool   `json:"has_ad_context"`
	TriggerMatch  bool   `json:"trigger_match"`
	TriggerText   string `json:"trigger_text,omitempty"`
}

func auditInbound(body []byte, decision core.Decision) InboundAudit {
	var e struct {
		Data struct {
			Message struct {
				Conversation string `json:"conversation"`
				Extended struct {
					Text string `json:"text"`
					ContextInfo struct {
						ExternalAdReply json.RawMessage `json:"externalAdReply"`
					} `json:"contextInfo"`
				} `json:"extendedTextMessage"`
				Image json.RawMessage `json:"imageMessage"`
				Video json.RawMessage `json:"videoMessage"`
				Audio json.RawMessage `json:"audioMessage"`
				Document json.RawMessage `json:"documentMessage"`
				Buttons json.RawMessage `json:"buttonsResponseMessage"`
			} `json:"Message"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &e)
	m := e.Data.Message
	a := InboundAudit{MessageType: "outro", Source: "normal"}
	switch {
	case strings.TrimSpace(m.Conversation) != "":
		a.MessageType, a.Text = "texto", auditExcerpt(m.Conversation)
	case strings.TrimSpace(m.Extended.Text) != "":
		a.MessageType, a.Text = "texto_estendido", auditExcerpt(m.Extended.Text)
	case len(m.Image) > 0:
		a.MessageType = "imagem"
	case len(m.Video) > 0:
		a.MessageType = "video"
	case len(m.Audio) > 0:
		a.MessageType = "audio"
	case len(m.Document) > 0:
		a.MessageType = "documento"
	case len(m.Buttons) > 0:
		a.MessageType = "botao"
	}
	a.HasAdContext = len(m.Extended.ContextInfo.ExternalAdReply) > 0 &&
		string(m.Extended.ContextInfo.ExternalAdReply) != "null"
	a.TriggerMatch = decision.Action == core.AdsProposal
	if a.TriggerMatch {
		a.TriggerText = a.Text
	}
	if a.HasAdContext || a.TriggerMatch {
		a.Source = "meta_ads"
	}
	return a
}

func auditExcerpt(s string) string {
	// Bound sensitive user-entered data before durable persistence / dashboard.
	var b strings.Builder
	runes := 0
	for _, r := range strings.TrimSpace(s) {
		if runes == 280 {
			break
		}
		if unicode.IsControl(r) {
			if unicode.IsSpace(r) {
				r = ' '
			} else {
				continue
			}
		}
		b.WriteRune(r)
		runes++
	}
	return b.String()
}
