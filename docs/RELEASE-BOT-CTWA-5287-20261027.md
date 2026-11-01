# Release controlado — BotWhatsApp CTWA 5287 → CRM Natal V4

> **ATUALIZAÇÃO P0:** sequência operacional do Antigravity condensada em [CTWA-P0-EXECUCAO-AI-FIRST.md](CTWA-P0-EXECUCAO-AI-FIRST.md). Provisionador seguro scripts/configure-ctwa-secrets.py; emissor/verificador Sanctum somente no host CRM real; teste focal scripts/validate-ctwa-p0-credentials.sh. O pré-flight confere apenas configuração/formato, não autenticação real. Nenhum token vai para chat/Git. O bloco operacional abaixo é histórico até reconferência dos paths do Compose e volume efetivo.\n\n
**Data:** 27/10/2026 · **Repositório:** `hypeneural/botwhatsapp` · **PR:** #10  
**Escopo autorizado para esta documentação:** preparar e validar o código, executar diagnóstico somente leitura e orientar um canário restrito. **Não autoriza alterar o CRM, Evolution, Status Engine ou dados históricos.**

## Estado comprovado e limitações

- `main` do bot auditada: `6af807225704c710b68a5a185c0725b26369cb3f`.
- Antigravity validou o HEAD `ad6a736c0204de76878d5c89ff24f25d4a1d01d1`: Go Test, Vet, Build e Node local PASS, Chrome desktop/390 px PASS; CIs do mesmo SHA passaram.
- Após isso, a PR #10 recebeu ajustes forenses adicionais: classificação CRM versus bloqueio do bot, atenção a leads sem encaminhamento, último status CRM por contato, testes de regressão, ordenação temporal e preflight privado de produção. **Exigir CI e PASS local no HEAD final antes de liberar.**
- Nenhum agente nesta auditoria implantou o código na VPS ou fez envio real; artefatos de preview são todos sintéticos.
- Endpoint CRM verificado no código: `POST /api/v1/extension/proposals/schedule`, Sanctum `extension:proposal-write` + usuário autorizado pela política `send`. Recibo `dispatch_uuid` não significa entrega no WhatsApp.
- Os 13 HTTP 422 anteriormente reportados devem ser reconciliados, nunca automaticamente reenviados.

## Gates P0, sempre nesta ordem

