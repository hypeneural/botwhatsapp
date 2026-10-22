# Auditoria forense: Antigravity PR #4, CRM 416 e Evolution Go 0.7.2

**Auditoria de código:** `feat/consolidate-multiinstance-outbox` em `98d964a8ca055a34f9a420262f1ce3d028d613f7` (PR #4).
**Correções:** `fix/ingress-outbox-production-guards-20261022`, PR dependente da #4.
**Limite:** leitura estática GitHub e execuções Actions; não foi realizado novo acesso SSH, consulta live CRM, webhook ou disparo de teste. Logs de canário são RELATO do Antigravity; não reabrir essa campanha.

## 1. Proveniência e stack
- `main` permaneceu no commit `7e2e3bd` ao iniciar. PRs #1, #2, #3, #4 continuam abertas; não efetuar merges paralelos/rebase indiscriminado.
- PR #3 contém worker do canário, PR #4 consolida os três ramos e adiciona registry/retention. **PR #4 é o alvo de integração**, não abrir quatro releases independentes para o mesmo conjunto.
- Host relatado: Ubuntu 24.04.4, Docker 29.7.2/Compose 5.5.0, 11 GiB RAM, disco ~78%; Evolution Go 0.7.2 customizada com WebP, Postgres 15, Caddy, status-engine e SmartDialer/Asterisk no mesmo host. Não reiniciar/atualizar esses serviços por causa do bot.
- O relatório AntiGravity afirma que no DB da Evolution Go `anderson` estava conectada e o webhook voltou ao status-engine, que estava healthy. Isso confirma a configuração relatada, mas **não demonstra o recebimento contínuo de todos os eventos**.
- CRM: a rota real do Laravel é `POST /api/campaigns/schedule/execute` (não `/api/campaigns/schedule`). Campanha padrão `416`, SEM NOME, `contacts` em DDD+número nacional, instância numérica de saída. Relato do canário: agendamento `28531`, saída `instance:1`; GET posterior timeout, portanto reconciliação **inconclusiva**. NÃO executar outro POST.
- `48 3220-0440` = Ads na GO; `48 99642-5287` = futura saudação GO não pareada; o telefone associado à saída CRM `4896425287` relatado é outro número, **não é a saudação**.

## 2. Principais P0 observados no código da PR #4

| Achado | Prova no código | Consequência |
| --- | --- | --- |
| ACK antes do outbox | `internal/ingress/server.go` usava `go OnDecision(...)` após escrever spool e responder 202 | Evento reconhecido mas worker poderia não persistir tarefas se o processo encerrasse |
| Dedup antes de tarefas | `Worker.IngestDecision` usava `CheckAndRecordEvent` antes de `EnqueueTask` | Falha na gravação deixava evento permanentemente marcado como concluído |
| Um único token GO | `cmd/botwhatsapp/main.go` inicializava um único `evolution.Client` Ads | A saudação poderia sair pelo token da instância errada |
| Áudio/imagem inventados | `NewWorker` aplicava `MOCK_AUDIO_FIXTURE_BYTES`/`MOCK_IMAGE_FIXTURE_BYTES` | POST inválido com mídia fictícia e status incerto |
| Opt-out descartado | `core.Select` retornava `Ignore` e ingress parava sem invocar persistência | O pedido PARAR não atualizava a supressão |
| Corrupção de contato | `Store.GetContact` retornava estado limpo quando JSON inválido | Perda de informação de opt-out/humano após erro no disco |
| Pause tardio | Opção 5 pausava somente como último efeito do worker | Tarefas comerciais pendentes podiam começar antes da pausa |
| Cooldown somente no ingresso | Múltiplas saudações podiam ser enfileiradas antes da primeira entrega | Imagens repetidas em sequência |
| Poll excessivo | `main` lia todos os arquivos a cada 500ms | I/O desnecessário na VPS compartilhada |
| Config perigosa | `main` tinha triggers genéricos e bind default `0.0.0.0` | Disparo por texto impreciso; superfície de rede maior |

## 3. Correções implementadas nesta branch
1. OnDecision **síncrono** com retorno de erro HTTP 503; 202 só após outbox persistido. Opt-out também passa pelo callback e deve ser gravado.
2. `Store.CommitEventTasks`: prepara tarefas em arquivos temporários com fsync, publica lote sob mutex e persiste marcador de evento **por último**, rollback em falha; IDs determinísticos hash de UUID/evento/ID/etapa. É confiável dentro de **um único processo com volume local**, não uma transação distribuída para múltiplos containers.
3. Registro GO com clientes distintos por UUID, sem fallback de token para outro número; uma instância pode operar isolada. Falha de registry impede boot, não há fallback para legacy.
4. Mídias só quando configuradas explicitamente; sem bytes fictícios. Antes de saída: validação OGG Opus/JPEG/PNG no boot, limite 8 MiB. Falta mídia → etapa suprimida sem chamada de rede; uma falha no áudio não bloqueia solicitação CRM.
5. Contato corrompido → erro fail-closed; ProcessTask bloqueia opt-out/pausa humana, revalida cooldown imediatamente antes da saudação; opção 5 registra pausa no ingresso, com exceção somente para o texto de confirmação humana.
6. Ingestão notifica worker por canal limitado; fallback de scan 60s + manutenção de tarefas concluídas a cada 6h. Sem scans a cada 500ms.
7. Exigidos `BOT_ENABLED=true`, `BOT_OUTBOUND_ENABLED=true`, `BOT_OPERATIONAL_AUTHORIZATION=true`; URLs e tokens explícitos, frase Ads apenas via `BOT_ADS_TRIGGERS`. Bind default apenas `127.0.0.1:8788`.

## 4. Limites que AINDA bloqueiam produção
- **Webhook fan-out não implementado**: `anderson` continua usando o status-engine; não substituir sua URL. Projetar relay DURÁVEL, ou outra integração autorizada que preserve o consumidor atual, em PR própria.
- **Uma única réplica** do bot por `BOT_DATA_DIR`. Mutex Go não sincroniza outros processos/nós; `ClaimLeadShared` vale apenas no volume local e não é garantia entre réplicas. Não habilitar autoscaling.
- `PruneCompleted` só cobre tarefas confirmadas/suprimidas. **Spool de ingresso, dedup events, lead claims, contatos e uncertain** exigem política de retenção/quota própria e limpeza cuidadosa. Não apagar supressões nem uncertainties sem reconciliação.
- Falta observar `delivered/read` na API da GO e mensagens da campanha após `scheduled` no CRM. ID retornado por API não prova recebimento pelo cliente.
- O bot de saudação ainda depende da instância GO pareada e imagem/caption aprovadas; não inferir UUID/token da instância CRM.
- Falta prova de continuidade do status-engine após o rollback apenas por consulta DB/status healthy.
- O CRM GET `/api/campaigns/schedule/search` foi inconclusivo por timeout externo. Não repetir POST; usar consulta interna autorizada ou registros de campanha já existentes.

## 5. Testes
- Na PR #4, Actions reportou PASS para `go test ./...` em 7 pacotes.
- Nesta branch foram adicionados casos: ACK durável/erro, opt-out, media ausente, desvio de instância, corrupção de política, commit de lote rollback/dedup, duas saudações pendentes e pausa humana.
- Confirmar no Actions do HEAD e rodar `go test -count=1 ./...` UMA vez após integração; se falhar, corrigir pacote afetado. Não usar canário real como teste de regressão.

## 6. Política de release
- Integrar a PR #4 apenas após revisão final da árvore e confirmação das dependências; integrar a PR desta auditoria em seguida, sem rebase destrutivo sobre main.
- Para homologação em produção, exigir relay seguro, credenciais/canais provisionados, mídia aprovada, plano de rollback independente, proteção das demais aplicações da VPS e destinatário teste autorizado.
- **Nenhum deploy, POST CRM ou WhatsApp real foi feito nesta auditoria.**
