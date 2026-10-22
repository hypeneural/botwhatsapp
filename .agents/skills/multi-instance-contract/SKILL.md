---
name: multi-instance-contract
description: Evoluir o bot para várias instâncias Evolution Go sem duplicar campanhas CRM nem substituir o webhook do motor de status.
---
# Multi-instância — contrato essencial
Leia `docs/MULTIINSTANCE-ARCHITECTURE.md`.
1. Aproveite o novo `internal/instances/registry.go` + testes, não implemente outro registry concorrente.
2. Mapear explicitamente role, UUID, número comercial, ENV do token e segredo, e enabled por instância. Sem deduzir instância da cifra do número ou de fields `instanceToken` no payload.
3. Reusar `core.Select`, worker e cliente Evolution GO já auditados. Nunca usar a mesma apikey indiscriminadamente para dois números.
4. Claim comercial `(tenant,phoneNational,campaign416,janela America/Sao_Paulo)` deve ser **compartilhada entre instâncias**, num repositório durável/transacional comum. EventKey inclui UUID; LeadClaimKey NÃO inclui UUID.
5. Não publicar ou alternar webhook `anderson`; status engine tem consumidor anterior. Escolha encaminhamento autorizado e durável ou relay fan-out depois de auditar os serviços existentes.
6. Reconciliar com worker local do AntiGravity ANTES de conectar `Registry.Decide` ao handler; nenhuma refatoração que perca journal e estados do canário.
7. Testar concorrência, restart e mesma lead em dois números. Sem envio real.
