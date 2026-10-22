# Relay Evolution Go → Status Engine → Bot | Gate forense de produção

**Versão estudada:** Evolution Go instalada reportada como 0.7.2 com patch WebP local; contrato oficial da tag `0.7.2` prevalece sobre a documentação geral v1.0. O caminho do status engine foi confirmado no relato do AntiGravity, mas **não obtivemos o código do motor de status** nesta auditoria.

## A correção de arquitetura mais importante
A proposta anterior de "ocupar a porta 8050 e mudar o status-engine para 8051" é **REJEITADA**. A porta 8050 e o serviço existentes DEVEM permanecer intactos.

Nova topologia proposta:
```text
ANTES (INALTERADO até aprovação):
Evolution Go anderson → status-engine:8050/webhook/evolution/anderson

DEPOIS (somente com mudança autorizada e reversível):
Evolution Go anderson → webhook-relay:8789/webhook/evolution/{secret}
                           ├─ 1º status-engine:8050/webhook/evolution/anderson
                           └─ 2º bot:8788/webhook/ads/{segredo-independente}
Bot → Evolution Go /send/text|media, por APIKEY própria
Bot → CRM HTTPS /api/campaigns/schedule/execute, campaign 416, instance NUMÉRICA do CRM
```
Sem mudança em nenhuma porta, image, Compose ou rede atual da Evolution Go/status-engine. O `webhook-relay` é um NOVO serviço e possui seu próprio volume.

## Implementação no repositório
- `cmd/webhook-relay`: processo independente `RELAY_ENABLED=false` por padrão.
- `internal/relay`: grava JSON integral do evento (modo 0600; volume privado) e fsync antes de ACK 202. Verifica UUID da instância e segredo independente na rota; encaminha **status primeiro** e **bot depois**, persistindo ACK de cada um separadamente. Se status falhar, o bot aguarda.
- Retry interno com backoff conservador 15s; upstream Evolution Go 0.7.2 pode tentar mais 5 vezes. Repetições idênticas são deduplicadas por UUID/evento/messageID ou hash da entrada.
- Ao concluir ambos os envios, remove o payload do registro, deixando tombstone por 7 dias; eventos pendentes NÃO são apagados automaticamente. Limites `RELAY_MAX_EVENTS` e `RELAY_MAX_SPOOL_BYTES` (padrão 128MiB) recusam 503 quando cheios, sem descartar eventos pendentes. A capacidade é contabilizada novamente após reinício.
- O status ENGINE precisa ser idempotente: queda entre recebimento 2xx e persistência do ACK do relay pode ocasionar reenquadramento ao status-engine. **Exatamente uma entrega física NÃO é garantida**.

## Bloqueadores obrigatórios antes de mudar o webhook
1. Auditar código/contratos reais do `whatsapp-status-status-engine-1`, especialmente schema de entrada, HMAC/autorização, ACK, retries e deduplicação. Se o motor não tolera replay, não fazer cutover.
2. Confirmar nomes de Docker networks (`evolution_internal`, `edge`), resolução DNS e link do status-engine por `docker network inspect` / leituras sem mutação. O status atual utiliza :8050; não mudar.
3. Verificar que o sidecar recebe e persiste em mídia durável e que os dois destinos são acessíveis por rede privada. Exigir `no-new-privileges`, usuário não-root e volume protegido.
4. Validar limites de tamanho dos webhooks reais da instância (podem conter mensagens com mídia). O default 256 KiB pode rejeitar payloads grandes; antes de produção observar tamanhos sem inspecionar conteúdo privado. Existe quota de bytes (128MiB por padrão), mas é preciso calibrar e criar alarmes de backlog/disco antes do cutover.
5. Confirmar versão exata/patch do binário GO instalado e se a mesma instância mantém apenas um webhook por instância. Não alterar `WEBHOOK_URL` global.
6. Para operar bot Ads: frase Meta exata confirmada, UUID da instância, token por instância, asset OGG Opus aprovado, token interno CRM e instância CRM numérica 1 confirmada em leitura; **nunca** repetir canário de campanha 416 que já foi agendada para o contato teste.
7. Para saudação: número `48 99642-5287` pareado como NOVA instância GO separada, imagem/caption reais, consentimento/mensagens de entrada, opt-out durável; não misturar com a saída CRM.
8. Instâncias conectadas precisam ter saúde operacional verificada e `BOT_ENABLED/BOT_OUTBOUND_ENABLED/BOT_OPERATIONAL_AUTHORIZATION` só ativadas quando aprovação real estiver registrada. Nenhuma flag ativada automaticamente pelo Compose.

## Rollback explícito
1. Salvar PRIVADAMENTE destino original exato do webhook e configuração `subscribe` ANTES da alteração.
2. Redirecionar em janela aprovada UMA VEZ para o relay; preservar Status e API GO em execução.
3. Em falha, restaurar URL e inscrição ORIGINAIS pelo mesmo gerenciador autorizado e confirmar persistência por leitura; observar novo evento status sem reabrir campanha.
4. Não executar `down` do status-engine, não ocupar 8050, não migrar banco, não tocar Caddy.
5. Mesmo após rollback, analisar pendências do volume relay e resolver efeitos incertos **antes de retomar**, para não gerar fluxo duplicado.

## Limitações ainda abertas
- Persistência de eventos do relay guarda temporariamente o payload original com `instanceToken` e PII. Em volume privado 0600, sem logs, com scrub na conclusão. Implementar proteção de disco/criptografia e observabilidade operacional conforme política de privacidade antes da operação permanente.
- Capacity por número de arquivos E total de bytes já foi implementada. Porém faltam métricas/alertas operacionais e dados de volume real; atingir a capacidade devolve 503 e a GO só tenta entregar um número limitado de vezes. **Produção contínua não liberada sem observabilidade e preflight**.
- Exibição do status `api_accepted` não significa entregue/lido; CRM `scheduled` não garante proposta entregue.
- Um processo por volume: não há trava distribuída. Rollout com `replicas:1`, sem autoscaling.
- Não presumir que a evolução Go embuta assinatura HMAC; o caminho secreto só protege dentro da rede privada. Nunca publicar webhook publicamente.

## Fontes oficiais
- Evolution Go tag 0.7.2: https://github.com/evolution-foundation/evolution-go/tree/0.7.2
- Eventos GO e categorias de webhook: https://github.com/evolution-foundation/evolution-go/blob/main/docs/wiki/recursos-avancados/events-system.md (há diferenças v1.0; usar somente como contexto).
- Docker networking: https://docs.docker.com/compose/how-tos/networking/
- Docker Compose services/secrets: https://docs.docker.com/reference/compose-file/services/ e https://docs.docker.com/reference/compose-file/secrets/
- Go suporte: https://go.dev/doc/devel/release
- Alpine atual: https://www.alpinelinux.org/releases/
