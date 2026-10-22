# BotWhatsApp — Estúdio Evidência

Serviço Go independente para dois workflows do WhatsApp no Estúdio Evidência, integrado à **Evolution Go 0.7.2 instalada** e à **API existente do Evidência CRM**. O número conectado aos anúncios recebe primeiro tratamento CTWA configurável, com saudação genérica se a frase não corresponder. Não modificar o código, banco ou infraestrutura do CRM.

## Estado em 22/10/2026

- `main` consolidou os contratos da campanha Natal 2026 **416 SEM NOME**, o worker outbox, suporte multi-instância, pausa humana, opt-out e cooldown.
- Esta atualização prepara compilação segura com **Go 1.27.3**, imagem de runtime Alpine 3.24.2 e um **relay durável separado**, desligado por padrão.
- **NADA foi implantado por esta atualização.** Os webhooks de produção, contêineres e campanhas existentes permanecem inalterados.
- O teste canário anterior foi relatado como bem-sucedido em texto/áudio e agendamento CRM, mas o `GET /api/campaigns/schedule/search` subsequente sofreu timeout. NÃO repetir esse agendamento.
- **Não liberar tráfego contínuo** enquanto a integração com o motor de status, quota de payload/disco e validação das mídias/sessões não forem concluídas.

## Separação dos números

| Canal | WhatsApp | Plataforma e função |
|---|---|---|
| Anúncios Meta | **48 3220-0440** | Evolution Go; gatilho literal da campanha, texto e áudio PTT; agenda pelo CRM |
| Saudação/Menu | **48 99642-5287** | Evolution Go; imagem única com legenda e opções 1–5, cooldown 24h, pausa humana |
| Emissor da proposta | Configurado no CRM | Número do CRM escolhido por `message_instances.id` explícito (relato: instância 1), não confundir com a saudação |

O inbound da instância `anderson` tem um webhook ocupado pelo serviço de status. **Nunca apontá-lo diretamente ao bot**, nem ocupar o :8050 ou alterar o status-engine para :8051.

## Fluxo CTWA ativo no código (configuração comercial ainda OFF)

- Texto padrão observado no relatório do Instagram: **Quero conhecer os pacotes do Primeiro Natal em Tijucas** e **Quero conhecer os pacotes de Natal em Tijucas**.
- Edite `config/ads-triggers.txt`: **uma ou mais frases completas, uma por linha**, até 50. O arquivo é carregado em `BOT_ADS_TRIGGERS_FILE` na inicialização; editar requer reinício **somente do bot** após autorização. Mensagens não correspondentes acionam a saudação no mesmo número Ads (`BOT_ADS_FALLBACK_MENU=true`).
- Etapas do anúncio: **texto opcionalmente personalizado → áudio PTT → imagem opcional com caption → vídeo MP4 opcional com caption → CRM 416 sem nome**. Falha/ausência de mídia é registrada, sem reenvio cego nem bloqueio do CRM; o CRM só é acionado com autorização/lead validado.
- A saudação usa uma imagem única com legenda quando configurada, senão envia o texto do menu 1–5. O número final **5287** terá somente saudação, em instância própria quando pareada.
- Webhook com metadados CTWA agora aceita até 1 MiB (limite, não garantia de aceitar qualquer miniatura). Exemplo sanitizado `testdata/ctwa-first-natal.json`.
- Leia `docs/CTWA-ROUTING-20261022.md` para decisões e ressalvas.

## Arquitetura

- `internal/core`: valida payload Evolution Go e roteia anúncios/saudação com filtro `IsFromMe`, LID/grupos e opt-out.
- `internal/instances`: identidade e role por UUID, número e token exclusivos.
- `internal/ingress`: intake autenticado; HTTP 202 após persistência do outbox.
- `internal/outbox`: intenções duráveis e estados `pending/attempt_started/confirmed/uncertain/suppressed`, dedup evento/lead e recuperação pós-restart.
- `internal/evolution`: clientes REST `/send/text`, `/send/media`, `/instance/status`.
- `internal/crm`: cliente somente consumidor do `POST /api/campaigns/schedule/execute` (campanha 416, contato DDD nacional sem 55, `Authorization` raw).
- `internal/relay` + `cmd/webhook-relay`: componente OPCIONAL em estudo. Grava evento antes do ACK; envia ao status-engine original PRIMEIRO, ao bot DEPOIS. Inativo.
- `cmd/botwhatsapp`: processo de produção com **flags de outbound OFF por padrão**.
- `deploy/compose.staging.yaml`: blueprint desativado, sem `ports`, independente do Compose dos serviços existentes.

## Antigravity — validação local única

Leia `AGENTS.md`, `docs/START-HERE.md`, `docs/INDEX.md`. Execute no Windows:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-local.ps1
```

O script valida Go, executa `go vet ./...`, `go test -count=1 ./...`, compila ambos os comandos e verifica **somente a sintaxe** de Docker Compose caso o Docker local esteja disponível. Não acessa a VPS.

Go 1.23 continua definido como *linguagem mínima* em `go.mod`, porém **a build de produção/CI usa Go 1.27.3** por segurança. Para reproduzir a build exata, recomenda-se instalar Go 1.27.3 no Windows.

## Documentação vigente

- `docs/INDEX.md` — ponto único de navegação: contrato vivo x histórico.
- `docs/CRM-API-416-PLAYBOOK.md` — JSONs e retornos do agendamento.
- `docs/MULTIINSTANCE-ARCHITECTURE.md` — identidades/sessões e shared lead claim.
- `docs/RELAY-ROLLOUT-GATES.md` — bloqueios do cutover e rollback do webhook sem tocar :8050.
- `docs/PRODUCTION-PREFLIGHT-20261022.md` — auditoria final e bloqueios objetivos.

A configuração `.env.example` não contém credenciais. Nunca versionar `.env.production`, mídia privada, tokens, logs brutos ou spool de clientes.
