# Auditoria cruzada canônica — BotWhatsApp CTWA × Evydência CRM Natal V4

**Verificado em:** 27/10/2026. **Repositórios:** `hypeneural/botwhatsapp` PR #10 e `hypeneural/evydencia-crm` (main). **Tipo:** auditoria estática do código e do contrato, não consulta ao banco produtivo, não comprovação de entrega WhatsApp.

> **CORREÇÃO P0:** a versão anterior deste dossiê afirmava que o bot chamava `POST /api/campaigns/proposals` com `crm_phone, instance_id, contact_name, campaign_id=416, source` e fazia GET `/api/customers?phone=`. **Essas informações estavam ERRADAS**. Nenhuma dessas duas rotas é o contrato da extensão nativa. A partir deste documento, usar apenas os contratos abaixo. A campanha não deve ser fixada pelo bot e o nome do WhatsApp não deve ser afirmado como nome CRM.

## 1. Arquitetura e endpoints REAIS

```text
Meta Ads CTWA -> Evolution Go / webhook
 -> Bot internal/core (gatilho, telefone E.164 ou LID não resolvido)
 -> spool/journal durável e task outbox por evento
 -> Evolution Go: reação -> boas-vindas -> áudio PTT -> PDF -> texto comercial
 -> CRM Laravel: POST /api/v1/extension/proposals/schedule (Sanctum)
 -> retorno: data.status=scheduled; data.dispatch_uuid=<UUID>
 -> painel: agendamento registrado no CRM, NÃO entrega de WhatsApp
```

Arquivos de verdade:
- Bot `internal/crm/native.go::NativeClient.ScheduleNative`, `internal/outbox/worker.go` e `internal/dashboard/events.go`.
- CRM `app/Providers/RouteServiceProvider.php`, `routes/extension-api.php`, `app/Http/Requests/Api/Extension/ProposalScheduleRequest.php`, `app/Http/Controllers/Api/Extension/ProposalScheduleController.php`, `app/Services/Proposals/ScheduleProposal.php`.

**POST** `https://evydencia.com/api/v1/extension/proposals/schedule`

Headers: `Authorization: Bearer <SANCTUM>`, `Accept: application/json`, `Content-Type: application/json`, `X-Idempotency-Key: <mesmo request_id>`.

Payload **real** que o Go monta:
```json
{
  "request_id": "11111111-2222-5333-8444-555555555555",
  "phone": "48999990001",
  "name": null,
  "resend_mode": "normal"
}
```
UUID acima é um exemplo de formato, **não reutilizar como idempotency key**. O UUIDv5 real é calculado de forma determinística a partir do ID durável da task `schedule_natal_crm` pela função `crm.NativeRequestID(task.ID)`.

`ProposalScheduleRequest::rules`: `request_id` obrigatório UUID; `phone` obrigatório string até 30 e celular brasileiro válido; `name` obrigatório estar presente mas pode ser null; `resend_mode` opcional e restrito a `normal`, `send_again` e `abort_and_resend`. `X-Idempotency-Key`, quando fornecido, deve **coincidir exatamente** com `request_id`.

O CRM escolhe `proposal_flow` pela configuração `proposals.natal_flow_slug`, `with_name_campaign_id` e `without_name_campaign_id` no banco, resolve instância e aplica regras de elegibilidade. **Bot NÃO envia ID da campanha nem da instância no contrato nativo**. `DecideProposalRecipient` usa nome do próprio CRM se telefone identificar uma única pessoa, mesmo com `name=null`; do contrário, seleciona variante sem nome. Os dados `source`, `utm_source`, `ad_id` e `campaign_id` não são campos validados por este FormRequest. **Não acrescentar campos para medir origem sem uma mudança coordenada do contrato e LGPD**; a atribuição disponível hoje fica no journal privado do bot.

`ProposalScheduleController` retorna `data.status=scheduled`, `data.dispatch_uuid` e `data.replay` no envelope JSON, **HTTP 201** na primeira gravação ou **HTTP 200** no replay. HTTP 201 comprova agenda registrada; **não comprova que Z-API/WhatsApp entregou PDF ou outra etapa ao cliente**.

