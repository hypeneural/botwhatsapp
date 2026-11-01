# ANTIGRAVITY — EXECUTAR HOTFIX CTWA P0 E PREPARAR TROCA FUTURA DE NÚMERO

**Meta:** remover a mensagem final prometendo encaminhamento por outro WhatsApp, deixar Ads configurável por número/UUID sem editar Go novamente, homologar e organizar a operação existente. NÃO fazer cutover de número ou envio real antes dos gates.

**Única worktree Bot:** PR #10 `fix/ctwa-native-proposal-handoff-20261027`, repositório `hypeneural/botwhatsapp`.  
**Dependência:** SmartDialer PR #41 `fix/evolution-webhook-ownership-p0-20261031`. Não alterar sua implementação de campanhas/VoIP. CRM/Z-API são somente leitura.

## 1. Sincronizar e entender

```powershell
git fetch --prune
git status -sb
git rev-parse HEAD
```

Fast-forward seguro somente se a worktree estiver limpa. Não criar outra PR e não desfazer os novos commits do ChatGPT.

Ler PRIMEIRO:
- `AGENTS.md`.
- `docs/CTWA_ACCOUNT_SWITCH_AND_HANDOFF_20261031.md`.
- `docs/AUDITORIA_VALIDACAO_CHATGPT_P0_20261031.md`.
- `docs/CTWA_GREETING_P0_20261029.md`.

## 2. Código já alterado — não reimplementar

`cmd/botwhatsapp/phone_config.go` já aceita `BOT_ADS_PHONE` E.164 BR válido, sem 5287 fixo. `instances.Registry` valida UUID/slug/número e separa chave por role. `workflow/plan.go` não cria mais `SendAdClosingText`. `outbox/worker.go` suprime mensagens finais antigas antes de qualquer rede, mantendo a tarefa CRM dependente; a saudação fallback também não promete outro número. `main.go` não lê mais `BOT_ADS_CLOSING_TEXT_FILE`.

O novo fluxo esperado é **reação (opcional) → boas-vindas → áudio → PDF → imagem/vídeo opcionais → despacho CRM V4 → fim**. Nunca adicionar mensagem de encaminhamento depois da API. Verificar se o ÁUDIO real menciona um segundo número; não editar mídia sem autorização.

## 3. Validação local única

```powershell
go test -count=1 ./internal/core ./internal/instances ./internal/workflow ./internal/outbox ./internal/ingress ./cmd/botwhatsapp
go vet ./...
python -m unittest scripts/test_preflight_ctwa_account_switch.py -v
python -m unittest scripts/test_relay_forensic_readonly.py -v
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1
```

Verificar workflow Go CI e Container Build no HEAD exato. Corrigir apenas falhas determinísticas e executar testes afetados uma vez. Não instalar nova stack, não fazer testes infinitos. Não disparar WhatsApp nos testes.

## 4. Homologação READ-ONLY de identidade

Para a conta Ads **ATUAL**, obter snapshot de `GET /instance/all` na Evolution Go e conferir:
- Número comercial conectado e UUID da sessão.
- `BOT_ADS_PHONE` + `BOT_ADS_INSTANCE_UUID` + `EVOLUTION_ADS_APIKEY` de produção (sem mostrar).
- UUID esperado do relay, `RELAY_BOT_URL`, `RELAY_STATUS_URL`, segredo do ingress e rede privada.
- Que papel Greeting não use o mesmo telefone.
- `BotAdsFallback` efetivo **OFF no código** apesar de env antigo possivelmente true.
- Spool antigo preservado e nenhuma tarefa pendente/iniciada/incerta na troca.
- O watcher da PR #41 não sobrescreva a URL e o `SMARTDIALER_WEBHOOK_URL` privado esteja configurado antes de eventual deploy do Status Engine.

Para preparar **FUTURA TROCA**, sem alterar tráfego ativo, executar:
```bash
python3 scripts/preflight_ctwa_account_switch.py \
  --bot-env /PRIVATE/candidate-bot.env \
  --relay-env /PRIVATE/candidate-relay.env \
  --inventory-json /PRIVATE/evolution-instance-all.json \
  --previous-bot-env /PRIVATE/current-bot.env \
  --outbox-tasks-dir /PRIVATE/current-spool/tasks
```

Arquivos devem ser 0600; somente o resumo de booleanos pode ir ao Git. Se a resposta do inventário não incluir `token`, habilitar `--verify-live-status --provider-base-url http://127.0.0.1:14000` no próprio preflight, para comprovar a chave com GET autenticado de status e JID da sessão. Bloqueie em qualquer resposta incerta; jamais desabilite o gate.

**A TROCA DE NÚMERO NÃO FOI PEDIDA NESTE CORTE.** Não alterar atualmente a instância 5287, Meta Ads, Evolution, CRM, relay ou as rotas por conta própria. Um futuro plano de troca deverá incluir atualização do número de destino dos anúncios Meta e rollback de ambas as instâncias.

## 5. Cutover do hotfix atual (somente após liberação operacional)

- Verificar SHA e digest da imagem ativa de 27/10, backup do spool/journal, um worker e Compose real.
- Verificar configuração sanitizada do número 5287 e mediadores Status Engine/SmartDialer.
- Compilar/recriar SOMENTE o serviço bot, preservando volume, sem reiniciar Evolution/Status Engine/Asterisk.
- No canário consentido, uma mensagem comum no 5287 NÃO causa saudação; mensagem `IsFromMe` NÃO causa proposta; gatilho exato num único contato de teste segue o funil e agenda CRM sem etapa de encaminhamento.
- Conferir em logs redigidos o estado `confirmed/suppressed/uncertain`; `dispatch_uuid` confirma despacho, NÃO entrega. Não repetir eventos históricos.

## 6. Entrega do AntiGravity

Criar na mesma PR #10 `docs/RECIBO_ANTIGRAVITY_CTWA_NUMBER_SWITCH_20261031.md` com:
HEAD, testes, CI, SHA/digest da imagem, porta/relay/UUID/telefone **mascarados**, contagens de tarefas antigas, comparação das etapas, trecho redigido do resultado do canário, erro/blocker, rollback e GO/NO-GO.

NÃO exportar números de clientes/tokens, NÃO copiar secrets, NÃO reprocessar 29–31/10, NÃO ativar campanhas. **Código preparado ≠ runtime homologado.**
