# P0 — Evitar saudação e campanha em conversas próprias / orgânicas (29/10/2026)

**Scope:** `hypeneural/botwhatsapp` PR #10, **zero alterações no Laravel** ou no provedor Evolution. Esta política faz parte do bot e NÃO declara implementação em produção antes do canário.

## Causa forense confirmada

1. `cmd/botwhatsapp/main.go` ativava `BOT_ADS_FALLBACK_MENU` como true por padrão. `internal/core/core.go::Select` interpretava qualquer texto não correspondente no **mesmo número do tráfego pago 5287** como `Greeting`, e opções 1–5 como resposta automática do menu. Portanto, respostas orgânicas e conversas iniciadas pela equipe podiam ser atendidas indevidamente por um bot.
2. Havia guarda `IsFromMe=true`, mas o webhook de saída era respondido com 204 sem persistir estado de conversa do destinatário. Não existiam `LastInboundAt`, `LastOutboundAt` e `LastActivityAt` para o bot considerar iniciativa do estúdio.
3. O `Registry` possui `BusinessPhoneE164`, porém esse número não era propagado para `core.Select`; se o webhook viesse com o próprio número como `Chat` e direção incorreta, o fluxo poderia ser acionado pela frase literal.
4. O CRM Laravel **separado**, `WhatsappConversationStateService`, já implementa janela de 120h, lock e cancelamento por `fromMe` da Z-API. Isso NÃO cobre automaticamente eventos Evolution Go neste bot; não presumir que os dois provedores partilham estados sem prova.

## Invariantes novos na PR #10

- **Instância Ads (5287):** somente frase literal de `config/ads-triggers.txt` pode criar `AdsProposal`. Qualquer outra mensagem, inclusive `Oi`, texto de atendimento humano, contexto de Meta sem texto aprovado e os dígitos do menu, resulta em `Ignore`, motivo `ads_unmatched_no_automation`. **NUNCA envia saudação genérica**; `BOT_ADS_FALLBACK_MENU` tornou-se legado sem efeito, mesmo se ambiente antigo ainda disser true.
- **Proteção da própria linha:** `Registry` injeta `BusinessPhoneE164`; se o `Chat` é o próprio número, é `own_business_number` e não gera proposta. Se `IsFromMe=true` **ou** `Sender` é o JID E.164 comprovado da própria linha, o evento é `OutboundObserved` e também nunca gera proposta.
- **Memória de conversação:** o webhook `OutboundObserved` grava somente horário de atividade por E.164 no spool privado e retorna 204, **sem salvar conteúdo da mensagem de saída no journal de leads**. Entradas ignoradas mas identificáveis também registram horário; essas entradas não geram tarefas.
- **Instância Greeting independente (se autorizada):** saudar apenas quando não houve atividade registrada nos últimos **5 dias / 120h**, nem houve saudação dentro do cooldown ou pausa humana. Antes do envio real, um outbound registrado nesse intervalo cancela tarefa antiga enfileirada; estados `opt_out` e `human_paused` continuam prioritários.
- **Painel:** entrada Meta sem frase reconhecida e com contexto de ad fica `atendimento_manual` (requer atendimento pela equipe, sem criar tarefa). Entrada orgânica sem contexto fica `conversa_organica`; os indicadores não podem confundir o primeiro com envio realizado.
- **Segurança:** nenhum `@lid` é convertido em telefone por dígitos. Payload `IsFromMe`/Sender de fornecedor é confiável somente após autenticação do webhook + UUID da instância. O bloqueio da própria linha precisa ser validado com a versão real Evolution Go. Eventos de CRM Z-API ou do WhatsApp Web que não apareçam no webhook Evolution ainda **não** são observáveis pelo bot.

## Matriz de comportamento esperado

| Situação | Ação |
|---|---|
| Primeiro clique com frase exata aprovada e telefone externo | `ads_proposal`, respeitando cooldown, opt-out, confirmação CRM |
| Mensagem comum "Oi", "Tudo bem?" ou resposta à equipe no 5287 | Silêncio do bot; fica no painel como conversa normal |
| Meta Ads com `referral` mas sem frase aprovada | Sem resposta automática; marcar para atendimento humano |
| Mensagem enviada pelo estúdio com `IsFromMe=true` | Registrar atividade; zero resposta/CRM; 204 |
| Sender é a linha oficial e `IsFromMe=false` incoerente | Tratar como outbound (sem automação) |
| Chat é o próprio número oficial do estúdio | Ignorar, sem atribuir o número como lead |
| Contato responde a saudação iniciada pelo estúdio nos últimos 5 dias | Sem nova saudação |
| Contato novo ou sem atividade por >=120h em instância Greeting separada | Saudação elegível, respeitando demais gates |
| Tarefa de saudação enfileirada seguida de mensagem de saída do estúdio | Tarefa suprimida antes do provider I/O |
| Bot de anúncios e saudação CRM conectados à mesma linha | Auditar origem exata; NÃO criar segunda saudação automática no bot |

