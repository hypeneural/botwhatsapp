---
name: evolution-go-v072-adapter
description: Implementa e testa clientes HTTP, parser de webhooks, mensagens de texto, imagem e áudio PTT para Evolution Go 0.7.2 instalada e customizada na Contabo. Use antes de mexer em rotas ou mídia.
---
# Evolution Go 0.7.2 — skill local curada

**Leia primeiro** `AGENTS.md` e `resources/verified-contract.md`. Não usar tutoriais para Evolution Node.

1. Versão pública de referência `0.7.2` de Evolution Foundation; deployment real do inventário: imagem local customizada WebP. Diff do patch e revisão de origem AINDA NÃO VERIFICADOS independentemente; não atualizar API.
2. Fonte técnica especializada externa: `6justgotme/evolution-go-skill` (MIT), commit `90d23690f7fa79bc252ee006675b4c7e50e0a176`. Use apenas referência — snapshot curto em `resources/verified-contract.md`.
3. Base prática de arquitetura: `estevao-tarifa/evo-chatwoot-bridge-go` (MIT), commit `7ba5bb1a69734c8f486e7515bd1b3b9b3f82571c`, módulos `internal/evolution/client.go`, `messages.go` e `internal/dedup`. Não instalar Chatwoot/MySQL; não copiar código integral sem revisão de licença.
4. No projeto atual, modificar somente `internal/core`, `internal/evolution`, `internal/ingress`, `internal/workflow`; testes de contrato em `go test ./...`.
5. Não alterar instâncias ou webhook durante auditoria. Quando houver autorização posterior, usar segredos independentes por rota, UUID/telefone confirmado, ambiente Docker privado e dry-run antes do primeiro send.
6. Se a imagem local divergir, atualizar a referência com evidência exata e testes, não adivinhar.
