# AGENTS.md — BotWhatsApp / Antigravity

**Fonte canônica de instruções.** O arquivo `agent.md` é um alias. Leia `docs/START-HERE.md` e a skill `.agents/skills/relay-local-preflight/SKILL.md`.

## Verdade operacional (22/10/2026)
- Repositório-alvo de escrita: `hypeneural/botwhatsapp`. `hypeneural/evydencia-crm` é **SOMENTE LEITURA** — não editar código, tabelas, campanhas ou deploy do CRM.
- Na Contabo estão Evolution Go 0.7.2 modificada, Postgres, Caddy, SmartDialer/Asterisk, WhatsApp Status, vídeo, música, SMS e demais stacks. **Não reiniciar, parar, mover portas, alterar Caddy, rede, firewall, banco, tokens, webhook ou instâncias** durante preflight.
- Novo relay NÃO deve ocupar porta 8050 nem exigir mudança do status-engine para 8051. O motor existente continua servindo no endereço original.
- `main` tem CRM 416, motor outbox, event dedup, lead claim BRT, opt-out, cooldown e pares de instâncias. O relay independente é NOVO e opcional, em staging, desativado por padrão.
- Modo produção ainda BLOQUEADO até confirmar idempotência do status-engine, quotas de tamanho/disco, fan-out com canário autorizado, asset oficial e sessão de saudação.
- Go 1.27.3 compilação/CI, Go1.23 linguagem mínima do módulo (compatibilidade local). Imagem Alpine 3.24.2, com tzdata/CA. Não instalar Go na VPS anfitriã.

## Contrato comercial e identidades
- Ads: **48 3220-0440 / 554832200440** na Evolution Go. Reconhecer somente frase literal aprovada `BOT_ADS_TRIGGERS`. Texto → áudio pré-gravado OGG Opus → agendamento CRM 416.
- Saudação: **48 99642-5287 / 5548996425287** em outra instância Evolution Go (pareamento não atestado). Uma imagem+caption, menu 1–5, cooldown, pausa humana e opt-out.
- CRM: **campanha 416 SEM NOME SEMPRE**, URL `POST /api/campaigns/schedule/execute`, `contacts` com DDD e 10/11 dígitos NACIONAIS, `instance` é ID numérico CRM >0 (relato do canário: 1), Authorization interna raw sem Bearer. Não inferir campanha 415 de PushName.
- Token e UUID GO são independentes em cada número. Nunca deduzir celular do sufixo `@lid`; somente JID telefônico comprovado.
- Confirmação HTTP de Evolution Go ou CRM não prova `delivered` ou `read`.

## Segurança e durabilidade
- `instanceToken` do JSON do webhook NÃO é HMAC. Segredo independente por path, rede privada e mapeamento UUID. Nunca logar URLs de webhook, segredos ou payloads de cliente.
- `internal/ingress` ACK 202 só após outbox durável. Relay grava payload antes do próprio ACK; encaminha status primeiro, bot depois. Em timeout incerto, jamais reexecutar cegamente uma campanha CRM.
- `attempt_started` após crash vira `uncertain`. Opt-out/pause/cooldown têm precedência sobre envios.
- Uma réplica por volume/journal. Persistência por arquivos não oferece transação distribuída entre processos.
- O relay mantém raw JSON sensível temporariamente (volume 0600, scrub após sucesso, tombstone 7 dias); monitoração/quota de bytes e política de dados ainda pendentes.
- Não commitar `.env*` reais, chaves, áudios privados, fotos, dados de clientes, `data/` ou transcript bruto.

## Comandos locais permitidos
```powershell
git status -sb
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-local.ps1
```
O script NÃO acessa VPS, não executa `docker compose up/down` nem chama APIs reais. Preserve trabalho local; nunca `git reset --hard`, `git clean -fd` ou force-push.

## Fluxo CTWA (prioridade do ciclo atual)
- Revise `docs/CTWA-ROUTING-20261022.md` e a skill `.agents/skills/ctwa-ads-funnel/SKILL.md`. Payload da conversa de tráfego pago foi fornecido pelo gestor como relatório, sem novo acesso ao banco.
- A lista de mensagens exatas em `config/ads-triggers.txt` já contém **duas frases reais** (Primeiro Natal e Natal em Tijucas); editar novas frases somente quando comprovadas em criativo ou aprovadas, carregada por `BOT_ADS_TRIGGERS_FILE`. Não usar match genérico, regex ou substring para acionar campanha CRM.
- O registro Ads agora propaga `AdsFallback`, e mensagens não correspondentes recebem saudação/menu no **MESMO** número de anúncios; greeting futuro em instância independente só faz saudação.
- Funil Ads: texto com `{{nome}}` seguro, PTT, imagem opcional, vídeo MP4 opcional, CRM 416 SEM NOME por último, com dependência durável de execução. `BOT_ADS_FLOW_COOLDOWN_HOURS` padrão 24h bloqueia nova sequência do mesmo contato e preserva `PARAR`/pausa humana por atualizações atômicas de contato. Não interpretar `uncertain` como confirmação de entrega.
- O relatório de webhook anexado expôs material como `instanceToken`: NÃO COMMITAR O JSON BRUTO e solicitar verificação/rotação do token se ainda válido.
- Mídias reais não estão incluídas: flags comerciais OFF, nada de disparos reais, CRM somente leitura.

## Documentação por prioridade
1. `docs/PRODUCTION-PREFLIGHT-20261022.md`
2. `docs/RELAY-ROLLOUT-GATES.md`
3. `docs/CRM-API-416-PLAYBOOK.md`
4. `docs/MULTIINSTANCE-ARCHITECTURE.md`
5. `docs/INDEX.md` (separação de arquivos atuais e históricos)
6. `.agents/skills/relay-local-preflight/SKILL.md`

Fontes upstream: https://go.dev/doc/devel/release ; https://www.alpinelinux.org/releases/ ; https://docs.docker.com/compose/how-tos/networking/ ; https://github.com/evolution-foundation/evolution-go/tree/0.7.2 .

**A próxima ação do Antigravity é SÓ VALIDAR LOCALMENTE.** Não ligar saída, parear instâncias ou mudar o webhook.
