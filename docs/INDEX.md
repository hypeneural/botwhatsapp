# Índice — documentação vigente vs histórico

## COMEÇAR AQUI (canonical)
- [Gates P0 de implantação real somente do bot](RELEASE-BOT-CTWA-5287-20261027.md): inventário Contabo, preflight privado sem segredos, backup do journal, um worker/canário e rollback do binário. **Não usar Compose staging.**
- [Painel do gestor (mobile + preview local)](PAINEL-GESTOR-CTWA-20261027.md): filtros compreensíveis, recibos, significado dos estados e homologação visual offline de 5 cenários. Fluxo recomendado para gestor e Antigravity.
- [Hotfix CTWA 5287 → CRM V4 + histórico auditável](CTWA-NATIVE-HANDOFF-20261027.md): PR #10 Draft, gateway nativo, eventos, recibos, segurança e script offline `scripts/validate-ctwa-handoff-local.ps1`. Fonte operacional atual; não usar API batch 415/416.
- [CTWA em produção vigente](CTWA-ROUTING-20261022.md): fluxo comercial Meta Ads na instância 25287 (`evydencia`), envio de PDF via URL `/static/`, reação `🎄`, fallback de menu e painel web `/painel`.
- [Skill CTWA](../.agents/skills/ctwa-ads-funnel/SKILL.md): roteiro local para Antigravity.

- [README](../README.md): arquitetura, responsabilidades, instâncias e posição atual.
- [AGENTS.md](../AGENTS.md): único contrato de instruções para Antigravity.
- [START-HERE](START-HERE.md): um script e só validações locais.
- [PRODUCTION-PREFLIGHT-20261022](PRODUCTION-PREFLIGHT-20261022.md): achados da revisão do código e gates operacionais.
- [RELAY-ROLLOUT-GATES](RELAY-ROLLOUT-GATES.md): mudança de webhook segura SEM mover status-engine de 8050.
- [deploy/README](../deploy/README.md): blueprint de deploy e rede privada.
- [CRM API 416](CRM-API-416-PLAYBOOK.md): endpoint oficial existente e respostas.
- [MULTIINSTANCE](MULTIINSTANCE-ARCHITECTURE.md): registro de IDs GO e separação de tokens.
- [SECURITY](SECURITY.md): cuidados com dados e rollback.

## Skills ativas por tópico
- `relay-local-preflight`: validar script/CI/Compose local, SEM VPS.
- `outbox-production-audit`: integridade fsync, ACK, crashes e supressões.
- `evolution-go-v072-adapter`: contrato REST da versão instalada.
- `crm-readonly-contract`: agendamento 416/consulta GET.
- `multi-instance-contract`: UUID, dedup por lead e tokens separados.

## ARQUIVOS HISTÓRICOS (preservar por trilha forense, NÃO seguir como plano atual)
- `FORENSIC-AUDIT-2026-10-21.md` e `VPS-AUDIT.md` — inventário inicial reportado, antes do canário.
- `AUDIT-CANARY-AND-MULTIINSTANCE-2026-10-21.md`, `HANDOFF-ANTIGRAVITY-CANARY.md` — análise de canário e PRs anteriores.
- `FORENSIC-PR4-20261022.md` — falhas do worker antes da consolidação de hotfix.
- `IMPLEMENTATION-NEXT.md`, `PROMPT-ANTIGRAVITY-NEXT.md`, `PROMPT-ANTIGRAVITY-PR4-GUARDS.md` — instruções concluídas de fases passadas.
- `RESEARCH-AND-CONTRACTS.md`, `THIRD-PARTY-RESEARCH.md`, `PERFORMANCE-POLICY.md`, `WORKFLOWS.md` — pesquisa/escopo histórico; validar sugestões com código atual antes de executar.

**Não usar tarefas antigas para reinstalar serviços, repetir o canário ou reconstruir módulos.** A implementação nova de relay não altera o status engine até autorização separada.
