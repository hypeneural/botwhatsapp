# Fontes verificadas — 21/10/2026
- Evolution Go: https://github.com/evolution-foundation/evolution-go/blob/main/pkg/routes/routes.go (rotas oficiais)
- https://github.com/evolution-foundation/evolution-go/blob/main/pkg/sendMessage/service/send_service.go (TextStruct, MediaStruct)
- https://github.com/evolution-foundation/evolution-go/blob/main/pkg/instance/service/instance_service.go (ConnectStruct)
- https://github.com/evolution-foundation/evolution-go/blob/main/pkg/events/webhook/webhook_producer.go (duplicidade global/por instância, tentativas)
- Skill de terceiros: https://github.com/6justgotme/evolution-go-skill (MIT, conferir contra versão instalada)
- Bridge Go: https://github.com/estevao-tarifa/evo-chatwoot-bridge-go (MIT, módulos evolution/client.go, messages.go e dedup; não instalar Chatwoot)
- CRM leitura: https://github.com/hypeneural/evydencia-crm/blob/main/routes/api.php e arquivos de CampaignScheduledController, SingleAuthentication, CampaignScheduledContactRule e CampaignScheduledContactsHelper.

**Limitação:** somente auditoria estática de GitHub. Não houve acesso à VPS; a revisão instalada, webhooks e IDs continuam não confirmados. O antiGravity deve tomar isso como BLOQUEIO para disparo real.

**Gatilhos e endpoints essenciais**: Evolution Go usa header `apikey` por instância; `POST /send/text` com `number,text`; `POST /send/media` com `number,type,url,caption`. `MESSAGE` é assinatura e `Message` é evento no payload. No CRM, `POST /api/campaigns/schedule/execute` exige `campaign` e `contacts`, enquanto `instance` é ID do CRM. O verificador de contatos aceita números com DDD e 10/11 dígitos, sem DDI. `Authorization` deve corresponder exatamente ao token interno configurado.

Não confundir HTTP 2xx de agendamento com entrega de WhatsApp. Não fazer segundo POST após timeout sem prova de não agendamento. Proteger o receptor de webhook por rede privada ou segredo independente.
