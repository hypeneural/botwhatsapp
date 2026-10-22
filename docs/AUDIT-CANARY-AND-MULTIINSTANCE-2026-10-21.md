# Auditoria forense pós-canário — BotWhatsApp / CRM / Evolution Go

**Data de análise:** 2026-10-21, revisão do relatório fornecido pelo gestor.
**Escopo de escrita:** somente `hypeneural/botwhatsapp`.
**Método:** comparação com GitHub remoto, código da tag Evolution Go 0.7.2 e CRM `main`; nenhuma conexão operacional à VPS, Windows local ou bancos.

## 1. Evidência relatada vs evidência verificável

O relatório do AntiGravity **AFIRMA**:
- Texto enviado, ID de mensagem retornado pela Evolution Go;
- Áudio OGG/Opus `PTT=true`, ID de mensagem retornado;
- CRM confirmou ID de agendamento de campanha 416, instância de saída CRM #1;
- Replay sintético do mesmo webhook ignorado;
- Segunda mensagem sintética do mesmo contato/dia gera `schedule_natal_crm=suppressed`;
- URL de webhook da instância `anderson` foi restaurada via formulário Chrome.

**NÃO foi confirmado por esta auditoria externa:**
- Recebimento natural da mensagem na sessão WhatsApp acionando a sequência inteira sem injeção sintética;
- Recibo `delivered` ou `read` do texto/áudio nem aceitação da primeira mensagem da proposta pelo número CRM;
- Conjunto exato de source files do worker usado no Windows;
- Persistência efetiva do rollback no banco/manager ou fluxo de eventos retomado no status engine;
- Correspondência entre o POST realmente executado e a rota do CRM, porque o relatório escreveu um caminho divergente;
- Resistência do worker a crash/timeout/concorrência e isolamento de mais de uma instância.

**Observação de rastreabilidade:** no GitHub remoto consultado, `main=7e2e3bd`, PR #1 do CRM 416 ainda aberta, sem os commits locais `16514e4`, `10348a6`, `ae40f97`. O log local não foi disponibilizado em arquivo montado; por isso a auditoria do worker *real* está pendente do push auditável (sem segredos/PII).

## 2. Achados por severidade

### P0 — Contrato de CRM divergente nos logs
O relatório cita `POST /api/campaigns/schedule`. O código vigente em `routes/api.php` e Scribe expõe `POST /api/campaigns/schedule/execute`. O adaptador da PR #1 utiliza a rota comprovada. Verificar URL HTTP efetiva em log local *sanitizado*, sem repetir o agendamento. Confirmar resposta com `id` na raiz (Laravel `JsonResource::withoutWrapping`), `campaign=416` e instância esperada.

### P0 — Status `confirmed` não equivale a entrega
Os IDs de texto/áudio na Evolution Go normalmente indicam aceite/identificação da mensagem pelo cliente/API. Sem receipts do WhatsApp, não declarar entrega/leitura. ID do CRM prova criação de agendamento (dependendo da resposta), não envio da proposta nem aprovação da política comercial. Estados separados: `api_accepted`, `crm_scheduled`, `provider_accepted`, `delivered`, `read`, `suppressed`, `uncertain`.

### P0 — Webhook da instância compartilhada com Status
A instância `anderson` já enviava eventos ao `whatsapp-status-status-engine-1`. Substituir temporariamente seu destino cria janela de perda de atualizações do motor de status. Uma imagem do campo na UI NÃO comprova persistência no backend ou retomada do consumidor. Validar read-only `/instance/all`, mostrando só bool de igualdade do webhook ao destino anterior (não imprimir tokens), e confirmar atividade saudável do status-engine sem disparos. Não alternar novamente o webhook para testar.

### P0 — Código canário ainda local
Publicar commits do worker em branch separada após revisar `git show --stat`, arquivos staged, credenciais, áudios/fotos privados, `testdata`, `data/spool`, scripts .cmd locais e logs. Gerar PR, não force-push `main`.

