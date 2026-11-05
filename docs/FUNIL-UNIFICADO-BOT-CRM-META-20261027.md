# Funil unificado — bot.etijucas | Contrato de implementação (27/10/2026)

**Status:** planejamento executável e sem integração LIVE. Issue #11; esta PR é empilhada sobre #10 (CTWA 5287), NÃO sobre main. Não é autorização de deploy ou de disparos.

## Auditoria cruzada
- Bot PR #10 `internal/dashboard/events.go`: eventos ingressados e tarefas locais; aceita CRM com recibo `dispatch_uuid`, não prova WhatsApp entregue nem pedido vendido. `internal/ingress/audit.go` guarda indício Meta mas não o ID de anúncio; textos-gatilho são heurísticos.
- CRM main `routes/extension-api.php`: GET `customer-context` e `customer-360` consultam um telefone autenticado; POST `proposals/schedule` gera despacho com Sanctum. O dashboard precisa de leitura incremental/autorizada dedicada para campanhas Z-API, SMS, VoIP, pedidos, pagamentos e conversão Natal 2026. Codex é dono desse contrato, em issue separada.
- Repo Meta Ads: scripts e documentação para operar conta `act_1627262698736055`, não endpoint LIVE de Analytics. Capturar Insights por ad ID/data e tipo de clique/conversa, mantendo acesso read-only e metadados de timestamp, janela/fuso, versionamento de API.

## Modelo visual do gestor
Adicionar futuramente uma aba **Visão geral** ao `/painel` já existente, sem reconstruir Histórico e Contatos. Filtros: hoje/7d/30d, campanha, conjunto, anúncio, status e atenção. Cards separados:
1. Meta — gasto em centavos, impressões, cliques especificados, conversas reportadas, custo/conversa.
2. WhatsApp — ingressos distintos, evidência de anúncio comprovada por referral e heurística por texto, respostas aceitas vs entregues/lidas apenas via callback.
3. CRM — proposta agendada, campanha acionada, entrega por canal, Natal 2026 comprado pela política oficial, pagamentos confirmados.
4. Reconciliação — CRM sem ingresso observado no bot, bot sem recibo CRM, resultado uncertain, opt-out e falhas. Timeline com IDs técnicos e origem, acessível só ao gestor autorizado.

## Invariantes
- **Meta click não é mensagem, conversa Meta não é contato único, dispatch CRM não é entrega, pedido criado não é pagamento.** Proibido somar denominadores de universos diferentes.
- Ausência de fonte/partial != zero. Cada snapshot contém `as_of`, `start`, `end`, timezone, version, status (available/partial/unavailable), cobertura completa e erro sanitizado. Janela de consulta [start,end) UTC; exibir BRT.
- Atribuição por anúncio apenas com metadado CTWA validado associado a ad ID, não inferir ID de anúncio de uma mensagem-gatilho, nome de campanha ou texto; `ctwa_clid` não é `ad_id`.
- Deduplicar por tenant + provedor + tipo + event_id; contatos pelo HMAC-SHA256 de E.164 completo **verificado** gerado só no backend restrito (não publicar hash ou telefone). Não derivar telefone a partir de LID. Não enviar PII para Meta ou GitHub.
- CRM é autoridade para `natal_2026.purchased` e pagamento; não atribuir ROAS por anúncio enquanto conversão não tiver prova do vínculo e janela.
- A diferença `CRM com campanha, bot sem evento` **não prova** bot desligado; requer cobertura integral do período de ambas as origens. Tratar casos sem histórico como desconhecido.
- Nenhum endpoint de disparo, replay dos 13 leads 422, alteração de produção, orçamento, webhooks, números 0440/5287, DNS ou tokens nesta PR.

## Plano de adaptação
F0: contrato e pacote de reconciliador offline; F1 Bot (Antigravity): obter event_ref/ad_id sanitizado em fixtures Evolution e expor leitura interna incremental; F2 CRM (Codex): API paginada somente leitura com ability própria, datas/tenant e estado de campanha/entrega/pedido/pagamento; F3 Meta (Antigravity): coletor de Insights somente leitura da conta oficial, por Ad ID/data; F4 Bot: conectar adapters com cache, falha parcial, KPIs e linhas de atenção, flag default OFF.

## Gates
Todos os adapters devem produzir fixtures onde source outage não vira 0; campanhas CRM independentes do bot aparecem; Evolution HTTP 202 nunca vira entrega; Meta sem source_id não gera ad_id; registros legados 0440 não são deslocados para 5287; sem N+1 CRM, sem scanner de spool em refresh, UI responsiva 390px, RBAC HTTPS privado/CSP, rollback só do bot. Confirmar backend REAL e versão em produção antes de integrar.
