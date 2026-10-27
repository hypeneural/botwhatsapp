# CTWA 5287 → Proposta Natal V4: correção cirúrgica 27/10/2026

## Evidência e estado

Main do bot auditada: `6af807225704c710b68a5a185c0725b26369cb3f`.
Main do CRM auditada: `60ba4e2de4fecee290f9b51ac85fac679e220315`.
No CRM, `NatalProposalEntryPoint::assertGenericAllowed` proíbe explicitamente 415/416 no POST genérico; 422 é rejeição definitiva, não timeout. Histórico relatado pelo operador/Antigravity: 13 ocorrências CTWA com PDF/áudio aceitos pela Evolution e CRM recusado com `natal_proposal_native_entry_required`. Sem acesso VPS a partir deste patch: não afirmar entrega final ou que o novo binário esteja em produção.

## Novo contrato, sem alterar o CRM

- Endpoint nativo já existente: `POST /api/v1/extension/proposals/schedule`.
- `Authorization: Bearer <Sanctum token do operador de serviço>` com ability **extension:proposal-write** e usuário ativo autorizado por `ProposalFlowPolicy::send`; diferente do token raw da API antiga.
- Body: `{"request_id":"UUIDv5 estável da task","phone":"DDDNÚMERO","name":null,"resend_mode":"normal"}`.
- Header `X-Idempotency-Key` é igual ao `request_id`.
- Receber apenas 200/201 `{"data":{"status":"scheduled","dispatch_uuid":"UUID"}}` como agendamento nativo confirmado. NÃO confundir com entrega Z-API/WhatsApp.
- 400/401/403/404/422: rejeição definitiva, não repetir automaticamente. Timeout/5xx/429/409/recibo inválido: **uncertain**, conciliar antes de qualquer tentativa.
- A variante 415/416 é resolvida somente no CRM segundo sua regra comercial (nome do WhatsApp é inseguro). Não liberar o caminho genérico.
- Flag secret `EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN` configurada APENAS em ambiente privado de produção. Contas, tokens e telemetria não devem ser commitados.
- O fluxo 5287 já usa opt-out, pausa humana e cooldown 24h; não zerar journal, não executar envio duplicado, não trocar webhook, portas, Evolution ou outros serviços.

## Painel já existente

O bot já expõe `/painel` e `/api/dashboard/contacts`. Nesta PR:
- Bloquear acesso sem `BOT_DASHBOARD_TOKEN`; exigir Basic Auth HTTPS no navegador, ou header `X-Dashboard-Token` em uso interno. Não usar `?token=` em links.
- Exibir `Rejeitado / não agendado`, `Incerto / revisar`, `Pendente` ou `Aceito no CRM (proposal-dispatch-UUID)`, sem confundir HTTP 2xx com mensagem recebida.
- O painel é leitor do journal local, **não consulta automaticamente o CRM**. Não rotular histórico não verificado como entregue. Não publicar diretamente na internet.

## Execução curta do Antigravity, com restrições

1. Comparar main/head/produção e PR #9. As proteções de telefone da PR #9 já foram incorporadas nesta PR #10; verificar paridade e evitar aplicar duas vezes.
2. Executar `scripts/validate-ctwa-handoff-local.ps1` uma vez; caso o gofmt falhe, usar `-Format`, revisar diff e repetir. Não confundir script publicado com PASS real.
3. Revisar o token do painel e acesso HTTPS privado (sem query secret), e prover Sanctum **com escopo mínimo**. Instância de saída é selecionada pelo agendador nativo do CRM, não pelo `message_instances.id` hardcoded do legado.
4. Pré-produção: verificar configuração real do 5287, decisão pelo trigger literal, spool e 13 rejeições sem reinicializar/limpar journal. Consultar primeiro o CRM; separar dispatch já existente, rejeição definitiva e incerto. Não fazer replay automático.
5. Canário **um novo lead autorizado**: receber texto no 5287, observar etapas Evolution, confirmar recibo nativo `dispatch_uuid` e verificar no CRM que a proposta foi de fato agendada. Confirmar processamento posterior pela instância CRM sem inferir entrega a partir de aceite HTTP.
6. Rollout de somente bot após aprovação operacional, com health e rollback preservando volume, UUIDv5 e journal. Nunca reiniciar Evolution, Status Engine ou CRM para resolver esta PR.

## Histórico a reconciliar