## 2. Autorização e taxas

- `routes/extension-api.php`: middleware raiz `extension.correlation → extension.nocache → auth:sanctum` via RouteServiceProvider; rota `proposals/schedule` exige `extension.enabled`, ability `extension:proposal-write` e `throttle:extension-write`.
- `ProposalScheduleRequest` usa Gate `ProposalFlowPolicy::send`: `User.can_send_proposals=true`. A ability sozinha **não é suficiente**.
- O token de escrita dedicado pode conter somente `extension:proposal-write`. O token completo da extensão costuma possuir abilities adicionais, mas compartilhá-lo amplia superfície de risco. Um token de escrita mínima **NÃO tem permissão para ler nomes**.
- API para nomes: **GET** `https://evydencia.com/api/v1/extension/customer-context?phone=48999990001`, com ability `extension:customer-read` e `throttle:extension-read`. Status único `data.lookup.status=matched` e `data.customer.display_name`; ambiguous/not_found nunca fornecem nome atribuído.
- `app/Providers/RouteServiceProvider.php`: `extension-read` limite **180/minuto e 2000/hora**, `extension-write` **20/minuto e 200/hora**, por identidade de usuário + token (em vez de IP quando token existe). Não confundir com `api` 60/min. Os checks reais devem respeitar 429 e `Retry-After`.
- `config/sanctum.php`: `expiration => null` como padrão repositório (confirmar override produtivo). Emissor do bot foi ajustado para `expires_at` individual de **90 dias**. Token expirado/revogado gera 401; não há renovação automática pelo bot.
- `app/Console/Kernel.php`: `sanctum:prune-expired --hours=24` diariamente às **03:00**. Cron de limpeza não é renovação do token.
- Rotação: emitir nova credencial no host Laravel autorizado com ability mínima, conferir `can_send_proposals`, guardar na VPS de forma privada, testar, trocar atomicamente sem expor valores e revogar anterior após verificação. Não enviar chave ao chat.

## 3. Banco, concorrência, duplicidade e risco de timeout

- `database/migrations/2026_07_13_000001_create_proposal_flow_tables.php`: `proposal_dispatches.request_id` **UNIQUE**; `proposal_recipient_guards (proposal_flow_id, phone_lookup_hash)` **UNIQUE**; índices de apoio por fluxo/telefone/data, despacho e contato agendado.
- `ScheduleProposal::schedule` aplica transação DB (até 3 tentativas de deadlock), `lockForUpdate()` em `proposal_flows` e em `proposal_recipient_guards`, e primeiro consulta `request_id`; o banco serializa a janela de duplicação no mesmo fluxo/telefone. **Nunca depende apenas de cache**.
- Mesmo `request_id`, mesmas credenciais/flow/fingerprint → retorno do mesmo `dispatch_uuid`, `replay=true`, HTTP 200. Mesmo UUID com payload diferente → 422 `idempotency_payload_mismatch`.
- Request diferente para mesmo telefone em `normal` dentro de `proposal_flows.duplicate_window_minutes` (default migração 15 minutos) → 409 `duplicate_recent_dispatch`, sujeito à configuração real do fluxo.
- `AssertProposalCommercialEligibility` bloqueia cadastros ambíguos (409) e clientes com Natal 2026 adquirido (403), faz fail-closed 503 se não puder provar elegibilidade; depois a transação repete o gate.
- O **bot** envia `name=null` por contrato, idempotency baseada em ID da task, registra `ErrUnconfirmed` para timeout, 408, 409, 429, 5xx ou resposta sem recibo; esses casos exigem reconciliação, **não novo POST cego**. `ErrRejected` 400/401/403/404/422 é reprovação definitiva para a task atual e precisa de diagnóstico antes de qualquer nova tentativa.
- `tests/Feature/Extension/ProposalConcurrencyTest.php` exercita unicidade, replay e bloqueio por guard em MySQL, mas a auditoria estática não substitui prova do banco da VPS em produção.

## 4. Nome/identificação do lead Meta Ads

