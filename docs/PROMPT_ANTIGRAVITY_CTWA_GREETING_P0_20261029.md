# Antigravity — hotfix P0 de saudação indevida e autoacionamento CTWA 5287

**Uma missão:** concluir validação e implantação **exclusiva do bot** após autorização operacional, sem tocar no CRM Laravel, Caddy, PrintServer, SmartDialer, Z-API ou Evolution Go durante a fase de diagnóstico. Mesma PR #10, branch `fix/ctwa-native-proposal-handoff-20261027`. Proibido abrir uma PR nova, dar reset/force push ou limpar spool/journal.

## 1. Ler o código já corrigido

Leia `AGENTS.md` e `docs/CTWA_GREETING_P0_20261029.md`.

**Nova regra:** o número 5548996425287 tem papel Ads. Mensagem comum, resposta de cliente, saudação normal, contexto Meta sem frase aprovada e opção numérica de menu **NÃO** disparam bot. Somente a frase exata autorizada em `config/ads-triggers.txt` inicia proposta. `BOT_ADS_FALLBACK_MENU` é obsoleto e não faz efeito, inclusive se a VPS ainda o tiver como true.

Eventos `IsFromMe=true` ou `Sender` igual ao número comercial devem registrar saída, sem responder ou criar campanha. `Chat` igual ao número oficial também nunca é lead. Históricos ignorados são preservados para auditoria, mas não devem virar nova tarefa.

Saudação de instância Greeting **separada** requer 120h sem contato; no worker ela é cancelada se o estúdio enviar mensagem antes da execução.

## 2. Validação local focada (Avell)

No worktree já existente:
```powershell
git fetch --prune
git status -sb
git rev-parse HEAD
go test -count=1 ./internal/core ./internal/instances ./internal/outbox ./internal/ingress ./internal/dashboard ./cmd/botwhatsapp
go vet ./...
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1
```

Se gofmt falhar, formatar **somente** arquivos novos/alterados e uma vez. Conferir `git diff`, corrigir somente regressões determinísticas e validar CI da PR #10 no HEAD final. Não fazer loops de testes.

## 3. Identificar a origem EXATA da saudação

**Achado LIVE já confirmado** por consulta read-only em 29/10/2026: Z-API conectada ao 5287, `sent_by_me_enabled=false`. Essa flag pode impedir que o CRM observe mensagens iniciadas pelo estúdio, embora o PHP já tenha o gate de 120h.

A Z-API possui documentação oficial em `https://z-api-docs.vercel.app/en/webhooks/update-notify-sent-by-me`. Antes de habilitar qualquer configuração, auditar em modo leitura:
- URL/evento `ReceivedCallback` do CRM, webhook de mensagens enviadas por mim, mapeamento da conta favorita e `WhatsappHookService`.
- `WhatsappConversationStateService::recordReceived` e `scheduleIfReserved`: garantir que `fromMe=true` apenas registre saída/cancele saudação, nunca execute CTA/menu/CRM.
- Diferença entre mensagens enviadas pelo celular/WhatsApp Web versus API; `DeliveryCallback` e `ReceivedCallback` têm semânticas diferentes. Verificar `notifySentByMe` de fato na instância e registro de atividade por conta/telefone.
- Confirmar por hashes/IDs redigidos se uma saudação real veio do BotWhatsApp Evolution ou do CRM Z-API; sem misturar autores.

**Não fazer PUT na Z-API nesta fase.** Se o problema for mesmo `sent_by_me_enabled=false`, preparar a mudança cirúrgica `notifySentByMe=true` para aprovação operacional específica, com estado anterior preservado, dedupe, telemetria sem PII e procedimento reversível. Nunca mostrar token ou URL de webhook.

## 4. Validar painel e estados

`https://bot.etijucas.com.br/dashboard`:
- Exibir `conversa_organica` para texto comum (nenhum envio).
- Exibir `atendimento_manual` para contexto Meta sem frase autorizada (operador deve ver e atender).
- Exibir recibo do CRM somente para proposta agendada (não entrega).
- Não registrar a mensagem enviada pelo próprio estúdio como lead inbound.
- Confirmar que o mesmo número oficial não é aceito como destinatário.

## 5. Homologação e corte futuro controlado

Verifique SHA **implantado** versus SHA PR (não confundir checkout git com imagem running). Antes de qualquer deploy: backup spool e imagem, um worker, env privado, health e rollback validado. Atualizar SOMENTE bot, após todos os gates; nunca reiniciar CRM/Evolution ou repetir envios históricos.

Canários autorizados, de preferência com contatos de teste e sem campanhas reais:
1. `IsFromMe` / resposta à mensagem do estúdio: zero saudação ou campanha.
2. Mensagem orgânica `Oi` no 5287: zero automação, status `conversa_organica`.
3. Meta referral sem frase exata: zero envio; status `atendimento_manual`.
4. Frase exata de um lead de teste autorizado: UMA sequência CTA permitida e `dispatch_uuid` CRM, sem duplicidade e sem inferir entrega.
5. Quando a Z-API for ajustada em procedimento separado, confirmar que mensagem iniciada manualmente gera `fromMe=true` no CRM e bloqueia saudação da resposta.

## Entrega (objetiva, sem PII)

Informe SHA, CI, source da saudação, flags Z-API (somente true/false), número de casos por classificação, recibo de testes, spool e digests preservados, resultado dos canários, GO/NO-GO. **Não** imprimir segredos, telefones de clientes ou mensagens reais. Corrigir somente o que foi reproduzido; sem testes infinitos.
