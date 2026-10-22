---
name: outbox-production-audit
description: Validar o worker Go e a segurança das duas instâncias Evolution Go antes de qualquer implantação; evitar ACK prematuro, media fictícia, token cruzado e reenviar campanhas CRM 416.
---
# Skill — auditoria final do BotWhatsApp (Flash Lite)

Leia `AGENTS.md` e `docs/FORENSIC-PR4-20261022.md`. PR #4 é base consolidada, e esta PR de hotfix é empilhada sobre ela.

1. Sem mudanças locais perdidas: `git status -sb`, `git fetch origin`, comparar PRs com `git log --oneline --all -20`. Não force-push.
2. Valide ACK apenas depois do `CommitEventTasks` fsync + marker final. `go test ./internal/ingress ./internal/outbox`.
3. Simule o mesmo evento 100x, falha de disco antes do marcador, crash durante `attempt_started`, fail-closed de contato corrompido e opt-out PARAR.
4. Verifique que cada UUID usa seu `EvolutionSender` e token próprios e que a CRM 416 nunca usa UUID como `message_instances.id`.
5. O boot NÃO deve ter gatilhos Ads genéricos, mídia placeholder nem bind aberto por padrão. Se asset ausente, resultado media SUPPRESSED sem chamada HTTP externa; CRM pode seguir conforme sua própria autorização.
6. Verifique pausa humana registrada de imediato e guardas antes de executar tarefa já enfileirada; saudação deve revalidar cooldown imediatamente antes do envio.
7. Confirme `BOT_OUTBOUND_ENABLED=false` e `BOT_OPERATIONAL_AUTHORIZATION=false` no ambiente de teste. Não efetuar POST real da campanha 416, que já teve canário.
8. Rode `go test -count=1 ./...` uma vez. Verifique Actions e não execute suites repetidas indiscriminadas.
9. **Bloqueador de produção:** o fan-out do webhook `anderson` para o status-engine ainda não existe. Não trocar webhook nem instalar outro app.
10. Entrega: PASS/FAIL dos testes, SHA da árvore, riscos ainda presentes (1 réplica, retenção spool/claims e receipts), nenhuma afirmação de deploy realizado.
