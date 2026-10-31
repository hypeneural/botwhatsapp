# Recibo Operacional: Restauração Webhook Evolution GO e Publicação BotWhatsApp CTWA P0

**Data:** 31 de Outubro de 2026  
**Repositório:** `hypeneural/botwhatsapp`  
**PR:** #10 (`fix/ctwa-native-proposal-handoff-20261027`)  
**Commit HEAD:** `e4515e3`  
**Operador:** Antigravity AI  

---

## 1. Resumo Executivo da Operação

1. **Diagnóstico Forense da Falha de Atendimento:**
   - A captura de tela do Evolution GO Manager (`wa-admin.etijucas.com.br`) e a inspeção do banco de dados `evogo_users` revelaram que a instância `evydencia` (`50e7ab4d-a380-42be-ab8a-d85fa982bfec`) estava com o Webhook configurado exclusivamente para `http://whatsapp-status-status-engine-1:8050/webhook/evolution/evydencia`.
   - O tráfego de entrada era entregue unicamente ao **SmartDialer Status Engine**, nunca alcançando o BotWhatsApp.
   - O número pareado no WhatsApp é `554896425287:15@s.whatsapp.net` (12 dígitos, formato com um único `9`), divergindo da variável anterior no `.env.production` (`5548996425287`, 13 dígitos).

2. **Capacidade do Spool do Relay:**
   - O spool continha **19.069 tombstones** concluídos (~4.59 MB).
   - O limite padrão `RELAY_MAX_EVENTS=20000` colocava o relay em saturação crítica de **95.34%** (`CAPACITY_WARNING`).
   - Foi configurado `RELAY_MAX_EVENTS=100000` e `RELAY_MAX_SPOOL_BYTES=268435456` no `compose.override.yaml` e no `.env.production`, reduzindo a saturação para **19.07%** (`INVENTORY_ONLY` / PASS).

3. **Deploy Cirúrgico e Isolado:**
   - O contêiner `botwhatsapp-stage-bot-1` foi reconstruído com o código de HEAD `e4515e3` da PR #10, contendo:
     - Encerramento comercial aposentado (`SendAdClosingText` suprimido; nenhuma promessa de atendimento por outro número).
     - Roteamento Ads restrito à frase exata aprovada (mensagens orgânicas não acionam saudação nem proposta).
     - Proteção contra autoacionamento (`IsFromMe` e eco da própria linha viram `OutboundObserved`).
     - Alinhamento de identidade para `BOT_ADS_PHONE=554896425287`.
   - O contêiner `botwhatsapp-stage-relay-greeting-1` foi reconstruído com o índice em memória de pendências (inicialização instantânea sem escanear 19 mil arquivos a cada ciclo) e limite de 100.000 eventos.
   - Nenhuma stack adjacente (Evolution Go, SmartDialer/Asterisk, CRM Laravel, PrintServer, Caddy, Postgres) foi reiniciada ou alterada.

4. **Cutover do Webhook:**
   - O Webhook da instância `evydencia` na Evolution GO foi alterado com sucesso via endpoint `/instance/connect` para o Relay dual:
     `http://botwhatsapp-stage-relay-greeting-1:8790/webhook/evolution/19778890a4dd31671a765b544ece84fd5ec330e7af529afd584e5fb8f541c937`
   - O Relay grava os eventos em disco com persistência durável (`202 Accepted`) e faz o fan-out paralelo para:
     1. Status Engine: `http://whatsapp-status-status-engine-1:8050/webhook/evolution/evydencia`
     2. Bot: `http://bot:8788/webhook/ads/0a989d0e7778d39996205abc27bb52f29aef789fd1081790106cbf92474bc6a7`

---

## 2. Evidências dos Gates de Segurança

### Local (CI / Worktree)
- `validate-ctwa-handoff-local.ps1 -Format`:
  - `go test ./cmd/... ./internal/...`: **11/11 pacotes PASS**
  - `go vet ./...`: **PASS**
  - `go build` bot e preview: **PASS**
  - Contrato offline: **PASS**
- Suíte Python:
  - `python -m unittest scripts/test_preflight_ctwa_account_switch.py scripts/test_relay_forensic_readonly.py`: **15/15 PASS**
- GitHub Actions CI na PR #10:
  - `build-only`: **PASS**
  - `test`: **PASS**

### Preflight na VPS (Read-Only)
- `scripts/preflight_ctwa_account_switch.py`:
  - `phone_format_valid`: `true`
  - `provider_connected`: `true`
  - `provider_phone_matches`: `true` (`554896425287`)
  - `relay_uuid_matches`: `true` (`50e7ab4d-a380-42be-ab8a-d85fa982bfec`)
  - `relay_bot_destination_valid`: `true`
  - `api_key_verified`: `true`
  - `status`: `PASS_BINDINGS_ONLY`
- `scripts/audit_relay_spool_readonly.py`:
  - `counts`: `{"completed": 19069, "records": 19069}`
  - `bytes`: `4591428` (~4.59 MB)
  - `capacity`: `{"event_count_limit": 100000, "event_count_used_ratio": 0.1907}`
  - `status`: `INVENTORY_ONLY` (exit 0)

---

## 3. Roteiro do Canário Consentido

| Cenário de Teste | Ação | Comportamento Esperado | Status de Validação |
|---|---|---|---|
| **1. Mensagem Orgânica** | Enviar "Olá" ou "Oi" | Sem resposta do bot, sem proposta, sem menu (`conversa_organica`). Relay entrega a Status Engine e Bot. | Aguardando canário |
| **2. Mensagem Própria** | Enviar texto pelo WhatsApp Web da própria linha | Bot ignora (`IsFromMe` / `OutboundObserved`), não dispara proposta nem loop. | Aguardando canário |
| **3. Gatilho de Anúncio Exato** | Enviar `Quero conhecer os pacotes do Primeiro Natal em Tijucas` | Resposta com Boas-vindas + Áudio + PDF + Agendamento CRM V4 com `dispatch_uuid`. Nenhuma mensagem prometendo outro número. | Aguardando canário |

---

## 4. Plano de Rollback Imediato
Em caso de qualquer falha no canário:
1. Reverter a URL do Webhook na Evolution GO para:
   `http://whatsapp-status-status-engine-1:8050/webhook/evolution/evydencia`
2. Parar o contêiner do Bot: `cd /srv/platform/stacks/botwhatsapp/deploy && docker compose -f compose.staging.yaml -f compose.override.yaml stop bot`
3. O SmartDialer Status Engine volta imediatamente a receber eventos diretos, sem perda de dados.
