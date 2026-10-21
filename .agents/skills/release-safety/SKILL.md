---
name: release-safety
description: Impedir implantação ou validação destrutiva em VPS compartilhada antes de aprovação.
---
# Segurança do release

Padrão: SEM DEPLOY, SEM MENSAGENS, SEM DISPARO DE CAMPANHA. Não executar `docker compose up/down` em projetos existentes, não alterar reverse proxy ou webhook sem mapa de consumidores.

Na fase de implementação: testar mocks e fixtures, `go test` somente nos pacotes alterados, checar docker compose config offline e segredos ausentes do Git.

Antes de qualquer homologação real: autorização expressa para destinatário de teste, identificação inequívoca das duas instâncias, confirmação da campanha e da instância de saída CRM, áudio e imagem providos pelo gestor, checagem de idempotência, plano de rollback independente e ausência de conflito com outros serviços.

Em caso de falha, parar apenas o novo bot quando autorizado; nunca derrubar Evolution Go nem outros containers.
