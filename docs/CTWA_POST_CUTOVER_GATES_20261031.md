# P0 — Auditoria independente pós-cutover Evolution Go × Relay × BotWhatsApp

**Atualização:** 31/10/2026. **PR #10:** `fix/ctwa-native-proposal-handoff-20261027`.
**Natureza da evidência:** GitHub/código revisado diretamente; fatos operacionais da Contabo **relatados pelo Antigravity** no commit `577ee460`, ainda sem consulta direta à VPS nesta rodada.

## Conclusão operacional

**GO para validação local e preflight read-only. NÃO é GO de canário ou prova de atendimento end-to-end.** O Antigravity reporta contêineres atualizados e URL da instância Evolution Go alterada para o relay. A rota do webhook foi mudada, mas o teste consentido do fluxo real continua pendente. Nenhum `HTTP 200 /healthz` prova que uma mensagem foi encaminhada ao bot, que duas entregas foram aceitas ou que o CRM gerou proposta.

### Riscos P0 e correções no Git

1. **Segredo exposto no Git/chat:** o recibo e dossiê publicados continham caminhos de bearer-webhooks completos (ingress relay e bot). Eles foram **substituídos por marcadores redigidos na versão atual**, mas a exposição continua no histórico de commits e no chat. **Rotacionar os dois segredos em arquivos privados na VPS** após preparar rollback/planejar a janela; não publicar URL, token, segredo, hash de APIKEY nem comandos completos em issue/log. Conferir `RELAY_INGRESS_SECRET`, `BOT_ADS_WEBHOOK_SECRET`, `RELAY_BOT_URL` e configuração persistida em `public.instances`. A troca do relay e do provider deve ser coordenada; NÃO tentar alternar aleatoriamente `/instance/connect` duas vezes.
2. **JID pareado ≠ E.164 comercial por regra geral:** o dossiê reporta JID da Evolution com 12 dígitos e sufixo de dispositivo (`:15@s.whatsapp.net`), enquanto o número oficialmente divulgado tem 13 dígitos. Não equiparar nem corrigir eliminando o 9 por aproximação. `internal/core/JIDPhone` agora remove somente o identificador de *dispositivo* após `:`, preservando todos os dígitos do telefone. `BOT_ADS_PAIRED_JID`, quando comprovado read-only na instância atual, permite cadastrar uma identidade alternativa **exclusivamente para detectar eco da própria conta**, nunca para mudar destinatário de CRM ou enviar para outro número. `internal/instances.Registry` rejeita aliases repetidos entre papéis.
3. **Preflight bug:** `preflight_ctwa_account_switch.py` imprimia `PASS_BINDINGS_ONLY` mas retornava status 2 por comparar com nome antigo; agora código 0 apenas quando os bindings são verificados. Sem prova do JID alternativo, diferença de 12 e 13 dígitos bloqueia em `BLOCKED_ACCOUNT_BINDING`. `PASS_BINDINGS_ONLY` NÃO prova entrega, não muda webhook e não autoriza envio.
4. **Relay sequencial, NÃO paralelo:** o código `internal/relay/server.go::Process` persiste evento com fsync e tenta **Status Engine primeiro, bot depois**. Falha do primeiro atrasa o segundo; ACK 202 da entrada não comprova os dois destinos. O Estado `StatusDone`/`BotDone` é por sink; o encaminhamento Status Engine → SmartDialer continua best-effort até prova de outbox própria.
5. **Spool e disco:** o recibo reportou 19.069 tombstones concluídos, ~4,59MB, limite ampliado de 20.000 para 100.000 eventos e bytes para 256MiB. A ocupação por slots passou a 19,07% **se o container realmente usa 100.000**. Confirmar por `docker inspect`/Compose efetivo, e não só por `.env` ou exemplo. O índice em memória acelera retries, mas só reconstitui pendências a partir dos JSON no startup. Tombstones de 7 dias protegem dedupe; **não apagar** para liberar espaço sem política/revisão. O dossiê também relatou 9,7GB livres após limpeza de cache Docker; não repetir prune indiscriminadamente.
6. **Rebuild/cutover ≠ canário:** foto, áudio, PDF, CRM idempotente e menu silencioso devem ser observados por uma sequência consentida; nenhum replay automático de 29–31/10. Evitar testes de massa e não alterar número do anúncio da Meta por inferência.
7. **PR #41 Status Engine:** ainda é uma dependência. Verificar o valor privado `SMARTDIALER_WEBHOOK_URL` e a versão realmente em execução antes de fazer deploy da PR #41; não interromper o repasse ao SmartDialer por engano.

## Evidências e checks obrigatórios (modo leitura)

