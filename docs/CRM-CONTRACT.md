# Contrato CRM existente — consumo externo apenas

Fonte: `hypeneural/evydencia-crm` (SOMENTE LEITURA). Arquivos: `routes/api.php`, `app/Http/Controllers/Api/CampaignScheduledController.php`, `app/Http/Requests/Api/CampaignScheduledRequest.php`, `app/Services/Api/CampaignScheduledService.php`, `app/Helpers/CampaignScheduledContactsHelper.php`, `app/Http/Middleware/SingleAuthentication.php`.

### Rota candidata
`POST /api/campaigns/schedule/execute`.

- `campaign`: ID da campanha (obrigatório, não confirmado para Natal 2026).
- `contacts`: lista textual de contato, normalmente `DDDxxxxxxxxx;Nome` por linha. Validador aceita telefone BR de 10–11 dígitos SEM DDI 55.
- `instance`: ID de `message_instances` no CRM (numérico; não confundir com Evolution Go UUID).
- `start_at`, `end_at` opcionais, formato `YYYY-MM-DDTHH:mm`.
- `use_leads_system` opcional, somente depois de verificar regra da campanha.
- Middleware `SingleAuthentication` compara `Authorization` a token interno configurado; não presumir `Bearer`.

O POST cria agendamento e **não é confirmação de entrega**. Não contém chave de idempotência evidente no contrato analisado. Registrar uma intenção no bot antes da chamada, e se houver timeout depois de enviá-la, marcar `uncertain`, não repetir cegamente.

`GET /api/campaigns/schedule/search` permite consulta por campanha ou telefone, mas resultados podem ser insuficientes para reconciliar ambiguidades em concorrência. Não alterar CRM para adicionar campos.

**NÃO CONFIRMADOS:** URL exata, ID campanha Natal 2026, ID da instância de saída, token, opt-in/consentimento e disponibilidade de serviço. Preflight sem POST real.


## Comprovações e ajuste de segurança posterior
`Natal2026ProposalManifest::VARIANTS` no GitHub confirma `415 = with_name` e `416 = without_name`; NÃO confirma que estão ativas ou provisionadas no DB do CRM. `CampaignScheduledResource` serializa `data.id` e `data.campaign`; aceitar agendamento só se resposta 2xx e ID de agendamento positivo/campanha correspondente.

`internal/crm/contract.go` constrói as duas variantes. `internal/crm/client.go` exige ID numérico explícito e positivo de saída do CRM (diferentemente da API legada, que aceita null). Isso evita usar por acidente a instância preferencial de número desconhecido. O cliente ainda NÃO está conectado ao receptor. Resultado HTTP incerto bloqueia reenvio automático e exige reconciliação.
