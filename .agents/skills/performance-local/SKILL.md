---
name: performance-local
description: Otimizar desempenho do bot Go de duas instâncias Evolution Go com benchmarks sintéticos e limites de memória, sem testar carga na VPS compartilhada.
---
# Performance local
1. Ler `docs/PERFORMANCE-POLICY.md` e `AGENTS.md`.
2. Testar com fixtures sintéticas no clone, primeiro `go test ./...`, depois benchmark específico `go test -run '^$' -bench . ./internal/core` se houver benchmarks.
3. Inspecionar alocações e tamanho de JSON no envio de mídia; limitar base64 a 8 MiB atual, considerar multipart streaming quando houver necessidade confirmada.
4. Nunca usar clientes reais ou gerar tráfego na Evolution GO, Caddy, CRM, Asterisk, status engine ou outras stacks como parte de performance. Benchmark em VPS só com autorização adicional e janela de manutenção.
5. Produzir tabela de p50/p95, memória e limite de disco **somente com medidas reais**, sem estimativas apresentadas como fatos.
