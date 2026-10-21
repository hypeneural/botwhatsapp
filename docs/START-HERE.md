# Início rápido — Antigravity Flash Lite

**Hoje:** ler, simular e implementar offline. **Não instalar nem disparar.**

## 1. Estado confirmado vs relatado
- Fase 1: inventário da VPS relatado pelo AntiGravity, resumido com ressalvas em `docs/FORENSIC-AUDIT-2026-10-21.md`.
- Evolution Go anunciada na Contabo: 0.7.2 com patch local; duas instâncias dos números-alvo NÃO encontradas naquele inventário.
- Repo atual: código Go `preview-only`, não dispara WhatsApp/CRM.
- Contrato CRM: campanhas 415 (com nome) e 416 (sem nome) identificadas no código; estado live ainda pendente.

## 2. Primeiros comandos no clone local
```bash
git status --short
# Se working tree limpo:
git pull --ff-only origin main
go test ./...
go run ./cmd/simulate ./testdata/ads.json
go run ./cmd/simulate ./testdata/greeting.json
go run ./cmd/simulate ./testdata/menu-human.json
go run ./cmd/simulate ./testdata/opt-out.json
```
Fixtures são fictícias e NÃO utilizam tokens, números reais de clientes ou rede. Em máquina sem Go, primeiro leia GitHub Actions PASS; não instalar Go na VPS só para rodar os testes.

## 3. Próximo desenvolvimento
**Siga o prompt completo em `docs/PROMPT-ANTIGRAVITY-NEXT.md`.** O principal trabalho restante é outbox durável, cooldown/pause/opt-out persistentes, recuperação após reinício e testes mocks.

## 4. Regras que não podem ser violadas
- `hypeneural/evydencia-crm` = SOMENTE LEITURA.
- Nenhuma alteração em Evolution GO, Caddy, Asterisk, SmartDialer, motor de status, Captive Portal ou outros serviços.
- Não autenticar webhook pelo campo `instanceToken`; segredo independente por rota/rede privada.
- A variável correta do webhook GLOBAL do Evolution Go é `WEBHOOK_URL`, não `GLOBAL_WEBHOOK`.
- É preciso confirmar instância de envio CRM, não confiar automaticamente em seu favorito.
- Imagem, áudio, frase real, pareamento e campanha ativa são pendências para fase autorizada.
