# Auditoria forense final — BotWhatsApp, GO 0.7.2, CRM 416 e relay independente

**Realizado em 22/10/2026.** Referência inicial: `main=0bcf0c3f38e9cd430d34a8060f9466b08b83936e` e relatório final AntiGravity. A atualização está numa branch de PR, não em produção. Verificações: repositório GitHub remoto, CI, documentos oficiais Go/Alpine/Docker, código Evolution Go da tag 0.7.2 e contrato CRM já confirmado em revisão anterior. **Não foi executado SSH nem POST externo neste ciclo**.

## Inventário de estrutura
- `cmd/botwhatsapp`: HTTP ingress e worker único com flags OFF; `cmd/simulate`: fixtures offline; `cmd/webhook-relay`: sidecar OFF novo.
- `internal/core`: parser/normalização; `instances`: UUID/role/claims; `ingress`: eventos autenticados com ACK; `outbox`: journal e step FSM; `evolution`: cliente API; `crm`: adaptação CRM 416; `relay`: spool status-first novo.
- `config/`: caption e exemplo editorial não lido diretamente pelo Go; `.env.example`: contrato real runtime; `deploy/`: staging sem host ports.
- `.agents/skills/`: skills orientadas a tarefas e referências de GO, CRM, canário, performance e validação local.
- `scripts/`: `vps-readonly-inventory.sh` não reinicia serviços nem lê .env; novo `validate-local.ps1` só testa localmente.
- `Dockerfile`, `Dockerfile.relay`, `.github/workflows/go-test.yml`: compilador atualizado; sem Go instalado no host Contabo.
- Documentação antiga foi mantida para rastreabilidade, mas **`README`, `AGENTS`, `START-HERE`, `INDEX` são os guias atuais**, evitando orientação contraditória.

## Constatações técnicas (evidências separadas)
**GitHub verificado:** `main` tinha PRs #1–#5 consolidadas e Actions PASS na versão anterior. A versão instalada da GO 0.7.2 é derivada do relatório do AntiGravity; patch local in-tree WebP não foi baixado nem recompilado nesta rodada. A instância `anderson` está vinculada ao webhook do status-engine no inventário reportado, não houve nova consulta ao PostgreSQL neste ciclo.

**API Evidência CRM (código):** campanha padrão **416 SEM NOME**, `POST /api/campaigns/schedule/execute`; `Authorization` com token interno raw, não `Bearer`. `contacts` é número nacional DDD+10/11 dígitos sem 55, `instance` é ID numérico do CRM. Retorno `CampaignScheduledResource` normalmente traz `id/campaign/status/instance` na raiz (Laravel `JsonResource::withoutWrapping`). `GET /api/campaigns/schedule/search` anterior teve timeout, sem prova de entrega e sem segundo POST. Status `scheduled` não é `delivered`.

**Fontes oficiais:** Go 1.23 saiu do suporte após Go 1.27. Go 1.27.3, publicado 19/10/2026, inclui correção de segurança de x509. Alpine 3.24.2 era versão de manutenção atual da série 3.24 em 22/10/2026. Docker Compose suporta comunicação interna DNS de serviços, redes compartilhadas externas, `expose` sem `ports`, read-only rootfs e secrets (com restrições de runtime).

## Melhorias nesta branch
- **Segurança de build:** Docker builder `golang:1.27.3-alpine3.24`, runtime `alpine:3.24.2`, com `ca-certificates` para o HTTPS CRM e `tzdata` para `America/Sao_Paulo`. CI usa Go 1.27.3, `go vet`, `go test -count=1 ./...` e actions com Node 24. `go.mod` mantém Go1.23 como versão de linguagem mínima para compatibilidade de testes Windows; o binário Docker NÃO compila com 1.23.
- **Relay seguro sem mutação:** novo `cmd/webhook-relay` e `internal/relay`: aceita evento só com segredo de rota + UUID, persiste o JSON antes de ACK, repassa primeiro ao status-engine original, depois ao bot, ACKs separados duráveis, tombstones 7 dias, scrub raw no sucesso, backlog limitado por contagem. Zero troca de porta 8050/8051.
- **Staging Compose:** dois containers novos, `RELAY_ENABLED=false`, `BOT_ENABLED=false`, `BOT_OUTBOUND_ENABLED=false`, `BOT_OPERATIONAL_AUTHORIZATION=false`; sem ports host; `cap_drop: ALL`, `no-new-privileges`, `read_only` e volumes isolados. Nenhum `up` foi executado.
- **Operação simplificada:** `scripts/validate-local.ps1` para Antigravity rodar exatamente uma verificação local e relatório, sem instalar nada na VPS; docs e `AGENTS` despoluídos e índice de histórico.

## Riscos importantes ainda não eliminados
1. **Cutover:** auditoria do código real do `whatsapp-status-status-engine-1` é necessária para garantir que tolera retries e campos recebidos. O relay NÃO garante entrega exactly-once quando o sink dá 2xx e cai antes da gravação ACK. Sem verificar status-engine, **produção contínua BLOQUEADA**.
2. **Quota em BYTES:** o relay tem limite por número de registros, NÃO por bytes. Payload máximo 256KiB pode rejeitar eventos maiores; calibrar com observação de tamanhos reais e configurar limite de armazenamento em bytes e monitoramento, sem PII nos logs, antes de cutover.
3. **Segredos no payload**: a GO pode enviar `instanceToken` no envelope. Relay guarda payload bruto somente até concluir os dois destinos, em volume privado 0600. Essa superfície temporária deve ser revisada segundo política de proteção/criptografia/backups.
4. **Outbox**: ainda exige única réplica e storage local exclusivo. Retenção existe para tarefas confirmadas/suprimidas; ingress spool/locks/claims e estados `uncertain` não devem ser apagados sem reconciliação, com risco de acúmulo.
5. **Mídias e contas:** áudio OGG Opus/arte reais devem ser aprovados e montados; número de saudação ainda não está comprovadamente pareado. `CRM instance=1` foi relatado ativo, não está codificado como default.
6. **Implantação:** ausência de portas host no Compose protege exposição, MAS `0.0.0.0` é necessário DENTRO de contêiner para que outros containers o vejam na bridge. Docker interno não é HMAC. Restrinja acesso às redes e caminhos.
7. **Sinalização de entrega**: confirmação `data.Info.ID` GO prova resposta da API; `crm schedule id` prova agendamento. Prova de leitura/entrega depende de receipts posteriores não incluídos nesta integração.

## Verificações permitidas e pendências do Antigravity
- LOCAL: `git status -sb`, `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-local.ps1`. O script não faz POST externos.
- NÃO executar `docker compose up/down`, deploy, mudança de webhook, reiniciar serviços, parear sessão ou reenviar campanha canário nesta missão.
- Analisar CI do head da PR. Se PASS, integrar hotfix depois de revisão e manter flags OFF.
- PROXIMA FASE somente após autorização do gestor: auditoria status-engine, preflight de rede/read-only, medir tamanhos de evento, limitar uso de disco, pareamento saudação/mídia e plano de rollback/monitoramento. Separar confirmação comercial do agendamento anterior de eventual novo teste.
