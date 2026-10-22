# Pré-homologação de contêineres — não é deploy de produção

Este diretório contém um **exemplo de Compose de staging** com flags de bot/relay FORÇADAS para false. A existência do arquivo não autoriza alterar uma stack da Contabo.

- Serviços novos: `bot` na porta interna 8788 e `relay` na porta interna 8789.
- Nenhum publica `ports:` no host. A porta atual 8050 do status-engine fica INTACTA.
- Docker Compose usa DNS das redes externas preexistentes. Os nomes `evolution_internal` e `edge` vêm do inventário reportado: conferi-los com `docker network inspect` SOMENTE LEITURA antes de qualquer execução.
- A rede `edge` permite ao bot acessar CRM via HTTPS; relay só ingressa na rede `evolution_internal`.
- Volume durável próprio para cada serviço, uma réplica apenas por volume. Read-only rootfs, usuário sem root e drop de capabilities.
- O serviço relay é isolado e atualmente não possui vínculo com a instância `anderson`: a URL original do status-engine continua funcionando. Mudança única de webhook é decisão operacional posterior, com rollback.
- Os secrets não são versionados e ficam em `.env.production` local/secret manager. Exigir controle de acesso ao `RELAY_BOT_URL` e `RELAY_INGRESS_SECRET`, sem imprimi-los ou salvar em logs.

## Validação offline permitida
```bash
go vet ./...
go test -count=1 ./...
docker compose -f deploy/compose.staging.yaml config --quiet
```
O comando `config` somente valida o YAML. **Não executar `up`, `down`, `restart` ou `pull` na VPS** durante esta auditoria.

Leia `docs/RELAY-ROLLOUT-GATES.md` para os bloqueios, inclusive confirmação do comportamento/idempotência do motor de status original.
