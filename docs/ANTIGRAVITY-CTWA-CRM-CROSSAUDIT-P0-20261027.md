# Antigravity 3.8 Flash — único plano executável CTWA ↔ CRM Natal V4

**Uma PR:** hypeneural/botwhatsapp #10, branch fix/ctwa-native-proposal-handoff-20261027. A implantação vigente NÃO é prova de que o HEAD mais novo da PR já está no container. **Não criar outras branches/PRs. Não modificar evydencia-crm, Evolution, workers vizinhos ou os registros do spool.**

## O erro forense identificado e já corrigido no Git
O dossiê entregue inicialmente descreveu rotas **inexistentes para esta integração**: `POST /api/campaigns/proposals` com `campaign_id=416` e `GET /api/customers?phone=`. Isso era alucinação documental, não evidência do código. Ler **docs/AUDITORIA-VALIDACAO-CHATGPT-20261027.md**, que foi corrigido com os arquivos reais do Laravel e os limites de banco.

**Contrato único e imutável desta release:**
- CRM POST `/api/v1/extension/proposals/schedule`; corpo `request_id` UUIDv5 por task, `phone` nacional, `name:null`, `resend_mode:normal`; header `X-Idempotency-Key` idêntico ao request_id.
- 201 agendamento novo; 200 replay seguro; confirmar `data.status=scheduled`, `data.dispatch_uuid` UUID. Isso **não comprova entrega** WhatsApp.
- Rejeições 400/401/403/404/422; timeout, 408/409/429/5xx → INDETERMINADO até reconciliação, sem POST cego.
- GET nomes `/api/v1/extension/customer-context?phone=DDDnumero` exige `extension:customer-read`; não funciona só com `extension:proposal-write`.
- O CRM escolhe 415 ou 416 pelo fluxo e pelo nome encontrado no CRM; bot nunca escolhe campanha/instância.

## Fase A: CI e testes (Avell, sem VPS)
1. Confirmar repo, worktree, branch, `git status -sb`, `git fetch`, `git merge --ff-only`, SHA. Se houver alterações locais, parar antes de sobrescrever.
2. Ler `AGENTS.md` e o dossiê corrigido. Validar regex, parser JSON e status UI com testes reais.
3. Rodar APENAS os gates úteis:
```powershell
gofmt -w internal/core/core.go internal/dashboard/events.go internal/dashboard/service.go internal/dashboard/ui.go internal/dashboard/task_timing_test.go internal/dashboard/identity_test.go internal/ingress/server.go internal/ingress/server_test.go
go test -count=1 ./internal/core ./internal/crm ./internal/dashboard ./internal/outbox ./internal/ingress
go vet ./...
python scripts/validate-ctwa-crm-contract.py
python scripts/audit-ctwa-legacy-422-readonly.test.py -v
python scripts/sync-ctwa-crm-names-readonly.test.py -v
```
`gofmt` só nos arquivos Go, naturalmente; não executá-lo em scripts/HTML. Caso exista checkout read-only do CRM, chamar `python scripts/validate-ctwa-crm-contract.py --crm-root <CHECKOUT_CRM>` — não clonar/editar CRM só por este gate. `bash scripts/validate-ctwa-p0-credentials.sh` complementa testes sem segredos reais. Se teste falhar, corrigir causa e dar um único ciclo de CI; não loops intermináveis.
4. Workflow "Go security and offline contract checks" e "Container builds and staging config": aguardar resultado do HEAD final. Nada de deploy antes de PASS.

