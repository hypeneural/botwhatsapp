# START-HERE — BotWhatsApp (vigente em 31/10/2026)

**Fonte operacional para Antigravity Flash Lite; somente a seção vigente abaixo orienta mudanças.** Documentos e exemplos de 22–27/10 podem descrever 0440, fallback de saudação ou endpoint CRM 416; são **históricos e estão superados**.

## Tarefa corrente — PR #10

**Branch:** `fix/ctwa-native-proposal-handoff-20261027`. **Não** abrir outra PR nem sobrescrever commits existentes. Leia `AGENTS.md` e os dois documentos:

1. `docs/CTWA_ACCOUNT_SWITCH_AND_HANDOFF_20261031.md` — auditoria de número configurável e retirada da mensagem final.
2. `docs/PROMPT_ANTIGRAVITY_CTWA_ACCOUNT_SWITCH_20261031.md` — sequência executável no Avell, sem necessidade de inventar scripts.

## Verdades que não podem regredir

- O número atual de anúncios é terminado em **5287**. A instância **0440** é antiga/desconectada e **NÃO** deve ser reativada automaticamente.
- O telefone do Ads é configurável pelo `BOT_ADS_PHONE`, mas só entra em produção depois de pareamento read-only com UUID/conta conectada/chave Evolution/destino do relay via `scripts/preflight_ctwa_account_switch.py`.
- Papel Ads aceita SOMENTE mensagens com frase exata configurada, não saúda conversa orgânica nem responde a `IsFromMe`.
- O bot encerra o atendimento automático após a etapa de CRM V4, **sem mensagem sobre encaminhamento por outro WhatsApp**. Um `SendAdClosingText` legado pendente é suprimido sem impedir a etapa CRM seguinte.
- API correta para proposta: `POST /api/v1/extension/proposals/schedule`, não o endpoint batch 416. HTTP 201 é despacho, não prova de entrega.
- `BOT_ADS_FLOW_COOLDOWN_HOURS`, opt-out, pausa humana e claims persistidos não podem ser reiniciados na troca de número.
- Relay escreve spool durável e envia primeiro ao Status Engine, depois ao BotWhatsApp. Status Engine PR #41 tem bloqueio de secret `SMARTDIALER_WEBHOOK_URL`; não publicar mudanças de roteamento antes do gate PASS.
- **Sem deploy automático:** a imagem relatada em 31/10 é de 27/10 e a rota Evolution ainda apontava para o Status Engine; antes do cutover é obrigatório confirmar o estado real, digest, backup, contrato e canário autorizado.
- Logs, relatórios e commits NÃO podem conter chaves, URL completa de webhook, payload bruto ou telefones dos clientes.

## Comandos locais (uma rodada)

```powershell
git fetch --prune
git status -sb
git rev-parse HEAD
go test -count=1 ./...
go vet ./...
python -m unittest scripts/test_preflight_ctwa_account_switch.py -v
python -m unittest scripts/test_relay_forensic_readonly.py -v
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1
```

Verificar CI no HEAD final. Corrigir apenas falhas determinísticas. Nenhum teste deve enviar mensagens reais, alterar o webhook, reprocessar históricos ou reiniciar os serviços da VPS.

## Dependências e dossiês

- `docs/AUDITORIA_VALIDACAO_CHATGPT_P0_20261031.md` — evidências materiais apresentadas pelo Antigravity sobre o desvio de webhook.
- `docs/CTWA_GREETING_P0_20261029.md` — saudação e conversas iniciadas manualmente.
- `docs/EVOLUTION_RELAY_P0_FORENSIC_HANDOFF_20261031.md` — relay e formato de envelope.
- SmartDialer PR #41 — preservação do Status Engine e dos outros consumidores.

**Entrega:** anexar recibo sem PII na mesma PR #10. `GO_CODIGO` não significa `GO_PRODUCAO`.
