# CTWA 5287 / CRM V4 — Rota canônica AI-first P0

Escopo: botwhatsapp PR #10, branch fix/ctwa-native-proposal-handoff-20261027. Sem CRM code edits, sem Evolution/Status Engine, sem reenvio dos 13 HTTP 422 antigos. Não criar PR paralela nem repetir testes em loop.

## Verdade auditada
- Relatório Antigravity registrou PASS local em d24bf7d792a9b73a61548248b6b5116a85c3a7d0; não garante que este ainda seja o HEAD nem prova deploy.
- Inventário relatado VPS: botwhatsapp-stage-bot-1, projeto botwhatsapp-stage, volume botwhatsapp-stage_botwhatsapp_data, /var/lib/docker/volumes/botwhatsapp-stage_botwhatsapp_data/_data. **Verificar tudo novamente no runtime efetivo.**
- Falhas configuracionais reportadas: BOT_DASHBOARD_TOKEN vazio e EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN ausente.
- O token dashboard é segredo aleatório local. O token Sanctum SÓ o Laravel pode emitir e validar para um User já autorizado com can_send_proposals=true e ability extension:proposal-write. Não inferir ability de tamanho ou formato. Nunca gerar Sanctum aleatório.
- Configuração efetiva do Docker pode vir do Compose environment e sobrescrever env_file, até com string vazia. Editar .env.production não altera container já iniciado. Confirmar sem imprimir valores.

## Protecoes de codigo adicionais desta auditoria
- internal/crm/native.go: Bearer Sanctum somente para HTTPS evydencia.com (localhost apenas nos testes); URL externa indevida e recusada ANTES de qualquer HTTP. Verificar teste TestNativeBlocksUnapprovedCredentialOrigin.
- cmd/botwhatsapp/main.go: BOT_ENABLED=true exige BOT_DASHBOARD_TOKEN com no minimo 32 caracteres e encerra inicializacao se ausente. Essa guarda e intencional; nao suprimir para forcar deploy.
- scripts/preflight-ctwa-production-readonly.sh: verifica formato Sanctum ID|segredo e origem oficial HTTPS, mas NAO prova token emitido no CRM. Para provar token, usar verificador PHP read-only no host Laravel autorizado.


## AUDITORIA POS-DEPLOY 27/10 — Gate real que falta

**Relato do Antigravity:** botwhatsapp-stage-bot-1 recriado na VPS sobre HEAD 13c9f8bccbb57279e58f3805de1370f747455d59; /healthz=200; /dashboard=401 sem auth e 200 com Basic; token Sanctum existente obteve HTTP 422 de validacao de formulario ausente; PASS_CONFIG 16/16; um worker e spool persistente. Tudo isso comprova liveness/autenticacao basica mas NAO processamento de lead, API JSON, TLS do gestor, PDF pela Evolution ou proposta entregue. Nao chamar "100% funcional" ate canario com recibos.

**Nova branch PR #10:** scripts/audit-ctwa-panel-readonly.py e teste offline, acrescidos do gate scripts/validate-ctwa-p0-credentials.sh. Um ajuste no internal/ingress/server.go agora proibe listagem de diretorios da rota /static/ sem bloquear GET do PDF concreto; teste em internal/ingress/server_test.go. **Esses arquivos/ajustes posteriores ao HEAD 13c9 nao estao necessariamente na imagem atualmente em producao**. Nenhum novo deploy esta implicito. Checar Git SHA, imagem/digest e testes antes de novo corte.

### PASS do painel HTTP — nao expor segredo
No host bot, a partir do checkout da branch (e sem imprimir .env):
```bash
python3 scripts/audit-ctwa-panel-readonly.py --env /srv/platform/stacks/botwhatsapp/.env.production
```
O script testa GET /healthz e /readyz; 401 anonimo para /dashboard, /painel e as duas APIs; HTML autenticado, URL absoluta same-origin, HEAD autenticado, JSON real de /api/dashboard/events?limit=10 e /api/dashboard/contacts. Apenas quantidades agregadas sao exibidas. Nenhum POST, webhook ou envio real; senha Basic so em memoria, nunca em argumento shell ou log; redirects e proxies desativados. O 200 HTML anterior nao comprova estes checks.

