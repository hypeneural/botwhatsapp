# ANTIGRAVITY FLASH LITE — FINALIZAR AUDITORIA P0 PÓS-CUTOVER (SEM DISPAROS AUTOMÁTICOS)

**Repositório:** `hypeneural/botwhatsapp`, PR #10 Draft, branch `fix/ctwa-native-proposal-handoff-20261027`.
**Dependência:** `hypeneural/smartdialer-platform` PR #41 (Status Engine); CRM, Z-API, VoIP/Asterisk, PrintServer, Caddy e Evolution Go NÃO devem ser alterados como efeito colateral.
**Objetivo:** confirmar a restauração reportada do webhook e o funcionamento real do bot/relay, impedir recidiva, atualizar recibo e preparar canário consentido com rollback. Não abrir outra PR. Os códigos e testes já foram preparados pelo ChatGPT.

## 1. LER PRIMEIRO

`AGENTS.md` → `docs/START-HERE.md` → `docs/CTWA_POST_CUTOVER_GATES_20261031.md` → `docs/RECIBO_ANTIGRAVITY_CTWA_NUMBER_SWITCH_20261031.md`. O recibo é **histórico reportado**, não prova automática da imagem atual.

Leia também o contrato do Evolution Go 0.7.2/fork implantado (digest, Swagger/OpenAPI local e source tag), sem inferir a partir das páginas `main`; a documentação oficial de instâncias descreve `POST /instance/connect` autenticado pela chave da instância, mas **NÃO EXECUTE ESSE POST** nesta rodada.

## 2. SINCRONIZAR WORKTREE SEM PERDER TRABALHO

```powershell
git fetch --prune
git status -sb
git rev-parse HEAD
git log -1 --format="%H %s"
```

Se worktree limpa e atrás da branch, realizar somente fast-forward; se houver alterações, preservar e documentar. Nunca reset, force push, `git clean` ou merge da PR #10. Todas as correções pertencem à mesma PR.

## 3. CÓDIGO CORRIGIDO: NÃO REFAZER

- `internal/core/core.go`: JID `:device@s.whatsapp.net` é tratado como a parte telefônica SEM alterar nono dígito. Outro formato ou `@lid` permanece bloqueado.
- `cmd/botwhatsapp/main.go`: alias opcional `BOT_ADS_PAIRED_JID` é explicitamente verificado em inventário privado. Serve SOMENTE para detectar própria linha; não troca `BOT_ADS_PHONE` ou telefone de cliente.
- `internal/instances/registry.go`: alias nunca pode sobrepor identidade da role Greeting.
- `scripts/preflight_ctwa_account_switch.py`: diferencia identidade comercial/pareada e retorna exit 0 somente para `PASS_BINDINGS_ONLY`; não autoriza deploy/cutover.
- `docs/AUDITORIA_VALIDACAO_CHATGPT_P0_20261031.md` e `docs/RECIBO_ANTIGRAVITY_CTWA_NUMBER_SWITCH_20261031.md`: segredos de webhook redigidos, fan-out documentado como **sequencial**.
- `scripts/check_ctwa_public_docs.py`: bloqueia vazamento de caminhos bearer no dossiê. **Os segredos antigos seguem comprometidos no histórico do Git/chat e DEVEM ser rotacionados.**

## 4. TESTES LOCAIS EXATOS (UMA RODADA; SEM REDE DE PRODUÇÃO)

```powershell
gofmt -w internal/core/core.go internal/core/routing_safety_test.go internal/instances/registry.go internal/instances/registry_test.go cmd/botwhatsapp/main.go
git diff --check
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./internal/relay ./internal/ingress ./internal/outbox
python -m unittest scripts/test_preflight_ctwa_account_switch.py -v
python -m unittest scripts/test_relay_forensic_readonly.py -v
python -m unittest scripts/test_check_ctwa_public_docs.py -v
python scripts/check_ctwa_public_docs.py
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1
```

OBS: o gate `0600` do preflight exige POSIX; executar provas privadas na VPS/WSL. No Windows, unittest que depende de permissões POSIX pode ficar SKIP legítimo. O GO é testado no Windows, mas o contrato do kernel Linux precisa de CI Linux.

