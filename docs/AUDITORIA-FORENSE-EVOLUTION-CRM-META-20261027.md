# Auditoria forense cruzada — Evolution Go 0.7.2 × BotWhatsApp × CRM × Meta Ads
**Data:** 27/10/2026 | **Escopo:** código/contratos reais, sem leitura de VPS ou clientes LIVE | **Vínculos:** Bot issue #11, PR #12 (base PR #10), CRM issue #442.

## 1. Evidência examinada e limites
- **Evolution Go tag `0.7.2`**: `pkg/whatsmeow/service/referral.go`, `referral_test.go`, `whatsmeow.go`, `pkg/message/model/message_model.go` e `pkg/message/repository/message_repository.go`.
- **Bot PR #10**: `internal/ingress/audit.go`, `internal/dashboard/events.go`, `internal/crm/native.go`, `internal/outbox/types.go` e seus testes. #10 está Draft, e as mudanças de sua branch **não equivalem ao binário publicado**. PR #12 é stacked, sem runtime em produção.
- **CRM main**: `routes/extension-api.php`, `ProposalScheduleController`, `CustomerContextQuery`, `Customer360Query`, `NatalCommercialConversionPolicy`, `NatalValidOrdersQuery`, `CampaignScheduledController`, `GetExistingProposalStatus`.
- **Meta repo**: P24 PRs e skill `evyd-whatsapp-crm-measurement`; planejamento e snapshots históricos NÃO são um conector de Insights vivo.
- **Ausentes:** imagem/digest e config do contêiner de produção, amostra sanitizada do primeiro webhook CTWA real, registros LIVE da Evolution/status-engine, CRM LIVE com contratos de campanha e financeiro, leitura atual Insights da conta oficial. NÃO afirmar reconciliação REAL ou valores observados.

## 2. Descoberta central na Evolution Go 0.7.2
`extractReferralFromMessage` lê `contextInfo.externalAdReply` de extendedText, image, audio, document e video; faz `protojson.Marshal`. O teste oficial usa `sourceType: ad`, `sourceID`, `ctwaClid`, `showAdAttribution`. Em `events.Message`, o servidor acrescenta o JSON a **`data.referral`** do evento `Message` (não exige um evento separado). Com `DATABASE_SAVE_MESSAGES=true`, grava `Message{Referral: referral}` no Postgres, inclusive conflict update quando há metadata. Webhook, queues, NATS e WebSocket não devem ser assumidos configurados no ambiente real apenas porque estão disponíveis no código.
- `sourceID` é a **referência da origem**, que pode ser anúncio/post. Apenas cruzar com o ad_id REAL Meta confirma a correspondência. `ctwaClid` é click ID, jamais equivalente a `sourceID` ou `ad_id`.
- O metadado costuma estar no **primeiro contato CTWA**, não em todas as mensagens posteriores; ausência posterior NÃO descarta origem anterior.
- Um payload com `sourceType=ad` sem ID é sinal de origem, sem anúncio específico.
- `externalAdReply` presente ou texto exato são *hints*, não comprovação de qual criativo e não autenticação criptográfica. A autenticação depende do webhook/relé confiável.
- **Não ligar `WADEBUG` nem dump de webhook bruto**, pois podem expor telefone, mídia e click ID. Inventário e consultas sanitizados.

