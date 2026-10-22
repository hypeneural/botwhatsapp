# AGENTS.md — Evyd WhatsApp Bot

## Instruções canônicas para Antigravity (inclusive Gemini Flash Lite)

Leia este arquivo, depois `docs/START-HERE.md` e as skills em `.agents/skills/`. Execute UMA etapa por vez e registre CONFIRMADO / NÃO CONFIRMADO / RISCO / PRÓXIMA AÇÃO. NÃO invente versão, UUID, endpoints, IDs ou resultado de testes.

### Regras absolutas
- Alvo de escrita: somente `hypeneural/botwhatsapp`. `hypeneural/evydencia-crm` é **SOMENTE LEITURA**: sem commits, migrations, alterações de dados ou deploy.
- A VPS Contabo hospeda OUTROS SERVIÇOS. Auditoria inicial SOMENTE LEITURA; proibido reiniciar, parar, instalar, subir containers, alterar proxy/firewall/DNS, webhooks, instâncias, tokens ou conexão WhatsApp.
- Nunca exibir, copiar ou versionar .env, tokens, chaves SSH, credenciais, dados de clientes, conversas ou logs brutos.
- Não usar n8n, Typebot, OpenClaw, Chatwoot nem Evolution Node V2. Stack existente: **Evolution Go** + serviço complementar pequeno.
- Proibidos envios de mensagens, áudio, imagem e campanhas reais sem autorização explícita e canário limitado. Flags começam OFF.
- Não alterar webhooks globais/por instância antes de mapear consumidores da VPS. Não reconfigurar `/instance/connect` durante preflight.
- Webhook não tem HMAC confiável no campo `instanceToken`; usar acesso interno ou segredo próprio e não registrar esse valor.
- CRM POST com resultado incerto: não repetir automaticamente. Proteger idempotência fora do CRM, que não será modificado.
- Testes locais com fixtures e mocks direcionados; evitar baterias e deploy indiscriminados.

### Identidade comercial dos dois números (não são UUIDs)
1. WhatsApp **48 3220-0440**, E.164 `554832200440`: recebe anúncio Meta Ads; reconhecer a FRASE EXATA ainda pendente, responder texto, enviar áudio gravado e solicitar campanha Natal 2026 pela API existente do CRM. Mensagem não reconhecida pode receber menu.
2. WhatsApp **48 99642-5287**, E.164 `5548996425287`: saudação com UMA imagem e caption/menu numerado; controle de cooldown, opções 1–5, pausa para humano.

Não associar número ao UUID Evolution Go sem prova, nem UUID Evolution ao `message_instances.id` do CRM.

### Ordem obrigatória
1. `vps-readonly-audit` → inventário seguro e dependências na VPS, sem mutações.
2. `evolution-go-preflight` → identificar versão/digest e contrato realmente instalado.
3. `crm-readonly-contract` → validar a API já existente, somente em leitura.
4. Preencher pendências de `docs/START-HERE.md`; operar com `config/instances.example.yaml` (OFF).
5. `two-bot-workflows` → implementar e testar exclusivamente neste repositório; não implantar sem autorização.

### Fontes
- https://github.com/evolution-foundation/evolution-go
- https://github.com/6justgotme/evolution-go-skill
- https://github.com/estevao-tarifa/evo-chatwoot-bridge-go (MIT; preservar licença ao adaptar trechos)
- https://github.com/hypeneural/evydencia-crm (somente leitura)


## Status de código, 21/10/2026
- Auditoria forense: `docs/FORENSIC-AUDIT-2026-10-21.md` (inventário da VPS é RELATO do AntiGravity, não nova inspeção remota).
- Base escrita: `internal/core` (roteamento), `internal/workflow` (plano/cooldown), `internal/evolution` (cliente HTTP não conectado), `internal/crm` (contrato/cliente HTTP não conectado), `internal/ingress` (webhook com dedup e spool em **prévia**), `cmd/botwhatsapp`.
- Para construir/testar: `go test ./...`. `Dockerfile` multi-stage compila sem Go no host; não fazer deploy nesta fase.
- ATENÇÃO: código atual NÃO ENVIA mensagens nem chama CRM. Nunca interpretar `readyz` como aprovação comercial; apenas confirma receptor preview.
- Sem worker/outbox com recuperação e lead claims, NÃO ligar nenhum adaptador aos webhooks de produção. Próximas etapas: `docs/IMPLEMENTATION-NEXT.md`.
- `WEBHOOK_URL` (não `GLOBAL_WEBHOOK`) é a variável oficial do webhook global; ausência ainda NÃO CONFIRMADA na VPS.
- `127.0.0.1:14000` identifica a API no HOST, não dentro de novos containers.
- Antes de POST CRM real é obrigatória instância de saída numérica explícita (>0), além da confirmação da campanha 415 ou 416 em produção.
- Skills novas: `.agents/skills/evolution-go-v072-adapter/SKILL.md` e `.agents/skills/bot-contract-tests/SKILL.md`. Essas skills CURADAS apontam para código upstream e não instalam software externo.

