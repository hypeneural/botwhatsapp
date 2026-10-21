# Evyd WhatsApp Bot — Evolution Go 0.7.2
Automação futura de dois WhatsApps do Estúdio Evidência. **Projeto fase 1: PREVIEW somente, sem envio de WhatsApp ou campanha CRM.**

## Números e fluxos

| Número | Papel | Workflow futuro |
|---|---|---|
| **48 3220-0440** (`554832200440`) | Meta Ads | Frase exata → texto acolhimento → áudio do gestor → agendamento CRM Natal 2026 (415/416) |
| **48 99642-5287** (`5548996425287`) | Saudação | Uma imagem com caption/menu 1–5 → opções, cooldown e pausa humana |

**Situação das sessões em 21/10/2026, pelo relatório recebido:** a Evolution Go tinha apenas a instância `Anderson` desconectada, vinculada a outro número e ao motor de status. Os dois números acima **não constavam como instâncias na GO**. Nunca reutilizar a instância `Anderson`. Provisionamento dependerá de autorização explícita.

## Comece aqui
1. `AGENTS.md` — regras do Antigravity, limites de produção e segurança.
2. `docs/FORENSIC-AUDIT-2026-10-21.md` — análise dos achados e correções forenses.
3. `docs/IMPLEMENTATION-NEXT.md` — handoff do trabalho seguinte.
4. `.agents/skills/evolution-go-v072-adapter/` — skill curada da GO instalada.
5. `.agents/skills/bot-contract-tests/` — testes de contratos offline.

## Código já publicado

- `internal/core`: parser de evento GO, normalização de texto, resolução BR, dedup sem inferir LID, bloqueio de opt-out.
- `internal/workflow`: planos determinísticos e decisão correta de cooldown; **não executa efeitos**.
- `internal/ingress`: receptor HTTP com segredo independente, validação UUID e registro atômico por ID de mensagem em `data/spool`. Apenas `planned_preview_only`.
- `internal/evolution`: cliente HTTP `/send/text` e `/send/media` isolado, sem retry incerto. **Não conectado ao servidor**.
- `internal/crm`: contrato das campanhas 415/416, header Authorization raw, cliente isolado que exige `message_instances.id` positivo. **Não conectado ao servidor**.
- `cmd/botwhatsapp`: processo HTTP somente prévia, sem saídas comerciais.

## Testes locais
```bash
go test ./...
go vet ./...
```
O projeto usa somente a biblioteca padrão Go 1.23. A imagem Docker usa build multi-stage, sem exigir Go instalado no host Contabo. Consulte o workflow CI em `.github/workflows/go-test.yml` para evidência. O modo preview não equivale a aprovação de deploy.

## Modo preview (somente ambiente local)
1. Copiar `.env.example` para .env local (ignorado por Git). Variáveis não são carregadas automaticamente pelo Go: exportar pelo terminal/IDE sem exibir segredos.
2. Manter `BOT_ENABLED=false` até existir UUID verdadeiro, dois segredos longos e permissão de teste.
3. Com `BOT_ENABLED=false`, apenas `GET /healthz` confirma processo; `/readyz` e webhooks retornam indisponibilidade.
4. Quando preview autorizado e **somente com fixtures sintéticas**, habilitar variáveis locais, sem configurar o webhook real na GO. Ingresso `/webhook/ads/<segredo>` ou `/webhook/greeting/<segredo>` retorna 202 se gravado em spool privado.
5. Não há worker; nada será enviado ao WhatsApp ou CRM neste estágio.

## Não incluído e ainda obrigatório para produção
Duas instâncias pareadas, UUIDs, texto do anúncio, áudio e imagem aprovados, ID numérico da instância de SAÍDA no CRM, verificação de campanhas ativas, outbox durável de efeitos, pausa humana/cooldown persistente, opt-out persistente e canário autorizado. **Não reconfigurar Evolution Go ou CRM até conclusão desses itens.**

Este é um serviço futuro independente: não modificar `hypeneural/evydencia-crm`, Caddy, Asterisk ou aplicações vizinhas da VPS.

## Simulador offline para o AntiGravity
Sem ligar nenhum número real, o agente pode executar:
```bash
go run ./cmd/simulate ./testdata/ads.json
go run ./cmd/simulate ./testdata/greeting.json
go run ./cmd/simulate ./testdata/menu-human.json
go run ./cmd/simulate ./testdata/opt-out.json
```
Os arquivos são **sintéticos**; `instance=999999` impresso pelo simulador é EXCLUSIVAMENTE fictício. Instruções para o próximo ciclo: `docs/PROMPT-ANTIGRAVITY-NEXT.md`. Performance: `docs/PERFORMANCE-POLICY.md`.
