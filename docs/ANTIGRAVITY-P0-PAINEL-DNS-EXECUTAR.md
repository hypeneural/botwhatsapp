# Antigravity 3.8 Flash — execução objetiva P0 painel e DNS

**Uma missão, uma PR, um operador:** hypeneural/botwhatsapp PR #10, branch fix/ctwa-native-proposal-handoff-20261027. **Não modificar Evydência CRM, Evolution Go nem aplicações vizinhas. Não reiniciar VPS.**

## O que está PRONTO no Git e o que NÃO está comprovado
- Correção core.go: PushName mesmo quando WhatsApp envia @lid sem telefone; isso não permite inventar número.
- Dashboard events.go: associação somente por @lid idêntico + telefone E.164 explicitamente confirmado em outro evento; conflitos ficam pendentes; nomes por CRM preferidos quando correspondência única.
- Dashboard service.go: nomes CRM via cache privado (read-only) e fallback WhatsApp. LID incompleto continua pendente.
- Scripts: sync-ctwa-crm-names-readonly.py, audit-ctwa-panel-readonly.py, audit-ctwa-dns-readonly.sh e testes sintéticos.
- Template HTTPS público SOMENTE do painel: deploy/caddy/bot.etijucas.com.br.caddy.example.
- Foto do usuário sugere DNS A bot.etijucas.com.br -> 169.58.210.14, mas comprovar registro SALVO, IP WAN CORRETO, DNS, TLS e rota Caddy. Não presumir endereço correto.
- O container anterior foi informado como em HEAD 13c9f8...; commits desta tarefa não estarão nele até novo build/deploy feito no host autorizado.

## Fase 1 — checkout e testes (sem deploy)
No Avell com o repo em worktree correta:
```powershell
git status -sb
git fetch origin
git switch fix/ctwa-native-proposal-handoff-20261027
git merge --ff-only origin/fix/ctwa-native-proposal-handoff-20261027
# Go formatar apenas arquivos ALTERADOS desta tarefa e conferir diff:
gofmt -w internal/core/core.go internal/dashboard/events.go internal/dashboard/service.go internal/dashboard/identity_test.go internal/dashboard/ui.go internal/ingress/server.go internal/ingress/server_test.go
go test -count=1 ./internal/core ./internal/dashboard ./internal/ingress ./internal/outbox ./internal/evolution
go vet ./...
```
Executar também `bash scripts/validate-ctwa-p0-credentials.sh` no ambiente com Bash/Python disponível. Esperado: sem POST, sem dados reais em logs. Se falhar, corrigir somente causa comprovada e publicar em **mesma PR**. Conferir workflow CI do novo HEAD, SHA, diff.

## Fase 2 — identidade e nomes, somente leitura no CRM
No host onde o spool está montado, revisar:
```bash
python3 scripts/sync-ctwa-crm-names-readonly.py \
  --env /srv/platform/stacks/botwhatsapp/.env.production \
  --spool /var/lib/docker/volumes/botwhatsapp-stage_botwhatsapp_data/_data
```
Usa token já existente somente por GET /api/v1/extension/customer-context; requer ability extension:customer-read. Se o token não tiver essa ability, **SKIP**, não alargar escopo por conta própria. Se preview conferir caminho e contagens, executar `--apply` (até 40 GET por rodada), registrar só contagens e sem PII. Conferir que a cache 0600 no volume está legível pelo UID do container. NÃO apagar journal, claims, opt-outs ou contatos.

Diagnosticar por contagem agregada quantos eventos são @lid sem SenderAlt; confirmar versão real da Evolution, e que Info.Sender/SenderAlt vem ou não vem no payload. Não converter LID em telefone e nunca disparar tentativa comercial para descobrir número. Eventos antigos só podem ser corrigidos se houver evidência preservada de associação exata.