**No endereco HTTPS privado REAL do gestor**, confirmar hostname, proxy Caddy e rotas na raiz; repetir opcionalmente com --base-url https://HOST_VALIDADO_PRIVADO sem usar hostname inventado. Antes de apontar a ferramenta para um host remoto, confirmar que ele pertence a infraestrutura autorizada (o script transmite Basic Auth a origem indicada). HTTPS certificado valido; jamais usar Basic Auth por HTTP remoto. Inspecionar browser desktop e 390px, inclusive refresh, filtros e carregamento da secao de contatos. As requisicoes fetch de JS sao same-origin e usam new URL('/api/dashboard/...', window.location.origin): isso so funciona se proxy expoe /api/dashboard/* no mesmo origin e na raiz. Verificar CSP, no-store e ausencia de erros 401/404/CORS/JS no DevTools.

### PDF e audio via Evolution: NAO inferir por healthz
Em producao o codigo do bot escolhe BOT_ADS_DOCUMENT_URL antes de BOT_ADS_DOCUMENT_FILE para enviar o PDF. URL padrao http://bot:8788/static/Proposta_Natal.pdf; a Evolution precisa resolver "bot" na rede Docker compartilhada e baixar o PDF. Conferir DNS e rede real, sem criar um segundo worker:
1. Identificar o container Evolution pelo inventario atual, nao nome presumido.
2. Fazer GET/HEAD **dentro da rede/contêiner Evolution**, sem token comercial, ao endereco exato BOT_ADS_DOCUMENT_URL efetivo; exigir 200 e arquivo PDF valido. Se imagem nao tiver curl/wget, classificar SKIP e usar probe de rede segregado autorizado, sem mexer no processo Evolution.
3. Confirmar API Evolution /instance/status conectada e instancia Ads UUID 5287 correta, SOMENTE GET autenticado por metodo privado seguro e resultado sanitizado; nao confundir "Connected" momentaneo com entrega.
4. Audio OGG Opus e tamanho conferidos na montagem real do bot. PDF/AUDIO presente no host nao implica GET alcançavel pelo Evolution. A rota /static/ nao pode revelar indice de pasta e Caddy nao deve expo-la publicamente; o arquivo concreto deve continuar disponivel internamente.

### Entrada duravel e ultimo gate
1. Confirmar painel JSON vazio ou historico real, sem vazamento de PII; comparar contagem com journal real e quantidade de tasks; consultar /readyz.
2. Um canario novo previamente consentido no 5287: webhook -> journal -> outbox -> texto/audio/PDF Evolution aceita -> CRM agenda com UUID nativo (idempotencia) -> verificar dispatch no CRM -> verificar processamento posterior (Z-API ou outro canal do CRM).
3. Sem confirmacao do dispatch em CRM e da cadeia Evolution, status e **GO somento para teste/canario**, nao GO TOTAL de funil. Nenhum replay dos 13 leads antigos.
4. Se houver erro ou UNKNOWN, nao rediscagem automatica. Conservar spool/claims/opt-outs e rollback apenas do bot usando backup/digest confirmado.

### CI e mudancas apos deploy
O job no SHA 13c9 foi consultado como "cancelled"; o PASS anterior refere-se a SHA distinto. Uma PR sem divergencia da main nao garante imagem identica. No novo HEAD, executar teste focado unico e CI; nunca aplicar git reset/clean ou trocar Compose/volume para forcar PASS. O runtime implantado permanece em 13c9 ate evidencia contraria por SHA/digest.

## Fase 1 — Avell / Antigravity (sem rede produtiva)
Na PR #10: git fetch, verificar HEAD/diff, atualizar com fast-forward, sem reset. Rodar gate local já existente scripts/validate-ctwa-handoff-local.ps1 no HEAD final. Para novos scripts, bash scripts/validate-ctwa-p0-credentials.sh executa preflight falso, unittest privado, php -l quando disponível. Não usar -Preview se já homologado o mesmo HTML.

## Identidade e instalacao efetiva confirmadas (27/10)
- Conector CRM LIVE verificou operador autenticado id=2 com ability extension:proposal-write e recurso de propostas habilitado. A ability NAO substitui a checagem real de can_send_proposals no banco pelo emissor.
- A arvore de deploy do CRM esta documentada em hypeneural/evydencia-crm/docs/vps-deploy-contract.md: /var/www/evydencia/current apontando para release ativa e /var/www/evydencia/shared/.env. Verificar o symlink e vendor/autoload.php antes de executar o emissor; nunca assumir que bot e CRM ficam no mesmo host.
- CRM composer.json declara laravel/framework:^10.10 e laravel/sanctum:^3.3. Checar versoes reais no host; referencia oficial: https://laravel.com/docs/10.x/sanctum. createToken(name, abilities, expiresAt) e suportado.
- O token CRM LIVE atual tem MAIS abilities que o necessario; NAO reutilizar nem extrair o segredo da sessao do MCP. Emitir token separado com ability unica.
- O provisionador do dashboard pode escrever o segredo e sair com codigo !=0 porque o Sanctum permanece faltando. Esse exit significa NO_GO GLOBAL, nao desfaz a senha gravada. Nao rodar --rotate-dashboard novamente so para corrigir esse exit.
- Relato do Antigravity de curl e /etc/hosts inconclusivo: repetir apenas verificacao HTTPS/DNS por diagnostico sanitizado, nao inferir rede saudavel pelo retorno do CRM no conector.
- Ao transferir segredo e aplicar .env, verificar YAML efetivo e overrides. environment do Compose tem precedencia sobre env_file mesmo quando vazio (Docker docs).