Corrigir apenas falha determinística observada, executar a suíte afetada e publicar o commit na mesma PR #10. Conferir CI **no último SHA**, não afirmar PASS de runs anteriores.

## 5. READ-ONLY NA VPS — AUDITORIA DO ESTADO ATUAL

Nunca copiar segredo, JID completo, contatos ou payload no Git/chat. Criar apenas recibos sanitizados.

- `docker inspect` do bot, relay e status-engine: SHA digest, `created_at`, reinícios, mounts, redes, health, `RELAY_MAX_EVENTS` e `RELAY_MAX_SPOOL_BYTES` efetivos, volume e usuário. Não executar `docker builder prune` novamente; conferir free disk. Verificar que 100.000 está **efetivamente** aplicado e não só num template.
- `GET /instance/all` no Evolution Go local: confirmar connected, nome, UUID, JID pareado e webhook para o relay, sem URL plena; hashes/booleanos, distinção entre E.164 comercial e JID de 12 dígitos com `:device`; global + por instância não podem duplicar.
- Validar public.instances via inventário PG read-only; identificar somente o fingerprint atual do webhook e se watchdog/Status Engine PR #41 poderia reescrever; checar versão da imagem Status Engine e `SMARTDIALER_WEBHOOK_URL` privado. Não fazer SQL UPDATE.
- Rodar `scripts/audit_relay_spool_readonly.py` com limites reais, `scripts/preflight_evolution_relay_contract_readonly.py` com amostra privada do próprio fork e `scripts/preflight_ctwa_account_switch.py` com `BOT_ADS_PAIRED_JID` se comprovado. **Nunca alterar dígitos do cadastro dos clientes**.
- Conferir contagens agregadas (últimas 1h/24h) de eventos Evolution, ingress 202, `StatusDone`, `BotDone`, backlog, erros/retentativas e tarefas CRM por estado. Se `StatusDone` sem `BotDone`, atendimento **NÃO** restaurado apesar do healthz.
- Verificar proteção `IsFromMe`, gatilho exato, cooldown de Ads, pause humana, opt-out e ausência de `SendAdClosingText` no novo plano. Não fazer replay dos dias 29–31/10.

## 6. SEGREDOS + CANÁRIO + ROLLBACK

**NO-GO para prosseguir automaticamente com chamadas reais enquanto os segredos publicados não forem rotacionados**, ou enquanto as métricas de ACKs por destino não estiverem verificadas. Não imprimir nem gerar secrets no chat.

Preparar em arquivos privados, com backup, a rotação COORDENADA de:
- segredo do ingress do relay + URL persistida da Evolution;
- segredo Ads no bot + caminho `RELAY_BOT_URL`.

Manter os destinos Status Engine e Bot; não apagar spool ou tombstones. Documentar passo de rotação e rollback antes de executar qualquer alteração. A rotação real precisa de autorização operacional/canário controlado separado; nunca usar token publicado no histórico como valor atual.

Após gates e autorização, testar um único contato próprio consentido em três casos: orgânico sem resposta; eco `IsFromMe` sem proposta; frase exata acionando UM fluxo e despacho CRM V4. Comparar recibos dos dois sinks, ausência de duplicidade e estabilidade após reinício. Não considerar healthz/202 suficientes.

## 7. ENTREGA NA MESMA PR

Criar/atualizar `docs/RECIBO_ANTIGRAVITY_CTWA_POST_CUTOVER_20261031.md` com:
- SHA do HEAD, CI Go/Go race/Python, diff e estado da PR #10;
- digest efetivo dos três contêineres e data de deploy;
- estado/assinaturas/identidade do provider (somente mascarado, sem URLs);
- estado dos 2 sinks e atraso do relay, limite de 100k e bytes atuais;
- diferença 12 vs 13 e prova de alias JID pareado;
- situação da rotação de segredos (PENDING/ROTATED, sem valores);
- resultado do canário (NOT_RUN/PASS/FAIL), plano de rollback;
- GO_CODE / NO_GO_RUNTIME / GO_CANARY, com evidência observável.

**Sem merge, sem deploy automático, sem reenvio em massa, sem acesso ao CRM para escrever.** Se alguma prova faltar, preencher `NOT_VERIFIED` em vez de inventar. Não reabrir tópicos antigos do CRM 416 ou instruções obsoletas da Evolution Go.
