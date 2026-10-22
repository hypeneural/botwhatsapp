---
name: canary-forensic-review
description: Revisar evidências de canário real Evolution Go e agendamento CRM 416, distinguir aceite API de entrega e confirmar rollback do webhook sem disparos de teste.
---
# Skill canário forense — Antigravity Flash Lite
Leia `docs/AUDIT-CANARY-AND-MULTIINSTANCE-2026-10-21.md` primeiro.
1. **Não repetir canário**; não trocar webhook de `anderson`, não reenviar a campanha, não chamar `POST` CRM.
2. Localmente inspecionar commits `16514e4`, `10348a6`, `ae40f97`, `git status`, `git show --stat`. Esses commits não foram encontrados na main GitHub auditada; preservar trabalho local.
3. Examinar `internal/outbox`, `internal/ingress`, `cmd/botwhatsapp` dos commits do Antigravity. Conferir fonte que gerou ID CRM, payload, rota real, status e dedup. **Proibir logs brutos com PII e tokens**.
4. Diferenciar: API GO id recebido / recibo entregue / lido; CRM agendado / provedor aceitou / entregue. Se ausente, declarar não confirmado.
5. Verificar **por leitura** o valor persistido do webhook de `anderson` e se status-engine retomou processamento. Snapshot UI não comprova consumidor.
6. Conferir `GET /api/campaigns/schedule/search` (token seguro, nenhum log de PII), somente para conciliar o agendamento canário existente. Não assumir que vazio autoriza repostar.
7. Publicar código local limpo em branch, nunca force-push main, criar PR para revisão; sem segredo, áudio privado, webhook URL secreta, logs ou spool.
8. Saída em 4 blocos: CONFIRMADO POR CÓDIGO, RELATADO PELOS LOGS, NÃO CONFIRMADO, BLOQUEADORES DE PROD.