## 3. Falhas/restrições verificadas no BotWhatsApp
- Na PR #10 `/api/dashboard/events` deriva 50/150/500 linhas mais recentes **dos journals locais**; ordena arquivos inicialmente por mtime, depois eventos por `received_at`. Um restore que altera mtime afeta seleção da janela. **Esses contadores NÃO têm cobertura global nem representam um período de Insights**.
- `audit.go` tinha `has_ad_context`/trigger e nenhuma referência `sourceID`: impossibilidade de join por anúncio. Fase #12 preserva `ad_source_id` numérico + `ad_evidence`, apenas para `sourceType=ad`; não persiste `ctwaClid`, URL, thumbnail ou payload bruto. PR #12 NÃO agrega ainda Meta nem venda.
- Inbound `received` != resposta Evolution aceita; resposta Evolution aceita != entregue/lida; CRM `dispatch_uuid` != entrega da régua; pedido != pagamento. Recibo temporal da última tarefa nunca esconder falhas novas.
- Registros antigos não ganham metadados retroativamente. O painel não deve inferir 0440→5287 nem converter `@lid` em número.
- Bot desligado/desconectado => journal do bot pode ficar vazio. A Evolution, seu storage opcional, o status-engine/relay ou o CRM podem ter registros independentes. Sem cobertura de alguma fonte, a classificação é `NOT_OBSERVED`, nunca `BOT_WAS_OFF`.

## 4. CRM: distinguir quatro verdades comerciais
**API nativa existente:**
- `GET /api/v1/extension/customer-context?phone=...`, ability `extension:customer-read`: cadastro e sinal `natal_2026.purchased` para UM telefone autenticado.
- `GET /api/v1/extension/customer-360?phone=...`: histórico de pedidos e status financeiros com mesma classe de permissão; NÃO chamar 1 vez por linha em tela de 500 contatos.
- `POST /api/v1/extension/proposals/schedule`, ability `extension:proposal-write`: `dispatch_uuid`, apenas agenda proposta, sem provar entrega.
- `GET /api/campaigns/schedule/search`: consulta existente por campanha/telefone, mas `CampaignScheduledService::search` executa `get()` sem limite/paginação; NÃO a usar para sincronizar dashboard inteiro, nem ignorar os agendamentos/dispatches históricos.
**Métricas diferentes:**
- `customer_converted`: política `NatalCommercialConversionPolicy` interpreta status de progresso ou parcela positiva, e pode sinalizar `review_reason`.
- `valid_natal_order`: `NatalValidOrdersQuery` tem regras temporais, comerciais, estornos e cancelamentos mais estritas, com `asOf`.
- `contracted_brl`: valor líquido contratado de pedidos válidos, NÃO receita recebida.
- `received_brl`: baixa confirmada de parcelas conforme razão financeira e corte temporal. Subtrair estornos conforme regras canônicas; não usar float para dinheiro.
- Não computar `conv` por haver cliente cadastrado ou proposta agendada; se cliente já comprou antes do ingresso CTWA, não atribuir a compra àquele anúncio.

## 5. Métricas Meta e reconciliação
Fonte oficial: Insights do Ad Account **`act_1627262698736055`**, `level=ad`, `time_increment=1`, campos explícitos `ad_id,campaign_id,adset_id,date_start,date_stop,spend,impressions,clicks,inline_link_clicks,actions`. Conferir action_type específico de conversa disponível no objeto atual, janela `action_attribution_windows` e `action_report_time`; nunca confundir `clicks` com clique no WhatsApp.
- Dinheiro em **centavos inteiros** ou DECIMAL exato; `spend` da Meta vem em BRL decimal, **não** assumir já em centavos.
- `meta_conversations`: ações atribuídas pela Meta; não representam necessariamente o mesmo universo/período que contatos únicos na Evolution.
- `observed_ctwa_contacts`: contatos únicos com evidência de origem; limitado pela cobertura do receptor e disponibilidade da primeira mensagem.
- `crm_assisted_sales`: compra/pedido depois de interação/campanha CRM identificada (ASSOCIAÇÃO temporal, não causalidade).
- `ctwa_matched_sales`: pedido válido posterior a lead com `sourceID` verificado contra ad_id, dentro de janela definida, com política de deduplicação e compradores pré-existentes excluídos desse crédito.
- Métricas gerenciais: `CPL_observado=spend/observed_ctwa_contacts`, `custo_pedido_associado=spend/ctwa_matched_valid_orders`, `ROAS_contratado_indicativo=valor_contratado_associado/spend`, `retorno_caixa_indicativo=valor_recebido_associado/spend`. Razão com denominador 0 ou coverage parcial é `null`, nunca inventar zero.
- Guardar snapshots com `as_of`, origem, período, conta, moeda, versão, status/frescura, janelas e cobertura. Meta gasta no dia do anúncio, venda ocorre em outro dia; usar visão por coorte de entrada e visão por data da venda, sem comparação ingênua das linhas do mesmo dia.