## Fase 3 — DNS e proxy, sem interferir em outras rotas
1. Validar IP externo REAL da VPS e comparar com o 169.58.210.14 do print; confirmar A/AAAA da zona autoritativa e DNS recursivo. Se divergente, NO-GO e apontar causa, sem trocar outros registros.
2. Inspecionar Caddyfile e Compose EFETIVOS, namespace de rede e upstream do bot. **Não substituir** Caddyfile existente pelo template.
3. Integrar host bot.etijucas.com.br usando template, com HTTPS automático e SOMENTE /dashboard /painel /api/dashboard/*. Bloquear /static, /webhook, spool e env no endereço público. Manter webhook da Evolution Go funcionando na rota PRIVADA atual, sem reconfigurá-lo.
4. Validar `caddy validate --config <CAMINHO_REAL> --adapter caddyfile`; fazer backup do Caddyfile e, só após gates, `caddy reload --config <CAMINHO_REAL> --adapter caddyfile` ou equivalente do container existente; NUNCA restart caddy/host sem necessidade.
5. Com DNS correto, executar `bash scripts/audit-ctwa-dns-readonly.sh <IP_WAN_CONFIRMADO>`; sem senha deve retornar 401 no painel e nas APIs; rotas /static/* 404 publicamente. O IP apresentado no print não deve ser passado cegamente como IP confirmado.

## Fase 4 — validar painel privado e aprovar rollout cirúrgico
Em host privado, script usa Basic Auth da env sem mostrar senha:
```bash
python3 scripts/audit-ctwa-panel-readonly.py --env /srv/platform/stacks/botwhatsapp/.env.production
```
No hostname público apenas APÓS checar DNS e TLS real:
```bash
python3 scripts/audit-ctwa-panel-readonly.py --env /srv/platform/stacks/botwhatsapp/.env.production --base-url https://bot.etijucas.com.br
```
Deve confirmar healthz/readyz no localhost, painel e API eventos/contatos com JSON legível, 401 anônimo, URL absoluta JS. **Nota:** o template Caddy isola healthz/readyz e o script HTTP externo exige esses caminhos: testar o domínio com chamadas curl sem autenticação aos caminhos expostos, ou ampliar o script para aceitar --panel-only quando em ambiente externo, sem expor healthz desnecessariamente. Não considerar 404 healthz no hostname como falha de privacidade; confirmar no localhost.

Validar UI no Chrome desktop e 390 px usando HTTPS final, sem CORS, sem rolagem horizontal e com nome/telefone representando evidências reais. CRM como fonte prevalente de nome; WhatsApp PushName deve ser identificado como auto declarado; LID sem prova deve ser "Número oculto pelo WhatsApp (LID)". Testar nomes em tabela de eventos e na aba de contatos. Não mostrar nomes ou telefones reais no relatório.

## Fase 5 — corte seguro opcional
Somente após CI, backup spool, image digest e Caddy gates: atualizar apenas serviço bot no Compose EFETIVO; preservar volume/worker; verificar /readyz, /dashboard autenticado, API eventos/contatos, Echo request. UM canário novo com consentimento, comprovando Evolution texto/áudio/PDF e CRM dispatch_uuid. **Sem reprocessar 13 erros HTTP 422**, sem alterar CRM ou outros containers. UNKNOWN não é sucesso nem motivo para retry cego.

## Saída, sem enrolação
- SHA/PR e testes PASS/FAIL; checksum do spool/backup intacto.
- Domínio DNS A e WAN da VPS batendo? HTTPS válido? Caddy validado/recarregado?
- Quantos telefones LID sem mapeamento e quantos nomes CRM conferidos? Só agregados.
- JSON painel real, navegador desktop/mobile, PDF Evolution, 1 canário e dispatch UUID (se realizado).
- GO ou NO-GO exato, com UMA pendência decisiva. Nunca revelar secrets/PII.

Documentação: docs/CTWA-DASHBOARD-IDENTIDADE-DNS-20261027.md; docs/CTWA-P0-EXECUCAO-AI-FIRST.md; docs/PAINEL-GESTOR-CTWA-20261027.md.
