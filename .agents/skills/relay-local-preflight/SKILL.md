---
name: relay-local-preflight
description: Valida localmente o relay Go durável, DNS Compose de staging sem deploy e o gate de cutover de webhook, sem qualquer alteração em Evolution Go, status-engine, CRM ou VPS.
---
# Skill relay — Flash Lite / Antigravity

Leia `AGENTS.md`, `docs/RELAY-ROLLOUT-GATES.md` e `deploy/README.md`. A porta 8050 do motor de status NÃO MUDA, nem o status-engine é reiniciado.

**Somente no Windows local:**
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-local.ps1
```
O script executa exatamente um `go vet`, um `go test -count=1 ./...`, build dos dois comandos, e `docker compose config --quiet` apenas quando Docker local existe. Não faz chamadas para a Contabo.

Conferir `internal/relay/relay_test.go`: ACK 202 após fsync; status-engine primeiro e bot depois; tratamento de 503/timeout; dedup após reinício; corpo com PII removido após sucesso; capacidade máxima com HTTP 503, sem descarte silencioso.

**Bloqueios:** falta acesso ao source do status-engine para confirmar idempotência, falta medir tamanhos de webhook e quota em bytes, conexão da saudação não confirmada, mídias não fornecidas. Não utilizar `docker compose up` ou alterar `/instance/connect` durante esta missão.

Resultado: informar SHA, testes PASS/FAIL, riscos não resolvidos e se o código está pronto para uma autorização futura de canário. Não afirmar DEPLOY.
