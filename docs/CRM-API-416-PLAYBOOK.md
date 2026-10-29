# CRM Evidência — contrato verificado em main (21/10/2026)

> **LEGADO / SOMENTE REFERÊNCIA FORENSE:** em 26/10/2026 o CRM passou a rejeitar campanhas individuais Natal 415/416 via `POST /api/campaigns/schedule/execute` com HTTP 422. O hotfix vigente da PR #10 exige `POST /api/v1/extension/proposals/schedule` com Sanctum `extension:proposal-write` e recibo nativo. **NÃO executar nem configurar o endpoint abaixo para o bot atual.** Ler [CTWA-NATIVE-HANDOFF-20261027.md](CTWA-NATIVE-HANDOFF-20261027.md).


**ESCOPO**: consumidor HTTP em `botwhatsapp`. O repositório `hypeneural/evydencia-crm` é somente leitura; não instalar/alterar nada nele. Este documento é uma revisão estática de código, não prova de agendamento na produção.

## 1. Rota e autenticação

- Base autorizada: `https://evydencia.com` (configurável via `EVIDENCIA_CRM_BASE_URL`).
- Método: `POST /api/campaigns/schedule/execute`.
- Recurso de pesquisa: `GET /api/campaigns/schedule/search?campaign.id=416&contacts.phone=48999999999`.
- **Header obrigatório**: `Authorization: <TOKEN_INTERNO>` exatamente como valor de `config('services.internal.authorization_token')`. **NÃO** usar `Bearer`.
- `Content-Type: application/json`. O token é fornecido SOMENTE por ambiente privado. Não aparecer em Git, screenshot, relatório ou log.
- Rede: HTTPS entre Contabo e CRM. O cliente `internal/crm/client.go` recusa HTTP não-local, redirecionamentos, token vazio e envio sem a instância de saída explícita.

Fonte: `routes/api.php`, `app/Http/Middleware/SingleAuthentication.php`, `app/Http/Controllers/Api/CampaignScheduledController.php`.

## 2. Campanha SEM NOME é a padrão absoluta

O código `Natal2026ProposalManifest::VARIANTS` declara:

- `without_name` -> **416** (padrão fixo deste bot)
- `with_name` -> 415 (somente opt-in futuro, jamais por inferência de PushName)

`internal/crm/contract.go`: `Build(...)` e `BuildWithoutName(...)` SEMPRE criam `campaign:416` e `contacts:DDD+telefone`, mesmo que exista nome no WhatsApp. `BuildWithName` é separado e não usado nos workflows atuais. O nome não é enviado no fluxo 416.

**Exemplo EXCLUSIVAMENTE ilustrativo** (não executar; tanto telefone quanto instância são fictícios):

```http
POST /api/campaigns/schedule/execute HTTP/1.1
Host: evydencia.com
Authorization: <TOKEN_INTERNO_CONFIGURADO>
Content-Type: application/json

{"campaign":416,"contacts":"48999999999","instance":2,"use_leads_system":false,"type":"campaign"}
```

- `campaign`: obrigatório; ID cadastrado na tabela campaigns.
- `contacts`: obrigatório; telefone nacional de **10 ou 11 dígitos com DDD, sem 55**. `5548...` será rejeitado pelo `CampaignScheduledContactRule` / `CampaignScheduledContactsHelper`.
- `instance`: ID **numérico** de `message_instances.id` dentro do CRM; NÃO é número WhatsApp, UUID GO ou ID da sessão de Meta Ads. A rota aceita `null` e escolheria instância favorita. Nosso cliente RECUSA isso para garantir o número de saída.
- `use_leads_system`: false. Evita criar novo lead sem solicitação.
- `type`: `campaign` é a chave válida de `CampaignScheduledKind`; explicitado para não depender de default indefinido.
- `start_at` e `end_at` opcionais em `YYYY-MM-DDTHH:mm`. Não enviar = agendar a partir de agora segundo timezone do CRM.
- Outras propriedades da API existem, mas não são necessárias neste fluxo. Não enviar `customer`, `order`, `product` sem necessidade comprovada.
- O ID **416 no código NÃO confirma que a campanha está ativa/provisionada em produção**; usuário relata que está ativa, confirmar no CRM pelo meio permitido.

## 3. JSON de resposta de agendamento