- `internal/core/core.go`: só gera `phone_e164` a partir de JID `55DDD...@s.whatsapp.net` explícito; evento `@lid` sem `Sender`/`SenderAlt` confiável permanece bloqueado, nunca enviar ao CRM telefone deduzido dos números do LID. `PushName` é dado declarado do WhatsApp, **não identidade verificada**.
- `internal/dashboard/events.go` distingue nome CRM, WhatsApp, LID sem telefone e ausência de tarefa. Trilha de APIs mostra eventos de outbox e aceita que API da Evolution responda sem garantir entrega real.
- `scripts/sync-ctwa-crm-names-readonly.py` usa **GET customer-context** e requer explicitamente `extension:customer-read`. O token somente `proposal-write` não funciona nessa rota. Cache local privado TTL 48h, com nomes apenas para `lookup.status=matched`; cuidado com permissão UID do volume. Consultas de nome nunca são gatilho de proposta e nunca fazem POST.
- Meta Ads contexto no webhook ou texto igual ao gatilho é evidência de atribuição de *entrada*; não representa necessariamente compra de mídia identificada por ad_id/UTM. Sem `ad_id` canônico armazenado, o painel não deve alegar rastreabilidade por criativo individual.
- O print do usuário exibe leads com CRM agendado e um LID sem número: isso é compatível com fail-closed do core, não defeito que se corrige atribuindo número falso.

## 5. Os 13 HTTP 422 históricos — RECONCILIAR, NÃO REENVIAR EM LOTE

Os 13 contatos relatados pertencem à falha da API legada das campanhas 415/416 (`natal_proposal_native_entry_required`). Não foram auditados individualmente contra banco de produção nesta investigação. Portanto **não há autorização nem base factual para tratar todos como sem proposta**.

**Procedimento aceito (somente leitura primeiro):**
1. Extrair eventos históricos e tasks 422 do spool, sem expor telefones/nomes; identificar hash de evento, horário, telefone normalizado em local privado, ID de task, estado, ID de CRM se existente.
2. No CRM verificar por telefone normalizado (e variante legada), compras 2026/Gold Rule, bloqueios e `proposal_dispatches`, `campaign_scheduled_contacts` e `communication_events`. Não presumir que o 422 de uma task impede agendamento por outro canal.
3. Classificar cada caso como `ja_agendado`, `convertido_2026`, `supresso`, `ambiguidade`, `sem_agendamento_confirmado`, `indeterminado`. Não divulgar PIIs na PR/chat.
4. Para `sem_agendamento_confirmado`, usar **somente a entrada nativa oficial** com novo `request_id` controlado e humano autorizando caso a caso, depois de confirmar consentimento e interesse contemporâneo. Priorizar a interface oficial do CRM se não existir uma ferramenta de reconciliação autorizada.
5. **NÃO** inserir registros via Tinker/SQL (isso contornaria validação, dedupe e trilha). **NÃO** aplicar automaticamente um loop de 13 POSTs; um timeout exige conciliação antes de nova tentativa.
6. Relatório agregado por classe; operador decide se contatar ou não. Nenhum replay executado neste trabalho.

## 6. Segurança de produção e testes

- O hostname gerencial `https://bot.etijucas.com.br/dashboard` exige HTTP Basic e proxy HTTPS; scripts na PR verificam DNS, TLS, APIs, Caddy e rotas públicas. Sem evidência de navegador real/CI no HEAD atual, o print isolado não comprova integridade do fluxo completo.
- Evitar alteração do CRM nesta fase: adaptar somente bot, documentação e validações, mantendo o software de produção separado.
- Teste offline: `go test -count=1 ./internal/crm ./internal/dashboard ./internal/core ./internal/outbox`; `go vet ./...`; `python3 scripts/validate-ctwa-crm-contract.py` e `bash scripts/validate-ctwa-p0-credentials.sh`.
- Referências oficiais: https://laravel.com/docs/10.x/sanctum ; https://laravel.com/docs/10.x/migrations ; https://laravel.com/docs/10.x/queues .

**Regra AI-first:** se um modelo sugere `POST /api/campaigns/proposals` ou `GET /api/customers?phone=` para esta integração, rejeitar imediatamente como desvio de contrato, conferir o source atual e NÃO emitir chamadas reais. O bot não define `campaign_id=416` na API nativa.
