# Reuso criterioso de projetos GitHub

## Prioridade 1 — skill Evolution Go 0.7.2
Fonte: https://github.com/6justgotme/evolution-go-skill/tree/90d23690f7fa79bc252ee006675b4c7e50e0a176
Licença: MIT. Benefício para Flash Lite: nomes dos handlers, payloads e separação `MESSAGE` vs `Message`, `/send/text` e `/send/media`. A skill original tem cobertura muito mais ampla (múltiplos transports, QR, instâncias, grupos, APIs interativas) e consumiria atenção sem necessidade.
**Decisão:** manter uma skill CURADA e versionada em `.agents/skills/evolution-go-v072-adapter/`, com recursos essenciais e links para origem. Não venderizar todas as referências sem necessidade. Para responder dúvida nova, ler o arquivo original relevante e comparar com o código da TAG e, sobretudo, com o patch instalado.

## Prioridade 2 — bridge Go com proteção de entrega incerta
Fonte: https://github.com/estevao-tarifa/evo-chatwoot-bridge-go/tree/7ba5bb1a69734c8f486e7515bd1b3b9b3f82571c/backend/internal
Licença: MIT. Arquivos inspiradores: `evolution/client.go`, `evolution/messages.go`, `dedup/dedup.go`, `dedup/store.go`, testes de cliente.
**Reuso:** padrões de `http.Client` com limite, `apikey`, confirmação via `data.Info.ID`, mensagens incertas sem retry e chave por instância/evento/ID. **Não importar o backend todo**, pois adicionaria Chatwoot, interfaces e MySQL, além do módulo declarar Go 1.27.

## Prioridade 3 — cliente Python do EvoNexus
https://github.com/evolution-foundation/evo-nexus — contém exemplo Python de chamadas diretas, mas nomes de campos divergem em alguns trechos (`instanceName` e `webhook` vs `name` e `webhookUrl`). Sua licença possui condições adicionais; não copiar código. Serve somente para inspiração de testes/smoke calls depois de revisar o contrato.

## Descartados
- `ericorenato/vibestack-openclaw` bridge usa dedup em memória e dependências de IA: não durável.
- `evolution-foundation/evo-bot-runtime` é arquitetura mais pesada; fora do escopo.
- APIs alternativas como GOWA/evolution Node exigiriam outra stack; não utilizar.

## Proveniência
A implementação atual em `internal/` é código próprio para este projeto, sem cópia textual de fontes externas. Se no futuro importar/colar um arquivo MIT, preservar LICENSE e copyright correspondentes e documentar exatamente arquivos/versões no inventário de licenças.

**Recomendação:** evoluir por testes locais e implementação por contrato, não copiar boilerplate genérico nem trazer motores completos.