## Fase B: confirmar identidade e front-end no hostname real
1. Fotos do gestor: 8 mensagens com sinal CTWA e 6 recibos CRM na janela consultada; não representam 8 clientes únicos nem 6 entregas confirmadas. Há pelo menos um @lid sem telefone resolvido.
2. Em host VPS bot, usar `scripts/audit-ctwa-identities-readonly.py --spool <SPOOL_REAL>` para contagens agregadas. Não imprimir LIDs, nomes ou números.
3. Verificar Evolution Go versão real e `Info.Chat`, `Info.Sender`, `Info.SenderAlt`, `Info.PushName`; LID sem E.164 seguro continua pendente. Associação de LID só por alias e telefone explicitamente preservados, nunca por sufixo.
4. Antes de `--apply` no enriquecimento de nomes, executar preview `scripts/sync-ctwa-crm-names-readonly.py --env <ENV_REAL> --spool <SPOOL_REAL>`. Se token de propostas não tiver customer-read, usar **somente** uma credencial da extensão JÁ EXISTENTE com essa ability armazenada em arquivo privado 0600: `--read-token-file <ARQUIVO_0600>`. Não criar usuário/ability, não imprimir token, não usar segredo do painel como token CRM. Fazer no máximo uma rodada de 40 GET com pausa/reconciliação; contar correspondências únicas e conferir UID/ACL do cache. Se não houver credencial, SKIP enriquecimento, bot comercial continua.
5. Verificar painel: `python3 scripts/audit-ctwa-panel-readonly.py --env <ENV_REAL>` (local) e, após TLS/DNS seguros, `--base-url https://bot.etijucas.com.br --panel-only`. Validar navegador desktop/390px, API eventos, API contatos, nomes com fonte, telefone e filtros. **CRM agendou** não é "WhatsApp entregue".
6. O status `em_processamento` foi corrigido para usar `AttemptStartedAt`, não horário antigo do evento. Confirmar o teste `TestAttemptStartedUsesTaskTimestampNotOldInbound` para evitar falsas incertezas.

## Fase C: 13 casos 422 antigos sem reenvio
1. Rodar `python3 scripts/audit-ctwa-legacy-422-readonly.py --spool <SPOOL_REAL>` para contagens, sem nenhuma PII e sem POST. A rotina inclui HTTP 422 genérico mesmo quando o log antigo perdeu a assinatura do erro.
2. Em fonte CRM autorizada, usar consulta SOMENTE READ de proposta/pedidos, status Natal 2026, comunicação e supressões; verificar por telefone/identidade de forma privada sem anexar CSV na PR.
3. Classificar já-agendado, convertido, suprimido, ambíguo, sem evidência de agendamento e desconhecido; não presumir que os 13 precisem de proposta. Nunca executar 13 POSTs, Tinker insert ou mutações no banco sem decisão comercial específica.

## Fase D: somente depois do GO
1. Conferir se DNS bot.etijucas.com.br realmente resolve para WAN da VPS, TLS, Caddyfile real e upstream. Publicar somente rotas autenticadas do painel, sem alterar /webhook interno nem /static privado.
2. Backup spool e imagem; inspecionar projeto Compose real, um worker, SHA/digest, flags runtime. Deploy somente bot, sem `down -v` ou reinício do host.
3. Após deploy, `/readyz`, painel HTTPS, JSON, cache nomes e um único canário novo consentido: Evolution aceitou etapas, CRM devolveu `dispatch_uuid` e despacho confirmado no banco; entrega eventual precisa de confirmação própria. Não reenviar os 13 históricos.
4. Se CI falhar, API CRM retornar 401/403/422, GET nomes 403 ou status incerto, registrar a causa, sem contornar permissão. Rollback somente bot com snapshot de spool comprovado.

## Entrega para gestor em até 7 linhas
PR/HEAD e CI; mudanças Go; endereço HTTPS/JSON navegador; contagens @lid e nomes CRM; fluxo Meta→Evolution→CRM (quantidade e evidência); reconciliação 422 por classe (sem PII); **GO/NO-GO** com bloqueio exato. Proibido colar segredo Sanctum, telefones, nomes, hashes privados, URLs com segredo ou payload de cliente real.

Docs oficiais: https://laravel.com/docs/10.x/sanctum e https://laravel.com/docs/10.x/migrations. Use a branch atual como fonte de verdade, não o dossiê antigo nem suposições do modelo.