Os 13 contatos relatados pelo Antigravity em 27/10 tiveram HTTP 422, mas **não foram reencaminhados por este patch**. Conferir elegibilidade, compra, opt-out e eventual dispatch antes de programar envios manuais com aprovação. A lista de 19 contatos dos prints não deve ser tratada inteira como falha CRM: são fontes diferentes, com sobreposições.


## Histórico durável e painel de auditoria (incremento desta PR)

- `internal/ingress/server.go` registra os eventos recebidos identificáveis em `BOT_DATA_DIR/<sha256>.json`, incluindo mensagem normal ou mídia sem gatilho, antes de HTTP 202. Mensagens sem ID ou saída `fromMe` não viram evento de ingresso. O payload bruto da Evolution (instanceToken, thumbnail e URL da mídia) **nunca é salvo**.
- `internal/ingress/audit.go` retém somente tipo, **até 280 caracteres** de texto inbound, sinal contextual de anúncio, gatilho literal e classificação; esses dados possuem PII e devem ficar em diretório 0700, arquivos 0600.
- Origem `meta_ads` é atribuída quando houve gatilho **exato** ou metadados `externalAdReply`; sem isso o painel classifica `normal` (não equivale a prova da origem orgânica). Não confiar somente no papel Ads da instância.
- `GET /api/dashboard/events?limit=150` mostra o histórico recente (limite servidor: 500), em ordem de ingresso, com correspondência hash da tarefa da outbox, janela de atendimento BRT, tentativa, retorno/recusa, receipt de Evolution e dispatch UUID do CRM. `/api/dashboard/contacts` continua compatível.
- `/painel` possui filtros de origem, situação, nome, telefone e trecho de texto, e expansão das etapas. O frontend escapa o conteúdo dos contatos antes de usar HTML; todas as APIs são somente leitura.
- `BOT_DASHBOARD_TOKEN` é obrigatório. Basic Auth do navegador via HTTPS privado ou `X-Dashboard-Token` de acesso interno. **Nunca `?token=`**, link público ou endpoint sem autenticação.
- 2xx da Evolution = API aceitou, não é confirmação de entrega/leitura; 200/201 do CRM com `dispatch_uuid` = despacho agendado, não mensagem entregue. `HTTP 422` torna a etapa rejeitada; timeouts/429/5xx permanecem `uncertain` sem replay.
- Histórico informativo é limitado pelo processo de manutenção (com worker ativo) de seis em seis horas a até 30.000 arquivos e 90 dias. O journal de **etapas** usa retenção separada de até 30 dias e limite de 20.000 tarefas, então registros antigos podem conservar o ingresso mas perder detalhes por etapa. A limpeza **não** remove tasks, eventos de dedupe, contatos, opt-out, claims nem estados `uncertain`. Registros antigos não possuem texto/tipo do ingresso retroativamente.
- Este painel lê apenas os journals que chegaram a este bot. Não substitui logs da Meta, do Evolution, do CRM ou do serviço Status e não inventa entradas desconhecidas.

## Validação local oficial em Windows

Abrir a branch/PR #10 em worktree própria. Conferir diffs e trabalho de outros agentes. Executar:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1
```

Se o gate parar especificamente por `gofmt`, executar **uma vez** com `-Format`, conferir `git diff` e repetir sem a flag. `-Race` é opcional apenas com CGO funcional. O gate roda teste CTWA/ingress/CRM/outbox/dashboard e `go vet`; não realiza POST externo, deploy nem reinicia serviços. Guardar resultado local com SHA real. A execução local ainda não foi realizada nesta edição remota.

## Composição das PRs

A PR #10 incorporou em código `phone_config.go`, `phone_config_test.go`, o `main.go` fail-closed e a configuração de `BOT_GREETING_PHONE` vazio da PR #9. **Não cherry-pickar a PR #9 outra vez às cegas**: comparar os dois diffs, reconciliar eventuais ajustes documentais e marcar #9 como superada somente após validação. Número 0440 é exclusivamente histórico. O bot já não inicia o papel Ads sem `BOT_ADS_PHONE=5548996425287`.


## Critérios de aceite

- NOVO lead: recibo `dispatch_uuid` persistido, proposta visível no CRM, sem duplicação, phone normalizado conforme `BrazilMobilePhone`.
- Rejeição 422 visível e sem novo POST, inclusive registros legados `uncertain` contendo `HTTP 422` (reclassificação apenas de leitura).
- Timeout preserva `uncertain`; painel não informa sucesso falso.
- Painel restrito, sem URL token nem HTML não escapado de perfis.
- Confirmação de que PR #9 corrige 0440/5287 antes da implantação.