## Fase 2 — segredos privados (sem chat, sem Github)
Provisionamento de painel NO HOST REAL DO BOT, após verificar qual arquivo .env é efetivo:

  python3 scripts/configure-ctwa-secrets.py /srv/platform/stacks/botwhatsapp/.env.production

O comando padrão é um plano sem escrita. Para instalar a senha aleatória local com backup privado e troca atômica:

  python3 scripts/configure-ctwa-secrets.py /srv/platform/stacks/botwhatsapp/.env.production --apply

Caso Sanctum não exista, sai NO_GO mas pode ter gravado a nova senha do painel. NÃO rotacionar painel existente sem autorização específica.

Em CRM Laravel REAL autorizado: verificar usuário existente com can_send_proposals=true e identificar USER_ID inequivocamente. Copiar os scripts PHP publicados na PR #10 para diretório 0700 fora da árvore do CRM e executar sem expor saídas secretas:

  php /private/issue-ctwa-sanctum-on-crm.php /var/www/evydencia/current USER_ID /private/ctwa-sanctum.token
  php /private/verify-ctwa-sanctum-on-crm.php /var/www/evydencia/current /private/ctwa-sanctum.token

Emissor altera APENAS a tabela Sanctum de tokens do CRM, requer usuário com permissão comercial e emite ability mínima com expiração em 90 dias (cobre Natal 2026; revogar após a temporada); não muda código, fluxos ou pedidos. Verificador somente lê DB e confirma hash, ability, usuário e validade sem emitir proposta. Exige acesso e identidade de usuário autorizados; se faltar, BLOCKED, não usar token antigo 416.

Transferir o arquivo com SCP/SSH criptografado em rede privada entre hosts autorizados, chmod 0600. Não passar token em linha de comando, URL, chat, issue ou log. No bot:

  python3 scripts/configure-ctwa-secrets.py /srv/platform/stacks/botwhatsapp/.env.production --sanctum-file /private/ctwa-sanctum.token --apply

## Fase 3 — preflight e runtime
Após revisar projeto/volume/arquivo reais:

  bash scripts/validate-ctwa-p0-credentials.sh --env /srv/platform/stacks/botwhatsapp/.env.production --spool /var/lib/docker/volumes/botwhatsapp-stage_botwhatsapp_data/_data

Somente PASS_CONFIG, PASS_SANCTUM_CRM, HEAD revisado e único worker autorizam considerar release. Preflight não chama CRM/Evolution. Conferir compose com saída **redigida** para não expor segredos; não fazer docker compose config bruto nem docker inspect Config.Env em logs.

Caddy/TLS privado obrigatório para painel; manter origens/volume e rede reais. Backup comprovado, UID e rollback de imagem. Nunca limpar spool, rodar down -v, reiniciar outras stacks ou usar compose.staging.yaml cegamente. Inconsistência entre env_file e environment sobrescrevendo token com vazio => NO_GO.

## Fase 4 — deploy e canário (somente se gates reais PASS)
Usar os arquivos Compose já efetivos, atualizar SOMENTE serviço bot, preservar volume e manter um worker. Testar healthz/readyz + painel privado autenticado. Um único canário de contato consentido NOVO no 5287: ingresso, Evolution aceitou áudio/PDF/texto, CRM retornou dispatch_uuid, consulta comercial confirmou agendamento. HTTP 201 scheduled não prova entrega. 401/403/422/429/5xx/UNKNOWN exigem investigação sem reenvio cego. Não tocar nos 13 contatos históricos.

## Resultado para gestor — até seis linhas
PR/branch/HEAD + PASS/FAIL/SKIP reais; painel configurado?; Sanctum REAL emitido/verificado?; PASS_CONFIG?; imagem/volume/worker efetivos?; canário real e GO/NO-GO ou bloqueio preciso. Não divulgar tokens ou arquivos privados.

Referências: https://laravel.com/docs/12.x/sanctum e https://docs.docker.com/compose/how-tos/environment-variables/envvars-precedence/
