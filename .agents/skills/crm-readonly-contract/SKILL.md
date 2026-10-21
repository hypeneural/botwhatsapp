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