## Ajuste comercial oficial — integração CRM campanha 416
- **Padrão irrevogável dos bots atuais:** `campaign=416` (Natal 2026 **sem nome**); não escolher automaticamente 415 por `PushName`. A opção 415 só pode ser ativada no futuro por decisão expressa.
- Contratos/retornos reais: `docs/CRM-API-416-PLAYBOOK.md`; `JsonResource::withoutWrapping` coloca `id` na RAIZ do JSON de sucesso; `GET /api/campaigns/schedule/search` permite leitura, mas não autoriza refazer POST incerto.
- Cliente CRM `internal/crm/client.go`: instância de saída numérica >0, HTTPS para endereço não-local, token raw `Authorization`, `type:campaign`, `use_leads_system:false`. Nunca escrever no CRM.
- API Evolution Go `internal/evolution/client.go`: `Status`, `SendTextWithID`, `SendMediaWithID`; ler `docs/EVOLUTION-GO-API-FUNCTIONS.md`. Não reutilizar token CRM na Evolution Go.
- Como AntiGravity pode trabalhar na main simultaneamente, as alterações deste ciclo foram isoladas em `feat/crm-416-contract-20261021`. Integrar por PR após validação de conflitos.\n
## Canário real relatado — verificação obrigatória antes de produção
- Consulte `docs/AUDIT-CANARY-AND-MULTIINSTANCE-2026-10-21.md` e `docs/HANDOFF-ANTIGRAVITY-CANARY.md` antes de retomar execuções.
- Os commits do canário `16514e4`, `10348a6`, `ae40f97` foram relatados como locais no Windows; não foram encontrados em `main` remota na auditoria. Auditar o código EFETIVAMENTE usado quando estiver publicado.
- O relatório do canário informa rota CRM `/api/campaigns/schedule`, mas o código CRM define `POST /api/campaigns/schedule/execute`. Nunca presumir que agendamento foi feito por rota incorreta.
- A instância `anderson` está/estava vinculada a um webhook de status. **Não substituir esse webhook para testes ou produção** antes de plano de fan-out durável e autorização.
- Snapshot UI de webhook restaurado não comprova persistência nem processamento; confirmar por leitura técnica sem alterar config.
- Para N instâncias, usar `internal/instances/registry.go` (ainda não conectado ao worker local), skill `multi-instance-contract`, claim GLOBAL por tenant/telefone nacional/campanha416/data BRT. Token e webhook secret exclusivos por instância.
- `0.0.0.0:8788` abre escuta em todas interfaces do Windows; conexão via Tailscale não garante exclusividade da bind. Restringir interface/firewall conforme rede autorizada.
- Não repetir um canário comercial só porque testes de dedup passaram. CRM id = agendamento; Evolution GO id = resposta API, não status delivered/read.

## Prioridade de integração pós-canário — 22/10/2026
- A PR #4 contém a consolidação do worker real, registry e contrato CRM 416; esta nova branch de correções é empilhada **sobre PR #4**, não sobre `main`.
- Leia `docs/FORENSIC-PR4-20261022.md`, `docs/PROMPT-ANTIGRAVITY-PR4-GUARDS.md` e `.agents/skills/outbox-production-audit/SKILL.md` antes de efetuar merge ou testar.
- Risco crítico corrigido: não enviar HTTP 202 de webhook antes de persistir TODAS as tarefas do outbox. `CommitEventTasks` grava lote com fsync e marca evento ao final.
- `BOT_OUTBOUND_ENABLED` e `BOT_OPERATIONAL_AUTHORIZATION` começam false. Exigir URL Evolution Go, token CRM e token por instância explícitos; nenhum áudio/imagem placeholder pode sair.
- Opt-out PARAR, pausa humana e cooldown são salvaguardas persistentes; falha em leitura do estado do contato é bloqueio de envio, não novo contato.
- Um único bot/worker/réplica deve possuir o `BOT_DATA_DIR`. Não montar o mesmo journal gravável em dois processos e assumir transação distribuída.
- Manutenção periódica só trata tarefas concluídas; retenção de ingress spool, locks de events/leads, contatos e uncertain NÃO está resolvida.
- O webhook `anderson` segue reservado ao motor de status. Implementar relay/fan-out autorizado em trabalho separado ANTES de deploy. Nenhuma mudança operational nesta PR.
- Não repetir o teste CRM de campanha 416 para atestar a API; canário anterior foi relatado e busca GET ficou inconclusiva.
