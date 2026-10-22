---
name: crm-readonly-contract
description: Integrar como CONSUMIDOR da API de agendamento do Evydência CRM, por padrão campanha Natal 2026 SEM NOME 416, sem nenhuma alteração no CRM.
---
# CRM — skill cirúrgica para Flash Lite

Repositório `hypeneural/evydencia-crm` é **SOMENTE LEITURA**. Comece por `docs/CRM-API-416-PLAYBOOK.md`. Não rodar POST de produção na auditoria.

1. **Padrão fixo sem nome:** `campaign=416`, `contacts="DDDxxxxxxxxx"`. Nunca selecionar 415 automaticamente por `PushName`; nome de WhatsApp não autoriza variante com nome.
2. `type="campaign"`, `use_leads_system=false`, `instance` é número inteiro positivo de `message_instances.id` no CRM, explícito e validado. Jamais enviar null/usar favorito.
3. `POST https://evydencia.com/api/campaigns/schedule/execute`; header `Authorization` é token interno raw, SEM Bearer; body JSON. `GET /api/campaigns/schedule/search?campaign.id=416&contacts.phone=DDD...` para reconciliação READ ONLY.
4. **Atenção crítica:** `JsonResource::withoutWrapping()` em `AppServiceProvider.php`: retorno de sucesso tem `id`, `campaign`, `instance.id`, `status.name` na RAIZ. O bot tolera também formato envelopado, mas não exige `data.id`.
5. 2xx sem ID positivo ou divergência de campanha/instância = `uncertain`. 401/422 etc = `rejected`. Timeout, 5xx, 429 = `uncertain`, NÃO fazer POST outra vez automaticamente.
6. GET de pesquisa pode retornar lista raiz; um resultado vazio não prova falha do POST anterior. Exigir reconciliação manual para caso incerto.
7. Arquivos editáveis **só no bot**: `internal/crm/contract.go`, `internal/crm/client.go`, testes adjacentes. `go test ./internal/crm`; sem Docker da VPS.
8. Distinguir agendado, aceito no provedor, entregue e lido. Somente agendado é provado pelo POST.

Fonte do CRM: `routes/api.php`, `SingleAuthentication.php`, `CampaignScheduledRequest.php`, `CampaignScheduledService.php`, `CampaignScheduledResource.php`, `AppServiceProvider.php`, `Natal2026ProposalManifest.php`, exemplos Scribe.\n