# Antigravity Flash Lite — tarefa local atual

O projeto já possui worker durável e contratos CRM/GO. **Não reimplementar**. Leia `AGENTS.md`, `docs/PRODUCTION-PREFLIGHT-20261022.md`, `docs/RELAY-ROLLOUT-GATES.md`.

1. Conferir `git status -sb` e o commit; preservar alterações locais sem reset destrutivo.
2. Testar **somente o clone local** com `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-local.ps1`.
3. Registrar resultado de cada pacote e build, versão Go Windows e validação de Compose local. CI oficial usa Go 1.27.3.
4. Se algum teste falhar, corrigir o arquivo específico e executar a suite completa **uma vez** na conclusão.
5. Conferir que `BOT_ENABLED=false`, `BOT_OUTBOUND_ENABLED=false`, `BOT_OPERATIONAL_AUTHORIZATION=false`, `RELAY_ENABLED=false`; nenhuma credencial no Git.
6. Não fazer SSH, restart, alteração de webhook ou POST real para CRM/GO. O canário comercial já foi realizado, não repetir.

**Ainda falta para produção:** verificar o código de consumidor status-engine (idempotência), tamanhos reais de webhook e quota, aprovar mídia, validar instância de saudação e autorizar corte para novo relay em janela operacional. Nenhuma dessas ações ocorre nesta tarefa local.

Relatórios anteriores (pré-worker/pre-relay) estão preservados para rastreabilidade; o índice `docs/INDEX.md` diferencia sua validade.
