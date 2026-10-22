# Handoff Antigravity Flash Lite — Fechamento cirúrgico das PRs

**Repositório:** `hypeneural/botwhatsapp`
**Objetivo:** integrar os PRs sem perder o trabalho do canário e manter os serviços existentes da Contabo intocados.

## Estado de origem
- PR #1 — CRM 416 sem nome (parte da consolidação).
- PR #2 — audit/multi-instance (parte da consolidação).
- PR #3 — worker auditado do canário (parte da consolidação).
- PR #4 — `feat/consolidate-multiinstance-outbox` (consolidação de código anterior).
- PR #5 — `fix/ingress-outbox-production-guards-20261022` (hotfix/auditoria em cima da PR #4).
- A `main` não foi alterada por esta auditoria. **Não fazer merge de PR #1/#2/#3 separado caso os patches já estejam em #4**, pois geraria retrabalho/duplicações.

## Procedimento curto e obrigatório
1. Primeiro `git status -sb`. Se há trabalho local, preserve-o; não usar `reset --hard`, `git clean -fd` ou force-push.
2. Leia `AGENTS.md`, `docs/FORENSIC-PR4-20261022.md` e a skill `outbox-production-audit`. Verifique que PR #4 contém os contratos/worker/registry das PRs anteriores.
3. Revise a PR #5 contra #4. Principais mudanças:
   - webhook somente ACK 202 depois de `CommitEventTasks` e spool, sem goroutine para persistir;
   - PARAR deve persistir supressão;
   - lote de tarefas com fsync, marcador de evento por último e IDs determinísticos;
   - sender Evolution Go diferente por UUID; mesmo token nas duas instâncias é erro de boot;
   - sem bytes MOCK, sem triggers amplos, sem bind aberto por padrão;
   - humano/opt-out/cooldown revalidados antes do envio;
   - notificação do worker imediata, recuperação 60s, pruning apenas de tarefas concluídas a cada 6h;
   - configuração `.env.example` usando campos REAIS de runtime.
4. Execute `go test -count=1 ./...` uma única vez após conciliar o diff e confira Actions do HEAD. Se falhar, corrigir o pacote específico e repetir suite uma vez no final.
5. Primeiro finalize/mescle a PR #4 na `main` apenas após revisão. Em seguida altere base da PR #5 para `main`, confirme testes e mescle #5. Feche PRs #1–#3 **somente se** conteúdo e commits estiverem preservados na #4. Não feche automaticamente uma PR não reconciliada.
6. Atualize o README e o registro de release com SHA de `main`, estados das PRs, flags que permanecem OFF e resultado dos testes.
7. NÃO ativar `BOT_OUTBOUND_ENABLED` nem publicar webhooks antes de resolver o **relay/fan-out durável para o status-engine**, pareamento da saudação, mídia real e credenciais privadas apropriadas.
8. Não repetir POST CRM 416 nem reenviar WhatsApp de canário já executado; `GET /api/campaigns/schedule/search` anterior sofreu timeout e segue inconclusivo.
9. Responder com mudanças confirmadas, testes, conflitos resolvidos, bloqueios e **somente** os comandos necessários ao próximo canário autorizado (sem executá-lo).

## Restrições inegociáveis
- `hypeneural/evydencia-crm` é SOMENTE LEITURA. Instância CRM de saída relatada: ID numérico 1; não confundir com UUID GO.
- `anderson` precisa continuar enviando webhook para `whatsapp-status-status-engine-1`. NÃO reconfigurar durante este trabalho.
- Não reiniciar/recriar Evolution Go, Postgres, Caddy, SmartDialer, Asterisk, status, portal, gateway de SMS nem música/jogos/vídeo.
- Não executar deploy, envio comercial ou exposição de portas públicas; produção segue BLOQUEADA até fan-out e homologação separados.
- Não commitar segredos, `.env`, áudio/imagem privados, logs brutos ou `data/spool`.
