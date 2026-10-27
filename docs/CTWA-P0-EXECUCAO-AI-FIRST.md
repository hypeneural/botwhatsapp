# CTWA 5287 / CRM V4 — Rota canônica AI-first P0

Escopo: botwhatsapp PR #10, branch fix/ctwa-native-proposal-handoff-20261027. Sem CRM code edits, sem Evolution/Status Engine, sem reenvio dos 13 HTTP 422 antigos. Não criar PR paralela nem repetir testes em loop.

## Verdade auditada
- Relatório Antigravity registrou PASS local em d24bf7d792a9b73a61548248b6b5116a85c3a7d0; não garante que este ainda seja o HEAD nem prova deploy.
- Inventário relatado VPS: botwhatsapp-stage-bot-1, projeto botwhatsapp-stage, volume botwhatsapp-stage_botwhatsapp_data, /var/lib/docker/volumes/botwhatsapp-stage_botwhatsapp_data/_data. **Verificar tudo novamente no runtime efetivo.**
- Falhas configuracionais reportadas: BOT_DASHBOARD_TOKEN vazio e EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN ausente.
- O token dashboard é segredo aleatório local. O token Sanctum SÓ o Laravel pode emitir e validar para um User já autorizado com can_send_proposals=true e ability extension:proposal-write. Não inferir ability de tamanho ou formato. Nunca gerar Sanctum aleatório.
- Configuração efetiva do Docker pode vir do Compose environment e sobrescrever env_file, até com string vazia. Editar .env.production não altera container já iniciado. Confirmar sem imprimir valores.

## Fase 1 — Avell / Antigravity (sem rede produtiva)
Na PR #10: git fetch, verificar HEAD/diff, atualizar com fast-forward, sem reset. Rodar gate local já existente scripts/validate-ctwa-handoff-local.ps1 no HEAD final. Para novos scripts, bash scripts/validate-ctwa-p0-credentials.sh executa preflight falso, unittest privado, php -l quando disponível. Não usar -Preview se já homologado o mesmo HTML.

## Fase 2 — segredos privados (sem chat, sem Github)
Provisionamento de painel NO HOST REAL DO BOT, após verificar qual arquivo .env é efetivo:

  python3 scripts/configure-ctwa-secrets.py /srv/platform/stacks/botwhatsapp/.env.production

O comando padrão é um plano sem escrita. Para instalar a senha aleatória local com backup privado e troca atômica:

  python3 scripts/configure-ctwa-secrets.py /srv/platform/stacks/botwhatsapp/.env.production --apply

Caso Sanctum não exista, sai NO_GO mas pode ter gravado a nova senha do painel. NÃO rotacionar painel existente sem autorização específica.

Em CRM Laravel REAL autorizado: verificar usuário existente com can_send_proposals=true e identificar USER_ID inequivocamente. Copiar os scripts PHP publicados na PR #10 para diretório 0700 fora da árvore do CRM e executar sem expor saídas secretas:

  php /private/issue-ctwa-sanctum-on-crm.php /var/www/evydencia/current USER_ID /private/ctwa-sanctum.token
  php /private/verify-ctwa-sanctum-on-crm.php /var/www/evydencia/current /private/ctwa-sanctum.token

Emissor altera APENAS a tabela Sanctum de tokens do CRM, requer usuário com permissão comercial e emite ability mínima com expiração em 30 dias; não muda código, fluxos ou pedidos. Verificador somente lê DB e confirma hash, ability, usuário e validade sem emitir proposta. Exige acesso e identidade de usuário autorizados; se faltar, BLOCKED, não usar token antigo 416.

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
