---
name: vps-readonly-audit
description: Auditar em modo passivo uma VPS Contabo compartilhada, sem alterar serviços.
---
# Auditoria VPS — primeira skill

Leia `AGENTS.md` e `docs/START-HERE.md`. Primeira etapa é somente leitura.

1. Verificar se há acesso SSH já autorizado pela configuração local. Sem acesso: registrar NÃO CONFIRMADO, não pedir senha nem tentar escalar privilégio.
2. Antes de executar, revisar `scripts/vps-readonly-inventory.sh`. Rodar somente com consentimento e acesso previamente disponível: `ssh -T ALIAS_SSH 'bash -s' < scripts/vps-readonly-inventory.sh`.
3. Inventariar apenas SO, CPU/memória, Docker/Compose, nomes/imagens/status/portas de containers, redes por nome e serviços em operação; identificar dependências compartilhadas.
4. Não consultar logs, payload de webhook, .env, banco, chaves, contatos ou histórico do WhatsApp. Não executar `docker exec`, `docker inspect` completo, reboot, restart, update, compose up/down ou alteração de firewall.
5. Revisar saída, remover IPs externos/nome privado de host/dados sensíveis antes de criar relatório `reports/vps-inventory.redacted.md`, arquivo ignorado por Git.
6. Resultado por item: CONFIRMADO, NÃO CONFIRMADO, RISCO, PRÓXIMA AÇÃO. **Não implantar ou reconfigurar nada.**
