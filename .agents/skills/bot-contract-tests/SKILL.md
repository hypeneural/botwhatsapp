---
name: bot-contract-tests
description: Executa testes rápidos, determinísticos e sem rede real dos contratos Evolution Go, CRM, roteamento e deduplicação. Use sempre que editar Go ou config do bot.
---
# Teste local dirigido
1. Executar `go test ./...` no clone local (Go >= 1.23) — sem acesso à VPS.
2. Verificar casos: anúncio estrito, fallback, mensagem própria, LID, grupo, campanha 415/416, telefone CRM sem DDI, webhook autenticado por segredo e UUID, evento duplicado não regravado, menu opção 5, cooldown expirado.
3. Clientes `internal/evolution` e `internal/crm` são testados APENAS com `httptest` local. Não ativar flags de envio real.
4. Quando teste falhar: corrigir a causa no pacote alterado, repetir pacote e depois executar `go test ./...` uma vez; não repetir suites de terceiros.
5. Verificar GitHub Actions `go-test.yml` somente após alterações, sem demandar deploy. Relatar teste executado, teste não executado e motivo.