## 6. Exemplo de timeline do operador
1. 10:00 Meta registra clique agregado num anúncio.
2. 10:01 Evolution `Message` entrega evento com `referral.sourceType=ad` e candidato `sourceID`; identificar lead único por telefone verificado e ID da mensagem.
3. 10:01 Bot ingress? se não houver evento, checar **cobertura** bot/relay. Não supor falha.
4. 10:02 Bot responde e aceita/erro incerto (não prova entregue).
5. 10:03 CRM agenda `dispatch_uuid`; consultar etapas efetivamente `SENT/RECEIVED/READ` pelos provedores.
6. D+N CRM registra pedido Natal com status e/ou pagamento: oferecer quatro contadores distintos (lead acionado, convertido, pedido válido, caixa recebido), com evidência de qual anúncio/campanha e janelas.
7. Marcador `comprou antes`, `já foi exposto antes`, `duas origens`, `múltiplas propostas` sem sobrescrever a primeira evidência; primeira e última interações exibíveis, crédito atribuído por regra v1 separada.

## 7. Gates somente leitura para Antigravity / Codex
**Antigravity em Evolution/VPS (somente leitura):**
A) Inventariar contêiner que executa a Evolution (nome, image tag+digest e versão real, sem `docker inspect` imprimindo env ou segredos). B) Checar boolean `DATABASE_SAVE_MESSAGES`, subscrição do evento `Message`, status-engine e rota do relay por instância 5287/legado 0440, volumes e datas de cobertura. C) Para um ID de mensagem TESTE autorizado, verificar **apenas** presença/nomes/tipos `data.referral.sourceType/sourceID` e se há evento no journal e/ou messages do banco; mascarar telefone e não registrar `ctwaClid`. D) Conferir `sourceID` com leitura dos Ad IDs ativos da conta oficial. E) Contar eventos distintos por ID na janela e falhas/uncertain, com proteção de backlog e sem replay.
**Codex no CRM:** confirmar consulta lote/paginada com `cursor` por evento, permissão read-only dedicada, tenant e retroatividade, joins de `proposal_dispatches`, `campaign_scheduled_contacts`, `scheduled_contact_stage`, providers e orders/installments. Não usar a API individual 360 em N+1. Resolver divergência de definições de conversão ANTES de publicar KPI/ROAS.
**Meta Antigravity:** validar READ-ONLY `level=ad` e action type, fuso, campanha e Ads ID reais. Diferenciar zero observado e indisponível/sem permissão.
**Go/No-Go:** não enviar contato, não editar campanhas, não trocar webhook ou reiniciar Evolution/status-engine, não habilitar `DATABASE_SAVE_MESSAGES` automaticamente (custo/PII), não implantar #12 enquanto #10 ainda se move. Testes unitários sanitizados antes de homologação por canário único explícito.

## 8. Fontes
- https://github.com/evolution-foundation/evolution-go/blob/0.7.2/pkg/whatsmeow/service/referral.go
- https://github.com/evolution-foundation/evolution-go/blob/0.7.2/pkg/whatsmeow/service/referral_test.go
- https://github.com/evolution-foundation/evolution-go/blob/0.7.2/pkg/whatsmeow/service/whatsmeow.go
- https://github.com/evolution-foundation/evolution-go/blob/0.7.2/pkg/message/repository/message_repository.go
- https://www.postman.com/meta/facebook-marketing-api/request/5wdl62t/attributionsetting
- https://github.com/hypeneural/evydencia-crm/blob/main/app/Services/Natal/NatalCommercialConversionPolicy.php
- https://github.com/hypeneural/evydencia-crm/blob/main/app/Services/Reports/NatalValidOrdersQuery.php
