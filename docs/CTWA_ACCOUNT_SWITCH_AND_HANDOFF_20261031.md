# P0 CTWA — Configuração de número e retirada da mensagem de encaminhamento (31/10/2026)

**Repositório:** `hypeneural/botwhatsapp`, PR #10 (`fix/ctwa-native-proposal-handoff-20261027`).  
**Vizinhos:** `hypeneural/smartdialer-platform` PR #41, Evolution Go 0.7.2, CRM V4 — **somente leitura nesta fase**.  
**Decisão:** Código revisado e publicado para CI; **NÃO é aprovação de deploy, cutover ou reenvio**.

## Fatos da auditoria

- `cmd/botwhatsapp/phone_config.go` aceitava SOMENTE a constante `activeAdsPhoneE164=5548996425287`; trocar `BOT_ADS_PHONE` não funcionava. Esta constante foi retirada e ambas as funções de telefone passaram a usar o validador E.164 BR de `internal/core`. Nenhum número arbitrário recebe credenciais ou mensagens sem um **pareamento real validado** de instância.
- `internal/instances/registry.go` já separa identidades por UUID, slug, telefone, token ENV e segredo ENV, rejeitando UUID ou número repetido. `cmd/botwhatsapp/main.go` já oferece `BOT_ADS_INSTANCE_UUID`, `BOT_ADS_PHONE`, `EVOLUTION_ADS_APIKEY`, `BOT_ADS_WEBHOOK_SECRET` e gatilhos exatos. **Reutilizar essa arquitetura**, não reescrever multi-instances.
- `internal/core/core.go` já protege conversas orgânicas na conta Ads, `IsFromMe` e o número da própria empresa. `internal/outbox` persiste tasks, idempotência e opt-out num spool independente. `ScheduleNatal` mantém o cliente nativo CRM V4, **não 416**.
- `internal/workflow/plan.go` criava `SendAdClosingText` para fechar com a frase "outro WhatsApp entrará em contato". `main.go` ainda carregava `config/ads-closing-text.txt`, com a mesma promessa. A etapa foi DESATIVADA na criação de novos planos; `worker.ProcessTask` suprime **também tasks antigas pendentes** `SendAdClosingText` ANTES de I/O externo, deixando a próxima etapa `ScheduleNatal` elegível. Tasks antigas que já foram confirmadas/enviadas NÃO podem ser desfeitas.
- Texto fallback no `NewWorker` também prometia outro número; foi substituído por mensagem neutra. O texto oficial efetivamente montado por arquivo `config/ads-welcome-text.txt` já não faz essa promessa.
- O arquivo antigo `config/ads-closing-text.txt` é histórico e não é mais lido pelo binário atualizado. Foi conservado para comparação/rollback do código antigo; não é conteúdo operacional do novo fluxo.

## Fluxo após a alteração

```text
Evolution Go (instância validada) → Relay durável (UUID esperado)
   → Status Engine (mantém SmartDialer/telemetria)
   → Bot ingress (segredo da role Ads + UUID exato)
   → core.Select: somente gatilho literal aprovado; sem fallback orgânico
   → outbox: reação opcional → boas-vindas → áudio PTT
      → PDF → imagem/vídeo opcionais → API nativa CRM V4
   → FIM. Sem mensagem de redirecionamento, sem segundo número prometido.
```

A API CRM poderá encaminhar comunicações pela régua própria. O bot **não presume** que o CRM usará o mesmo número nem promete que usará outro; confirmação de despacho não equivale a entrega do cliente.

## Procedimento para mudar o número futuramente (sem mudar Go)

**Regra:** não basta editar `BOT_ADS_PHONE`. Troca exige observar qual Evolution instance tem o aparelho realmente conectado, e atribuir `UUID` + `apikey` daquela conta + webhook e relay à **mesma identidade**, nunca por nome de contato ou LID. A mesma linha não pode ter role Greeting simultânea.

