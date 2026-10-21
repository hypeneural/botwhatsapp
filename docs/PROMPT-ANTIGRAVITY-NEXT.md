# Prompt pronto — próximo ciclo Antigravity Flash Lite
**OBJETIVO:** dar continuidade SOMENTE no repositório `hypeneural/botwhatsapp` à fase offline. Não interagir com serviços reais.

1. Atualize clone sem perder mudanças locais (`git status`, depois `git pull --ff-only` apenas se working tree limpo).
2. Leia `AGENTS.md`, `docs/FORENSIC-AUDIT-2026-10-21.md`, `docs/IMPLEMENTATION-NEXT.md`, `docs/PERFORMANCE-POLICY.md`.
3. Skills para usar nesta ordem:
  - `bot-offline-simulator`: `go test ./...` e `go run ./cmd/simulate ./testdata/ads.json`, `greeting.json`, `menu-human.json`, `opt-out.json`.
  - `evolution-go-v072-adapter`: verificar código/contratos da TAG oficial `0.7.2`, sem tocar GO instalada.
  - `crm-readonly-contract`: confirmar contratos da API via GitHub leitura, sem alterar CRM.
  - `two-bot-workflows`: implementar **o executor durável que ainda falta**, somente mocks.
  - `bot-contract-tests`: validar apenas código alterado, sem repetição de suites inúteis.
4. O que já existe: `internal/core` parser/roteamento, `internal/workflow` planos e cooldown, `internal/evolution` cliente REST isolado, `internal/crm` cliente REST isolado, `internal/ingress` webhook preview-only com dedup e spool, `cmd/botwhatsapp`. NÃO reescrever esses componentes se testes PASS.
5. O que fazer nesta rodada:
  - Implementar motor pequeno de outbox/journal em diretório privado ou SQLite local, com dedup de evento (UUID + ID mensagem) separada de lead (contato + campanha + janela).
  - Transições atômicas `pending -> attempt_started -> confirmed|uncertain`, sem replay automático de envio incerto. Reinício durante POST deve continuar pendente de reconciliação, não reexecutar.
  - Persistir cooldown de 24h, pausa humana, opt-out, e garantir que apenas worker controlado execute efeitos (nunca no HTTP handler).
  - Na etapa ADS, manter texto/áudio/CRM como efeitos separados. Falha em áudio não bloqueia CRM; CRM só com número válido, campanha 415/416, ID numérico explícito de instância CRM e autorização operacional.
  - Na etapa GREETING, 1 imagem + caption do menu. Não existe imagem real ainda. Construir mocks/fixtures, não inventar arquivos de mídia.
  - Usar um único processo/worker conservador para evitar disputa; sem filas Redis/RabbitMQ, sem instalar outra API.
  - Testar crash/restart, 100 duplicados concorrentes, phone LID sem Pn mapping, timeout, opt-out e handoff. Rodar somente `go test ./...` no fim.
6. **Proibido:** SSH adicional automático, Docker Compose na Contabo, restart, mudar Caddy, alterar webhook, criar ou parear instâncias, enviar WhatsApp ou CRM real, mexer no banco/código do CRM, expor tokens/PII.
7. Finalizar com: resumo do diff, SHA commit, testes realmente PASS, riscos novos, pendências para canário e rollback. Se houver contradições, sinalize e PARE sem inventar.

A próxima etapa NÃO tem como objetivo fazer deploy. O objetivo é entregar engine testada offline pronta para ser parametrizada quando as duas sessões forem provisionadas com autorização.
