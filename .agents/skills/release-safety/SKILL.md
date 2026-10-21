---
name: release-safety
description: Impedir implantação ou validação destrutiva em VPS compartilhada antes de aprovação.
---
# Segurança do release

Padrão: SEM DEPLOY, SEM MENSAGENS, SEM DISPARO DE CAMPANHA. Não executar `docker compose up/down` em projetos existentes, não alterar reverse proxy ou webhook sem mapa de consumidores.

Na fase de implementação: testar mocks e fixtures, `go test` somente nos pacotes alterados, checar docker compose config offline e segredos ausentes do Git.

Antes de qualquer homologação real: autorização expressa para destinatário de teste, identificação inequívoca das duas instâncias, confirmação da campanha e da instância de saída CRM, áudio e imagem providos pelo gestor, checagem de idempotência, plano de rollback independente e ausência de conflito com outros serviços.

Em caso de falha, parar apenas o novo bot quando autorizado; nunca derrubar Evolution Go nem outros containers.


## Gate adicional
O código atual é preview-only e não pode enviar. A falta das duas instâncias na GO NÃO autoriza pareamento automático. Falhas de `/readyz` ou ausência da campanha/instância explícita no CRM são bloqueios, não motivos para forçar deploy. Testes locais primeiro, depois aprovação distinta para pareamento e aprovação distinta para envio canário. O container futuro NÃO deve publicar porta de webhook em interface pública.