1. Obter, somente leitura, `GET /instance/all` da Evolution Go com chave global privada, guardar resposta completa SOMENTE em arquivo protegido 0600. Os tokens presentes nesse arquivo são sensíveis; nunca enviá-lo ao ChatGPT/Git.
2. Preparar novos arquivos privados de `.env` do bot e do relay (sem trocar os ativos ainda), preservando `BOT_DATA_DIR`, CRM V4 e demais stacks; alterar `BOT_ADS_PHONE`, `BOT_ADS_INSTANCE_UUID` e `EVOLUTION_ADS_APIKEY` com base em instância real. `BOT_ADS_WEBHOOK_SECRET` deve corresponder ao sufixo de `RELAY_BOT_URL`. `RELAY_EXPECTED_INSTANCE_UUID` deve corresponder à nova UUID. Revisar status/assinaturas/webhook do novo provider.
3. Executar o script **somente leitura**, localmente ou no Avell, contra os arquivos privados:

```bash
python3 scripts/preflight_ctwa_account_switch.py \
  --bot-env /PRIVATE/candidate-bot.env \
  --relay-env /PRIVATE/candidate-relay.env \
  --inventory-json /PRIVATE/evolution-instance-all.json \
  --previous-bot-env /PRIVATE/current-bot.env \
  --outbox-tasks-dir /PRIVATE/current-spool/tasks
```

Arquivos de entrada precisam ser 0600. Se o inventário da API não fornecer `token`, acrescente `--verify-live-status --provider-base-url http://127.0.0.1:14000`: o script usa a `apikey` candidata num **GET autenticado** a `/instance/status` (somente loopback), verifica a conexão e o `myJid`, e não imprime resposta privada. Se esse GET não passar, o corte fica bloqueado. Valores não são impressos, só booleanos. A ferramenta não lê/escreve o banco CRM nem cria instância/QR/campanha.

4. Verificar `old_outbox_clear=true` (nenhuma task `pending`, `attempt_started` ou `uncertain` associada à antiga UUID). **Não apagar ou reenviar fila** para conseguir passar. Reconciliação pontual é necessária quando houver pendências.
5. Confirmar em modo leitura que o relay usa porta REAL (8790 segundo dossiê; 8789 no template), tem volume durável, rede e duas rotas; Status Engine PR #41 terá encaminhamento SmartDialer privado funcionando. Snapshot do spool, imagem atual e rollback ANTES de reconstruir exclusivamente o bot.
6. Desvincular rota da instância anterior somente depois de backup e, quando autorizado, fazer **um corte reversível** da Evolution para a nova instância + relay usando endpoint comprovado no fork/tag instalado. Não chamar `/instance/connect` por inferência; ele pode afetar a sessão e inscrições. Verificar nova URL/assinaturas no Postgres/GET e deixar outro aparelho intacto. **Mudar o número Meta Ads/CTA** é uma operação externa separada; atualizar anúncios/conta para o número novo após o corte.
7. Canários: mensagem orgânica → nenhum disparo, `IsFromMe` → nenhum disparo, gatilho CTWA exato → UMA sequência de teste e recibo CRM; verificar zero `SendAdClosingText` e zero promessa de outro número no texto/áudio escolhido. Áudio gravado precisa ser revisado separadamente se menciona outro WhatsApp.
8. Restabelecer apenas nova instância; nunca redirecionar automaticamente o número anterior nem reconectar 0440 sem uma solicitação específica. Em falha, retornar routing anterior e reconciliar idempotências antes de qualquer reenvio.

## Gates de testes no Avell

```powershell
go test -count=1 ./internal/core ./internal/instances ./internal/workflow ./internal/outbox ./cmd/botwhatsapp
go vet ./...
python -m unittest scripts/test_preflight_ctwa_account_switch.py -v
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1
```