## Descoberta independente LIVE: Z-API não notifica mensagens enviadas pelo estúdio

Em **29/10/2026 às 09:52 BRT**, `zapi_instancia_diagnostico` (escopo
`perfil_dispositivo`, exclusivamente leitura) confirmou uma instância
Z-API conectada ao número oficial terminado em **5287**. Na verificação
read-only `assinatura_configuracoes`, o provedor retornou:

```text
connected = true
sent_by_me_enabled = false
```

Esse último estado é **compatível com a causa de saudações indevidas
originadas no CRM**, já que o `WhatsappConversationStateService` precisa
receber `ReceivedCallback.fromMe=true` para registrar o atendimento
iniciado pelo próprio WhatsApp/operador. A ausência da notificação
significa que o CRM pode não enxergar essas saídas; não é evidência de
bug no bloqueio de 120h do código PHP.

Documentação oficial do provedor:
https://z-api-docs.vercel.app/en/webhooks/update-notify-sent-by-me
A flag `notifySentByMe=true` habilita callback para mensagens
enviadas pelo número (quando o webhook de recebimento está configurado).
**NÃO ativar cegamente em produção.** Verificar primeiro o roteamento
atual de ReceivedCallback, processamento idempotente de mensagens
`fromMe`, carga esperada, diferença de mensagens enviadas pela API
versus celular/WhatsApp Web, e testes de canário controlados. Nenhuma
chave, URL privada ou nome/telefone de cliente deve ir ao Git/chat.
O CRM `WhatsappHookService`/`WhatsappConversationStateService` deve
ser inspecionado em modo somente leitura. Se a configuração for
aprovada, fazer UMA alteração específica da Z-API e testar recebimento
`fromMe=true` sem disparar automação de saudação, junto com rollback.

Importante: o bot Evolution Go e a Z-API usam a mesma linha comercial,
mas isso **não comprova** que ambos recebem os mesmos webhooks. Cada
sistema tem estado de saudação próprio; os guards do bot não substituem
a notificação de saída necessária ao CRM.

## Pontos ainda pendentes de prova real

- O status de CI prova regressões e compilação, **não** que `fromMe` chega para mensagens do WhatsApp Web ou CRM por outra conexão. Verificar em amostras **sanitizadas** do webhook, sem nomes/telefones/tokens.
- No bot, o marcador de atividade usa horário de chegada do webhook, não o instante declarado pelo provedor. Mensagem de saída atrasada pode ser observada depois do cliente; sem prova de causalidade, a ausência de evento não autoriza inferir que a conversa foi iniciada pelo cliente.
- A integração do CRM Z-API controla a sua própria saudação, com janela/locks independentes. Requer auditoria de configuração de campanhas/instâncias se o sintoma continuar mesmo com `BOT_ADS_FALLBACK_MENU` desativado.
- Persistência do spool deve manter o mesmo volume/UID e apenas um worker. Não reiniciar o bot antes de snapshot e CI verde; não limpar contatos/claims/journal.
- Em conversa própria com a frase aprovada exatamente reproduzida, caso não haja evidência de um outbound anterior, o gatilho literal permanece elegível por desenho. Uma próxima evolução poderá exigir `referral` com comprovação de conversão/ad origem e exclusão de operadores; **não** endurecer esse gate sem provas de cobertura dos eventos reais Meta.

## Testes obrigatórios por HEAD

```bash
go test -count=1 ./internal/core ./internal/instances ./internal/ingress ./internal/outbox ./internal/dashboard ./cmd/botwhatsapp
go vet ./...
```

Conferir também `go test -count=1 ./...` e CI GitHub. No Avell, usar `scripts/validate-ctwa-handoff-local.ps1` (somente local). Ao testar na VPS, primeiro confirmar os flags de produção, backup do spool e identidade do container, sem mexer no CRM, Z-API, Evolution, PrintServer ou Caddy.

## Critérios GO/NO-GO

**GO código** apenas com CI HEAD verde e os novos testes acima; **GO implantação exclusiva do bot** apenas após snapshot, digests, restore de rollback e confirmação do operador; **GO canário** após 1 evento orgânico/own (zero envio) e 1 evento CTWA autorizado (proposta nova). Sem outros envios e sem reprocessar históricos.

**NO-GO** para afirmar correção completa da saudação Z-API do CRM só com testes deste bot. Verificar `WhatsappConversationStateService` e a origem do disparo em logs privados; jamais criar campanha CRM para teste.
