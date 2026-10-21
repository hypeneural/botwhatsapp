# Performance e confiabilidade — duas instâncias GO 0.7.2

## Situação e objetivos
Inventário remoto relatado: 11 GiB RAM, ~9 GiB disponíveis, 44 GB livres com disco já em 78%. Outros serviços críticos de voz, status, video e Caddy competem pelo mesmo host. NÃO rodar benchmarks de carga na VPS em produção sem autorização. Números de capacidade ou throughput **não foram medidos**.

### O que já foi implementado
- Go stdlib apenas, sem ORM/LLM/framework: build multi-stage CGO_ENABLED=0 e binário pequeno.
- Webhook HTTP usa limite de entrada 64 KiB, `ReadHeaderTimeout` e `ReadTimeout`.
- Handler não faz chamadas HTTP externas; grava evento preview de forma atômica via link+fsync e devolve 202.
- Dedup por SHA-256 do UUID, evento e messageID, lookup por nome de arquivo (sem varrer banco).
- Somente dados mínimos (decisão, remetente necessário, timestamp); não persiste `instanceToken` ou payload completo. Volume privado 0700, arquivo 0600.
- Clientes HTTP possuem timeouts (GO 12s, CRM 15s) e não repetem POST ambíguo. **Ainda não são executados**.
- CI Go tests isolados, sem conectividade real.

### Limites e tradeoffs do preview
Cada evento gera um arquivo imutável; pode acumular arquivos e PII indefinidamente e não gerencia encerramento/recuperação de etapas. NÃO usar este spool como outbox de produção sem rotacionamento/retention (proposta: 30–90 dias conforme política do gestor), purga segura e testes de crash.
`SendMedia` em base64 lê bytes na memória (limite local 8 MiB), produz payload ~33% maior; para arquivos mais volumosos, estudar multipart streaming e limites reais do GO. Não encaminhar mídia antes de validação de tamanho/MIME/codec.
Restringir concorrência do futuro worker (inicialmente 1–2 operações por instância), evitar varredura de todos os arquivos concluídos por loop e implementar backpressure/rejeição quando fila ou disco estiverem cheios.
Fazer healthz/readiness por componente; disponibilidade do servidor preview não prova conectividade WhatsApp ou readiness CRM.

### Métricas de aceitação (NÃO realizadas)
- Benchmark local de parsing e persistência com dados sintéticos e 10/100/1000 mensagens, sem rede real.
- Simular 1000 eventos duplicados: somente um journal por chave; sem payload de token.
- Múltiplos eventos do mesmo número: no máximo um pedido de campanha para janela definida.
- Reinício no momento entre `attempt_started` e resposta: nenhuma segunda tentativa sem reconciliação.
- p50/p95 tempo de ACK e RSS observados, não valores prometidos.
- Rotação de spool, disco 80/85/90% e retenção verificadas antes de deploy; proibir deploy se espaço inadequado.

### Infraestrutura preferida
Criar novo Compose isolado, único replica, volume persistente dedicado, rede interna ligada à GO após mapear alias DNS, sem porta pública `0.0.0.0`. Jamais conectar serviço novo diretamente ao Postgres GO sem justificativa. Usar `127.0.0.1:14000` somente em processo no **HOST**; contêiner novo deve usar DNS de rede autorizado. Não alterar Caddy ou outras stacks neste ciclo.