### P0 — Escopo do endereço Tailscale
O endereço Tailscale relatado é privado; entretanto `0.0.0.0:8788` no Windows escuta TODAS as interfaces. Para repetir canário, fazer bind no IP Tailscale real do host OU limitar firewall à interface/rede Tailscale. Não publicar rota de webhook no Caddy sem revisão. Segredo em URL tem risco de aparecer em access logs; middleware deve tratar URL como dado sensível. `instanceToken` dentro do JSON **não é assinatura**.

### P1 — Dedup de evento ≠ dedup de lead
O journal `(instanceUUID,eventType,messageID)` evita replay do mesmo payload; **não** evita novo ID para mesmo contato. Claim comercial `(tenant,telefone_nacional,campaign,janela-local)` deve ser compartilhado entre todas as instâncias. A janela observada do canário é um dia; se a decisão comercial mudar, usar configuração versionada; nunca derivar data pela timezone UTC silenciosamente.

### P1 — Persistência e perda de eventos
Se o processo retornar HTTP 202 antes de journal/outbox durável, crash pode perder o evento. Se houver `attempt_started` no journal na hora do crash, replay sem reconciliação duplica mensagem ou campanha. Verificar `fsync`, exclusividade entre workers, transição atomizada, locks expiráveis e resultado `uncertain`, sem replay automático.

### P1 — Opt-out e atendimento humano
`core.Select` do baseline retorna `Ignore` para `PARAR`, portanto o receptor de preview responde 204 sem persistir supressão. Não tratar como proteção completa. O worker local precisa provar opt-out DURÁVEL, pausa humana e precedência desses estados antes do planejamento e execução de ações. Sem prova, manter fluxo comercial OFF.

### P1 — Retenção de journal e saúde do disco
Inventário anterior apontou host com disco ~78% usado. Um arquivo permanente por evento não pode crescer para sempre. Planejar compactação SQLite/WAL ou retenção segura e limites (capacidade, idade, métricas), sem varredura O(N) a cada webhook. Monitorar fila, `uncertain`, últimos ACKs e capacidade, com PII mascarada.

## 3. Separação correta dos números
- **48 3220-0440**: instância ads/conversa de anúncio na Evolution Go; usuário relata que o canário partiu da instância `anderson`.
- **48 99642-5287**: futura instância de saudação Evolution Go; não deduzir que foi pareada.
- **48 9642-5287** (aparece no relatório como saída CRM `4896425287`): número de saída **distinto** do `48 99642-5287` da saudação. Não confundir! São canais de transporte diferentes com provedores/configurações potencialmente diferentes.
- CRM deve receber `instance:1` SOMENTE após leitura operacional que confirme ser remetente desejado; nunca inferir número CRM por UUID GO.

## 4. Teste suficiente antes de homologação plena
1. Sem enviar novas mensagens, obter resposta do CRM para o agendamento já relatado por `GET /api/campaigns/schedule/search` filtrado por campanha/telefone autorizado. Conferir ID, instância e status; ausência na busca não prova que não existe.
2. Ler estado atual Evolution GO e comparar destino do webhook com o status engine original, sem modificar nada.
3. Validar que o motor de status voltou a consumir eventos e a instância segue conectada, sem mudar seu webhook.
4. Auditar código worker local postado no GitHub: durabilidade, concorrência e supressões.
5. Canário de entrada NATURAL somente numa próxima janela expressamente aprovada e com fan-out seguro, sem substituir destino do motor de status.

## 5. Arquitetura futura
Ver `docs/MULTIINSTANCE-ARCHITECTURE.md`: um serviço, registros por instância, clientes Evolution separados, journal compartilhado por marca; bot de saudação em instância separada; envio da proposta sempre pela instância CRM configurada.

## Fontes
- Código oficial GO 0.7.2: `pkg/routes/routes.go`, `pkg/events/webhook/webhook_producer.go`, `pkg/sendMessage/service/send_service.go`.
- CRM main: `routes/api.php`, `CampaignScheduledController.php`, `CampaignScheduledResource.php`, `AppServiceProvider.php`.
- Evidência operacional da execução: exclusivamente o relatório/texto enviado pelo gestor, não o banco/Windows.
