# Prompt Antigravity — validar localmente o funil CTWA Natal 2026

**Repositório:** `hypeneural/botwhatsapp`.
**Meta:** usar a PR de código desta tarefa, não recriar motor nem repetir deploy/canário.

## Passo 1 — sincronizar e proteger trabalho local
- Execute `git status -sb`; se houver alterações, preserve-as. Não `reset --hard`, `clean -fd`, force-push.
- Faça `git pull --ff-only origin main` SOMENTE quando a PR estiver integrada; caso contrário, examine o diff da PR sem sobrescrever arquivos locais.
- Confira `AGENTS.md`, `docs/CTWA-ROUTING-20261022.md` e `.agents/skills/ctwa-ads-funnel/SKILL.md`.

## Passo 2 — conferir código e configuração
- Frase do anúncio real: `Quero conhecer os pacotes do Primeiro Natal em Tijucas`, em `config/ads-triggers.txt` (uma frase inteira por linha; futuras variantes podem ser adicionadas). O arquivo tem precedência sobre `BOT_ADS_TRIGGERS`; reiniciar bot autorizado para recarregar. NÃO usar regex, wildcard, contains.
- `core.Parse` lê `extendedTextMessage.text`; `@lid` nunca vira número CRM. `instanceId` na configuração corresponde ao **Ads GO** conectado, não ao CRM.
- `instances.Registry.Decide` deve repassar `AdsFallback=true`. Mensagem sem match no 48 3220-0440 → saudação com imagem+caption ou fallback texto/menu, com cooldown. Opções 1–5 continuam funcionando.
- Mensagem de anúncio com match → texto `{{nome}}` seguro, OGG PTT, imagem opcional/caption, vídeo MP4 opcional/caption, agendamento final CRM **416 SEM NOME**; `PreviousTaskID` força a ordem das tentativas.
- Número 48 99642-5287 (ainda não pareado): role greeting, token próprio; sem automação CRM.
- Revisar que o limite de webhook relay/bot é 1 MiB, com 413 quando excede; sem armazenar thumbnail/contexto bruto desnecessariamente em logs.

## Passo 3 — validação única sem efeitos externos
Execute UMA vez:
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-local.ps1
```
Verifica vet/test/build e Compose config. Os testes foram acrescentados em `internal/core/ctwa_test.go`, `internal/instances/ads_fallback_test.go`, `internal/outbox/ctwa_flow_test.go` e `cmd/botwhatsapp/triggers_test.go`, com fixture SINTÉTICA.

Se falhar pacote específico, corrigir somente a causa e refazer suite completa uma vez no final. A CI Go e build de containers também precisam estar PASS.

## Proibições
- Não enviar campanha real, áudio ou vídeo, não alterar webhook da instância `anderson` e status-engine, não parear WhatsApp nem reiniciar GO/CRM/Caddy/VoIP/Contabo.
- Não incluir dados/identificadores reais do contato auditado, mídia de clientes ou payload integral do relatório no Git. O anexo inclui um `instanceToken` não sanitizado: recomendar ao operador verificar/rotacionar se permanecer válido, mas não efetuar rotação durante esta validação.
- Não editar `hypeneural/evydencia-crm`, nem executar POST para `/api/campaigns/schedule/execute`.

## Entrega do Antigravity
Responder com SHA final, `go version`, PASS/FAIL da execução única, verificação de que o gatilho CTWA e fallback foram cobertos em testes, lacunas de assets/pareamento e status do gate de fan-out. Sem relatório repetitivo de 50 páginas.
