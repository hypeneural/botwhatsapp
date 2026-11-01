# Handoff P0 — relay Evolution Go e proteção de atendimento (31/10/2026)

**Alvo:** esta PR #10, branch `fix/ctwa-native-proposal-handoff-20261027`.  
**Parceiro:** `hypeneural/smartdialer-platform`, branch `fix/evolution-webhook-ownership-p0-20261031`.  
**Estado:** código de testes/preflight publicado, **nenhum cutover realizado**.

## Forense em cadeia

```text
Evolution Go (instância evydencia, número oficial 5287)
 -> webhook por instância no PostgreSQL public.instances [DESTINO ÚNICO CANÔNICO]
 -> relay privado (container reportado na porta 8790, imagem/override não verificados)
     -> (1) Status Engine /webhook/evolution/evydencia, commit ACK
         -> telemetria status_engine.wa_webhook_logs
         -> encaminhamento SmartDialer best-effort (não durável)
     -> (2) BotWhatsApp /webhook/ads/<SEGREDO_PRIVADO>, commit ACK
         -> ingress journal durável e decisão core.Select
         -> outbox → Evolution Go /send e CRM /api/v1/extension/proposals/schedule
```

**Não é fan-out simultâneo:** `internal/relay/server.go::Process` exige Status Engine primeiro. HTTP 202 da entrada indica persistência do evento, NÃO entrega aos dois consumidores. `StatusDone`/`BotDone` são ACKs internos, NÃO prova de WhatsApp lido, áudio enviado ou proposta entregue. O Status Engine atualmente pode ACKar 200 mesmo quando o processamento canônico falha.

### Bloco de compatibilidade obrigatório

A documentação oficial geral da Evolution Go pode emitir
`{"event":"MESSAGE","instance":"evydencia","data":{"key":{...},"message":{...}}}`,
enquanto:
- `internal/relay/store.go::EventKey` exige `instanceId=UUID` idêntico ao `RELAY_EXPECTED_INSTANCE_UUID`;
- `internal/core/core.go::Select` exige `event=Message`, `data.Info.ID`, `data.Info.Chat`, `data.Info.IsFromMe` e `data.Message`;
- `internal/ingress/server.go` compara a UUID no próprio envelope.

**A diferença pode produzir 400 no relay ou lead ignorado pelo bot após cutover.**
Não adaptar por aproximação de campos, normalizar nem inferir telefone de LID sem amostra **da tag/fork/digest implantado** (o código Evolution Go main/v1 difere do 0.7.2). A amostra deve ser colhida da tabela Status Engine em modo privado, sem exposição no Git. Não salvar URLs secretas neste documento.

### Scripts publicados

```bash
# Da worktree botwhatsapp PR #10, OFFLINE
python3 -m unittest scripts/test_relay_forensic_readonly.py -v

# VPS, todos read-only e sem imprimir mensagens
python3 scripts/audit_relay_spool_readonly.py --spool /VOLUME_REAL_DO_RELAY

# Duas entradas privadas 0600: payload Evolution real e UUID esperada
python3 scripts/preflight_evolution_relay_contract_readonly.py \
  --sample-file /ARQUIVO_PRIVADO/event.json \
  --expected-uuid-file /ARQUIVO_PRIVADO/instance_uuid

# Testes Go do código relay/core/ingress/outbox
go test -count=1 ./internal/relay ./internal/core ./internal/ingress ./internal/outbox
go vet ./...
```

`scripts/audit_relay_spool_readonly.py` não imprime arquivo/evento/telefone; resume contagem/status/erro de permissões, bytes e idade do pendente. Deve ser rodado com mesmo UID ou acesso somente leitura legítimo; arquivos do spool têm permissão 0600.

`scripts/preflight_evolution_relay_contract_readonly.py` retorna `BLOCKED_CONTRACT` e exit=2 para formato incompatível; **não** envia evento ao relay nem converte payload em produção. Salve apenas o status/checks bool no recibo.

## Gates do Antigravity (sem reenvio)

1. Confirmar SHA de `main`, PR #10 e digest do container real. PR #10 tem correções de 29/10 de `IsFromMe`, bloqueio do próprio número e fallback Ads desligado. **Imagem antiga não pode voltar a responder automaticamente a mensagens orgânicas.**
2. Confirmar composição/porta/URL/volume efetivos, `RELAY_STATUS_URL` e `RELAY_BOT_URL` por comparação de hash/redação. Não confiar no Compose staging (porta default 8789 vs relato 8790).
3. Confirmar `RELAY_EXPECTED_INSTANCE_UUID`, chave secreta do ingress (rotacionar a publicada em chat), `BOT_ADS_INSTANCE_UUID`, flag `BOT_ENABLED`, repositório de origem, logs de falha e tamanho de evento; NUNCA registrar valores.
4. Provar formato do evento e assinatura `MESSAGE` com amostra sanitizada privada; sucesso exige todos os booleanos do script, não somente `/healthz`.
5. Confirmar `status-engine` e SmartDialer mantêm fluxo atual com credencial de forwarding privada após correção em smartdialer-platform.
6. Testar somente fixture sintética, com mocks HTTP nos dois consumidores e falha/timeout; nenhum POST real a serviços comerciais.
7. Apenas após gates e autorização de corte: canário único consentido para verificação de caminho real, sem replay dos eventos de 29–31/10, verificando simultaneamente proteção humana e agendamento CRM isoladamente da entrega.
8. Plano de rollback: guardar a URL e as inscrições anteriores **privadamente** e confirmar reconfiguração de `evydencia` com contrato do binário instalado. Os registros pendentes do relay não podem ser reprocessados em massa sem conciliação CRM.

**Bloqueios:** contrato real do webhook, capacidade/ownership do relay, segredos expostos, imagem efetiva do bot, backup/replay, schema PG divergent. `GO_CODIGO` não é `GO_PRODUCAO`.