O serviço usa `CampaignScheduledResource` e `AppServiceProvider::boot` executa `JsonResource::withoutWrapping()`. O exemplo Scribe `.scribe/examples/campaigns/execute-success.json` confirma objeto **na raiz**:

```json
{
  "id": 4721,
  "campaign": 416,
  "created_at": "2026-10-21T20:00:00.000000Z",
  "start_at": "2026-10-21 20:00:00",
  "finish_at": null,
  "type": {"id": 1, "name": "campaign", "description": "Campanha"},
  "use_lead_system": false,
  "contacts": [],
  "instance": {"id": 2, "name": "Instância configurada"},
  "status": {"id": 1, "name": "scheduled", "description": "Agendado"}
}
```
**Campos e valores usados neste exemplo são ilustrativos.** A presença de `id>0` com `campaign=416`, `instance.id` igual à configurada e `status.name` preenchido é o ACK verificável **de agendamento**, NÃO confirmação de envio Z-API, entrega ou leitura.

Por compatibilidade, `internal/crm/client.go` aceita também `{"data":{...}}`. A versão atual do CRM normalmente retorna objeto sem esse envelope.

Se o serviço interno falhar, seu `createScheduledCampaign()` pode retornar `null` e a API devolver corpo nulo/vazio. Resposta `2xx` sem ID verificável => resultado **uncertain**, não repetir POST.

## 4. Erros — sem duplicação comercial

| HTTP | Interpretação no cliente | Ação |
|---|---|---|
| 2xx com ID positivo, campanha e instância corretos | Agendamento confirmado | Salvar ID e status |
| 400, 401, 403, 404 ou 422 | Rejeição da requisição (auth/validação/rota) | Não repetir automaticamente; corrigir credencial/payload/config |
| 408, 409, 429, 5xx, timeout ou desconexão | Resultado desconhecido | `uncertain`; jamais repostar automaticamente |
| 2xx malformado, nulo ou ID incompatível | Resultado desconhecido | `uncertain` |

Os erros expostos ao bot trazem apenas códigos/tipos, sem corpo de erro que possa conter token ou dados de contato.

## 5. GET de consulta e reconciliação

`GET /api/campaigns/schedule/search?campaign.id=416&contacts.phone=48999999999` usa o mesmo token interno. Implementação atual filtra `campaign_id` e busca correspondência exata `campaign_scheduled_contacts.whatsapp`.

O recurso de coleção também é tipicamente **array raiz** por causa do `withoutWrapping()`. `internal/crm/client.go` suporta array raiz e `{"data":[...]}`, com limite máximo e verificação local de campanha/telefone. O retorno inclui agendamentos com `id`, `campaign`, `status.name`, `instance.id`, `contacts[].whatsapp`.

**Limites de prova:** um array vazio **não prova** que o POST anterior falhou; pode haver concorrência, latência ou diferenças de visibilidade. O bot não deve reabrir `uncertain` para nova tentativa sem reconciliação explícita pelo operador. A API de busca não recebe idempotency key e não verifica univocamente a tentativa específica.

## 6. Distinção de três estados

1. `crm_schedule_confirmed` = a API criou um agendamento `id`.
2. `first_message_provider_accepted` = sistema do CRM registrou aceitação no provedor WhatsApp, **não é informação retornada pelo POST**.
3. `delivered/read` = recibos de entrega/leitura, **não retornados por esta API**.

O novo bot só deve afirmar ao cliente que vai encaminhar a proposta, sem garantir que já foi entregue.

## 7. Evidência e testes

Arquivos CRM de referência somente leitura:
- `app/Providers/AppServiceProvider.php` (`JsonResource::withoutWrapping`)
- `routes/api.php`
- `app/Http/Middleware/SingleAuthentication.php`
- `app/Http/Requests/Api/CampaignScheduledRequest.php`
- `app/Http/Requests/Api/CampaignScheduledSearchRequest.php`
- `app/Services/Api/CampaignScheduledService.php`
- `app/Services/CampaignScheduledService.php`
- `app/Services/CampaignScheduledContactService.php`
- `app/Http/Resources/CampaignScheduledResource.php`
- `app/Services/Proposals/Natal2026ProposalManifest.php`
- `.scribe/examples/campaigns/execute-success.json`
- `.scribe/examples/campaigns/search-success.json`

Testes mockados do nosso repositório: `go test ./internal/crm`. Nenhum POST de produção foi feito nesta auditoria.
