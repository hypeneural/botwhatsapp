---
name: crm-readonly-contract
description: Auditar em leitura o contrato já existente da API de campanhas do Evidência CRM.
---
# CRM contrato — absolutamente read-only

Repositório externo `hypeneural/evydencia-crm`: SOMENTE LEITURA, sem deploy, modificação de código ou dados, criação de campanha, credenciais ou migrations.

Arquivos de leitura: `routes/api.php`, `app/Http/Controllers/Api/CampaignScheduledController.php`, `app/Http/Requests/Api/CampaignScheduledRequest.php`, `app/Services/Api/CampaignScheduledService.php`, `app/Helpers/CampaignScheduledContactsHelper.php`, `app/Http/Middleware/SingleAuthentication.php`.

Candidato à integração: `POST /api/campaigns/schedule/execute`. Cabeçalho `Authorization` esperado exatamente como token interno configurado (não assumir 'Bearer'). Campos obrigatórios `campaign` ID numérico, `contacts` texto separado por ponto e vírgula. `instance` opcional: ID numérico de `message_instances` do CRM, **não UUID da Evolution Go**. Validador aceita telefone BR com DDD, 10 ou 11 dígitos sem DDI. Conferir contrato antes do uso.

Ainda NÃO CONFIRMADOS: ID da campanha Natal 2026, instância de envio, token de serviço, URL e se a rota é a correta. Não inventar.
A aceitação do agendamento não significa mensagem entregue. Timeout ao POST é ambíguo: nunca repetir automaticamente; reconciliar com consulta já disponível se houver prova inequívoca, ou exigir intervenção.
Entregar tabela: rota, headers, campos, evidência, não-confirmados; sem editar o CRM.


## Evidência adicional
`Natal2026ProposalManifest.php` contém as constantes `with_name:415` e `without_name:416`. Isso NÃO é confirmação de campanhas ativas no banco da produção. `CampaignScheduledResource.php` retorna `data.id` se o agendamento foi criado. No bot, só um retorno parseável com ID positivo e campanha correta pode ser `confirmed`; outros resultados são `uncertain`.

Mesmo que a API legada aceite `instance:null`, o adaptador `internal/crm/client.go` deve exigir um `message_instances.id` explícito e validado antes do primeiro POST real. Não assumir a instância favorita do CRM. Nunca fazer POST durante auditoria.