Testar o **HEAD da PR #10**, além de `git status --short`, workflow CI e digest. Não rodar canário externo sem autorização; nem reprocessar 29–31/10. Preserve `BOT_ADS_FLOW_COOLDOWN_HOURS`, telefones e históricos de opt-out.

## Desempenho e capacidade do relay — melhoria aplicada

- `internal/relay/store.go::Pending()` fazia `ReadDir + ReadFile` em **todos** os JSON a cada 30 segundos, mesmo para eventos antigos completamente entregues. O dossiê reportou **19.069 tombstones concluídos, 0 pendentes**, portanto o custo teórico seria cerca de 2,3 milhões de leituras de arquivos por hora se a contagem permanecesse estável. **Isso não é um benchmark real de latência**, mas é o comportamento confirmado pelo código anterior.
- Agora `NewStoreWithLimits` reconstrói **uma única vez** um índice em memória somente dos receipts ainda não concluídos, conferindo identidade e JSON do spool. `write()` mantém o índice sincronizado com o registro durável depois da renomeação atômica e antes do fsync do diretório; `Pending()` visita apenas os pendentes e preserva ordenação determinística. Após restart, o índice vem dos arquivos, **nunca é fonte de verdade independente**. Somente um processo pode ser dono do volume. Não remover tombstones nem dedupe.
- `scripts/audit_relay_spool_readonly.py` agora aceita os limites **reais** para emitir `CAPACITY_WARNING` se slots ou bytes ocuparem 90% ou mais; é erro de preflight (exit 2), não uma limpeza.
- O `.env.example` tem `RELAY_MAX_EVENTS=20000`. **Se** este for o limite efetivo da VPS, 19.069/20.000 representam **95,345% dos slots**. Isso deve ser validado no Compose/env privado antes de restaurar webhook. Não aumentar capacidade cegamente sem conferir free space, total de JSON, TTL e política de tombstones de 7 dias.

```bash
# Somente leitura; informar LIMITES VERIFICADOS da VPS:
python3 scripts/audit_relay_spool_readonly.py \
  --spool /VOLUME_REAL_RELAYER \
  --max-events 20000 \
  --max-bytes 134217728
```

**Atenção:** os dois valores acima são EXEMPLOS do `.env.example`; Antigravity deve substituí-los pelos valores efetivos da VPS. Se retornar `CAPACITY_WARNING`, não realizar o cutover até revisar limites, capacidade física, retenção e recibo de rollback. Não apagar arquivos completos só para ter espaço; tombstones preservam idempotência.

## Melhorias futuras (fora do hotfix, NÃO refatorar agora)

- Inventário privado de instâncias na interface gerencial mostrando somente rótulo e os últimos 4 dígitos, com mudança proposta via duas fases e auditoria, **sem expor APIKEY nem modificar provider pelo browser**.
- Observabilidade com métricas de latência por destino, pendências do relay e outbox, erros incertos e estado da instância; não fazer scan full do spool a cada request de painel.
- Migração do formato de journal file-based para persistência indexada só quando volume/latência exigirem, preservando tombstones de dedupe e recibos.
- Separar explicitamente `send_accepted`, `crm_dispatch_accepted`, `delivered` e `read` no dashboard para não apresentar proposta agendada como mensagem entregue.

## Fontes

- https://github.com/evolution-foundation/evolution-go/blob/main/docs/wiki/guias-api/api-instances.md
- https://github.com/evolution-foundation/evolution-go/blob/main/docs/wiki/recursos-avancados/events-system.md
- Dossiê operacional de 31/10/2026: `docs/AUDITORIA_VALIDACAO_CHATGPT_P0_20261031.md`
- PR #41: `https://github.com/hypeneural/smartdialer-platform/pull/41`

**Status de implantação:** NO-GO até CI final do novo HEAD, verificação da imagem 27/10, dependência SmartDialer PR #41, prova da instância/UUID e canário controlado. Não fazer deploy ou troca de webhook nesta auditoria.