| Gate | Condição de PASS | Bloqueio |
|---|---|---|
| SHA/digest | `HEAD` PR #10, SHA completo da imagem/label e containers são rastreáveis | Imagem de data antiga ou sem commit reproduzível |
| Evolution | `evydencia` connected, instância correta, webhook para relay com segredo rotacionado, subscriptions verificadas | URL antiga/status-engine direto, segredos expostos, global duplicado, reconnect inesperado |
| Conta comercial | E.164 anunciado e JID pareado documentados **separadamente** e associados à MESMA conta por API/provider | Inferência de 12=13 sem documento do provider |
| Relay | volume correto, `RELAY_MAX_EVENTS` efetivo, bytes, latência, contadores `StatusDone/BotDone`, não apenas health | pendentes antigos, fila sem espaço, ACK do status sem ACK do bot |
| Bot | role Ads sem Greeting/fallback, `IsFromMe`, alias próprio, gatilho exato, fila/opt-out/cooldown | qualquer envio espontâneo para conversa orgânica |
| CRM | recibo despacho nativo isolado, sem duplicar `request_id`, nenhuma mensagem final antiga | 422, timeout não conciliado ou disparo duplicado |
| Vizinhos | Status Engine/SmartDialer continuam recebendo, sem reinícios não planejados | forwarding quebrado ou webhook sobrescrito por rotina |
| CI | testes Go/Python/race e scanner de segredos no HEAD exato PASS | falha, cancelamento ou teste inexistente |

### Comandos seguros, executados na worktree local (sem enviar mensagens)

```powershell
git fetch --prune
git status -sb
git rev-parse HEAD
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./internal/relay ./internal/ingress ./internal/outbox
python -m unittest scripts/test_preflight_ctwa_account_switch.py -v
python -m unittest scripts/test_relay_forensic_readonly.py -v
python scripts/check_ctwa_public_docs.py
```

Na VPS, acessar exclusivamente os containers/volumes com `docker inspect`, `docker logs` com *redação de credenciais*, `GET /instance/all` e os preflights read-only existentes. Os valores de segredos devem permanecer apenas em arquivos privados do operador; relatório contém checks e métricas agregadas.

Preflight JID/conta com arquivos privados modo 0600:

```bash
python3 scripts/preflight_ctwa_account_switch.py \
 --bot-env /PRIVATE/current-bot.env \
 --relay-env /PRIVATE/current-relay.env \
 --inventory-json /PRIVATE/evolution-instances.json \
 --verify-live-status --provider-base-url http://127.0.0.1:14000
```

Se o provider registrar uma identidade alternativa de 12 dígitos com device, cadastrar `BOT_ADS_PAIRED_JID` **no ambiente do serviço** após comprovação de `GET /instance/all`, mas nunca alterar `BOT_ADS_PHONE` (número comercial) silenciosamente. `BOT_ADS_PAIRED_JID` pode permanecer vazio quando os dois valores coincidirem.

### Canário e contingência

**Nenhum canário foi observado nesta auditoria GitHub.** Quando operacionalmente autorizado, usar número de teste próprio e sem histórico comercial pendente: mensagem orgânica → sem resposta, `IsFromMe` → sem envio, gatilho CTWA exato → um único fluxo e recibo CRM. Conferir separadamente as entregas Evolution → relay → Status Engine → Bot, a ordem dos ACKs e a idempotência; atenção aos eventos cujo aceite 202 antecedeu erro do segundo sink.

Se houver falha: manter snapshot privado da URL anterior e do spool, restaurar rota somente conforme contrato efetivo do fork, reconciliar mensagens pendentes e **não parar o bot nem limpar o relay às cegas**. A volta ao endpoint antigo do Status Engine pode restabelecer telemetria, mas não comprova que o bot receberá leads. Parar o bot pode complicar a conciliação e não é condição automática de rollback.

### Fontes oficiais e observações de versão

- Evolution Go: https://github.com/evolution-foundation/evolution-go/blob/main/docs/wiki/guias-api/api-instances.md — `POST /instance/connect` usa token da instância, aceita `webhookUrl` e `subscribe`.
- Evolution Go: https://github.com/evolution-foundation/evolution-go/blob/main/docs/wiki/recursos-avancados/events-system.md — global+por instância, retries, eventos e possíveis múltiplos destinos.
- Evolution Go: https://github.com/evolution-foundation/evolution-go/blob/main/CHANGELOG.md — tag 0.7.2. **As páginas main podem divergir do binário modificado**; confirmar contrato pelo digest/Swagger da imagem implantada, nunca por exemplo de outra versão.
- Fluxo efetivo: `internal/relay/server.go`, `internal/relay/store.go`, `internal/core/core.go`, `cmd/botwhatsapp/main.go`, `scripts/preflight_ctwa_account_switch.py`.

**Status de gestão:** `CODE_READY_PENDING_RUNTIME_PROOF`. Rotação de segredos + verificação das duas identidades + canário continuam P0. Nenhuma troca de webhook, emissão CRM, rotação real, deploy ou modificação do spool foi feita pelo ChatGPT neste corte.