### Gate A — verificar PR e rodar local (Antigravity Windows)

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1 -Format -Preview
```

O script verifica branch/SHA, normaliza **somente** os Go da PR, Go Test, Vet, build, sintaxe JS e, se Git Bash existir, executa simulação offline de preflight de produção com segredos fictícios. Se Bash/Node não existirem, o resultado é **SKIP explícito**, não PASS. Encerrar preview com Ctrl+C; preservar diff local para revisão antes de push. Registrar `git status --short`, SHA, gate, screenshots do preview dos **sete cenários fictícios**, incluindo bloqueio local e lead sem tarefa, e resultado em 390px. Não substituir o histórico de produção por dados de demo.

### Gate B — inventário read-only na Contabo

Sem fazer stop/start/redeploy, identificar e registrar **internamente sem segredos**:

- projeto/arquivo Compose **real** em uso, nome do serviço e container do bot; `docker compose ls`, `docker ps`, nome do volume/bind mount e UID da aplicação;
- imagem efetiva e digest/ID de rollback, portas privadas, rede Evolution, Caddy e webhook em uso, atual health e `/readyz`;
- caminho real de `BOT_DATA_DIR`, se a outbox, o journal, os eventos de deduplicação, o opt-out, os claims e os registros incertos persistem no **mesmo volume**;
- backup consistente do journal (volume ou bind mount real), com integridade e local de restauração comprováveis, **sem exibir números, nomes nem tokens nos relatórios**;
- nenhum outro worker usando a mesma outbox. Duas instâncias ativas sobre o mesmo spool são **NO-GO**.

**CRÍTICO:** `deploy/compose.staging.yaml` é só laboratório: contém `BOT_ENABLED=false`, `BOT_OUTBOUND_ENABLED=false`, `BOT_OPERATIONAL_AUTHORIZATION=false` e cria volume de projeto staging. **NUNCA** subir esse arquivo para substituir o serviço real, nem criar outro volume vazio. Nada de `docker compose down`, `down -v` ou limpeza do journal. Docker Compose pode recriar contêineres conservando volumes montados, mas isso somente é seguro quando o projeto/montagem são os mesmos e foram verificados.

### Gate C — validar configuração privada sem imprimir credenciais

Somente com caminho do arquivo de ambiente efetivo e **caminho do journal efetivo** já comprovados:

```bash
bash scripts/preflight-ctwa-production-readonly.sh "/caminho/privado/.env.production" "/caminho/real/do/spool"
```

Esse script **não importa nem imprime** segredos, não modifica arquivos e retorna `NO_GO` se o 5287, os três flags operacionais, token Sanctum, token do painel, chave Evolution, secret webhook, caminhos de mídia, HTTPS do CRM ou volume estão ausentes/inadequados. Valida arquivo de configuração candidato, não o container em execução; comparar **também** com a configuração efetiva sem emitir valores.

O deploy requer:
- `BOT_ADS_PHONE=5548996425287`; `BOT_GREETING_PHONE` e `BOT_GREETING_INSTANCE_UUID` vazios para operação de papel único; UUID Evolution Ads efetivo correto;
- `BOT_ENABLED=true`, `BOT_OUTBOUND_ENABLED=true`, `BOT_OPERATIONAL_AUTHORIZATION=true`;
- token Sanctum **distinto do token raw antigo**, limitado a `extension:proposal-write`, usuário válido autorizado pelo `ProposalFlowPolicy` e fluxo Natal pronto;
- `BOT_DASHBOARD_TOKEN` forte, Caddy HTTPS **privado**, sem autenticação por credenciais em URL. `new URL(path, window.location.origin)` resolve URL do `fetch`, mas não autoriza URLs `user:pass@host` na produção;
- confirmar áudio OGG e PDF montados/servidos de fato a partir do container, conferindo os `media` paths e o endpoint estático, não somente a variável definida;
- `BOT_ASSETS_DIR`, gatilho exato `config/ads-triggers.txt`, recepção de webhooks e clock BRT preservados.

### Gate D — corte mínimo, um worker e canário autorizado

Somente depois de A/B/C PASS, repositório revisado, backup e decisão operacional GO:
1. Registrar identificação exata da imagem antiga e volume; preservar rollback.
2. Construir imagem da PR #10 usando Dockerfile atual, **sem trocar mounts/redes/webhooks** nem alterar serviços Evolution/CRM/Status Engine.
3. Fazer troca **somente do serviço real do bot** e garantir um único worker ativo por journal. Usar Compose **existente**, nome real do serviço; `docker compose ... config --quiet` antes. Não publicar portas novas.
4. `/healthz` e `/readyz` saudáveis; painel GET privado autenticado e `/api/dashboard/events` retornando somente os registros reais do mesmo volume, sem 500/401 inesperados.
5. Um **novo contato de teste autorizado**, jamais alguém dos 13 pendentes: verificar gatilho no 5287, retorno de aceite da Evolution para texto/áudio/PDF, um único `request_id`, aceite nativo `dispatch_uuid` no CRM e estado `crm_aceito` no painel.
6. Confirmar **no CRM** o despacho e a continuação/entrega posterior na sua própria trilha. O bot não consegue provar entrega só pelo `dispatch_uuid`.
7. Reinspecionar ingressos novos no painel, `incerto`, `bloqueado_local`, `crm_rejeitado`, `sem_tarefas`, `pendente` e o journal persistente. Qualquer duplicidade, 422 novo sem explicação, 5xx sustentado, painel sem acesso privado ou mudança do volume implica **NO-GO/rollback do binário do bot apenas**.

### Gate E — conciliação de histórico

Separadamente do canário novo, conferir os 13 HTTP 422 em **somente leitura**, por contato e request ID no bot e no CRM. Identificar quem já recebeu proposta, quem comprou, opt-outs, casos ambíguos e quem continua elegível. Proibir loop de replay do backlog; qualquer recuperação requer lista aprovada, plano anti-duplicidade e disparo individual explícito. A lista de 19 prints não equivale ao conjunto dos 13 erros.

## Critérios objetivos de GO/NO-GO

**GO possível:** CI do HEAD PASS; script Windows PASS; live config/instância/volume confirmados; Caddy privado; credencial nativa autorizada; apenas um worker por journal; backup/rollback prontos; canário único com CRM dispatch confirmado e sem regressão no 5287.

**NO-GO obrigatório:** token nativo ausente; número/UUID errados; conteúdo de mídia inacessível; `BOT_OPERATIONAL_AUTHORIZATION=false`; volume do journal mudou; duas instâncias concorrentes; só preview sintético sem canário; divergência nos estados; CI falhando; emissão de credenciais em logs; necessidade de modificar o CRM para contornar 422.

**Responsabilidades:** Antigravity faz validações locais e inventário/canário do bot apenas conforme autorização; mudanças na lógica/deploy do `evydencia-crm` permanecem fora de escopo e devem ser escaladas ao Codex responsável. Rejeições do CRM são tratadas como regra comercial, não desbloqueadas por API batch.

## Referências oficiais

- Docker Compose `up`: https://docs.docker.com/reference/cli/docker/compose/up/
- Docker Compose volumes e `down -v`: https://docs.docker.com/reference/cli/docker/compose/down/
- MDN Fetch credentials: https://developer.mozilla.org/en-US/docs/Web/API/Request/credentials
