# CTWA — telefone, nomes, domínio HTTPS e painel — auditoria 27/10/2026

## Capturas e sintomas
- Captura do gestor mostra histórico do bot com registros sem telefone e nomes incompletos. Uma tela renderizada NÃO comprova que a informação de identidade foi resolvida corretamente nem que as APIs externas já funcionam.
- Print de DNS mostra um formulário **Adicionar A** com host bot.etijucas.com.br e IPv4 informado 169.58.210.14. O print NÃO comprova que o registro foi salvo, que o IP pertence à instância produtiva, que existe certificado TLS nem que o Caddy está ouvindo o host.

## Causa confirmada no código
- internal/core/core.go/Select: para @lid sem Info.Sender ou Info.SenderAlt confiável no formato 55DDD+numero@s.whatsapp.net, a decisão é Ignore/unresolved_phone_or_non_private_chat, PhoneE164 vazio. Isso é segurança correta: LID NÃO é telefone.
- Info.PushName é o nome declarado pelo WhatsApp, não prova nome CRM; em webhook legado/sem PushName não existe nome confirmado.
- internal/dashboard/events.go mostrava "Telefone não resolvido" e "Contato WhatsApp" mesmo quando podia juntar eventos do **mesmo ReplyJID** com telefone explicitamente resolvido em outro evento, ou nome validado por telefone.
- internal/dashboard/service.go buscava nome apenas nas tarefas com ContactName, sem consulta CRM nem fallback consistente dos journals.

## Correções na PR #10 (NÃO presumir em produção)
1. Core guarda PushName saneado mesmo em @lid não resolvido, mas NUNCA infere telefone com dígitos do LID.
2. Eventos mostram "Número oculto pelo WhatsApp (LID)" com estado visível "Telefone pendente de identificação"; tentam associação somente por identidade @lid EXATA já vinculada a E.164 em outro evento carregado; conflitos ficam sem resolução, não reenviam ao CRM.
3. Serviço de contatos prioriza nome cadastrado no CRM, quando comprovado pela consulta GET read-only com correspondência UNICA; senão, usa PushName associado a número verificável, senão "Nome não informado". Sem associar pelo nome, sem consulta por sufixo de telefone.
4. scripts/sync-ctwa-crm-names-readonly.py preenche cache privado 0600 de nomes a partir de GET /api/v1/extension/customer-context?phone=DDD+numero e somente status lookup=matched. Ability necessária extension:customer-read. A credencial Sanctum compartilhada da extensão contém essa ability no caso consultado, mas **um token só proposal-write retornaria 403**: nesse caso bloquear enriquecimento, NÃO modificar CRM nem privilegiar o token.
5. A cache é lida apenas pelo painel de contatos e expira após 48 horas; CLI tem limite máximo de 100 consultas por rodada, padrão 40, sem imprimir nomes/telefones. Execute no host real do bot, nunca na recepção e nunca contra o CRM de escrita. Uma rodada só lê os nomes disponíveis para até 40 contatos ainda não verificados; reexecutar com intervalo suficiente para respeitar throttle do CRM.
6. Os arquivos da PR foram preparados SEM restart do worker/Compose nem edição do CRM; preserva spool, claims e opt-outs.

## Usar scripts existentes, NO-GO até validar
Dentro de checkout atualizado para PR #10, no host do bot:
```bash
bash scripts/validate-ctwa-p0-credentials.sh
python3 scripts/audit-ctwa-panel-readonly.py --env /srv/platform/stacks/botwhatsapp/.env.production
python3 scripts/sync-ctwa-crm-names-readonly.py --env /srv/platform/stacks/botwhatsapp/.env.production --spool /var/lib/docker/volumes/botwhatsapp-stage_botwhatsapp_data/_data
# Somente após confirmar scope customer-read, caminho REAL e modo privado:
python3 scripts/sync-ctwa-crm-names-readonly.py --env /srv/platform/stacks/botwhatsapp/.env.production --spool /var/lib/docker/volumes/botwhatsapp-stage_botwhatsapp_data/_data --apply
```
Atenção: cache gravada no host deve ser lida pelo mesmo diretório montado no container, e UID/permissões reais devem ser conferidos. Na ausência de token com customer-read, registrar SKIP/NO-GO de enriquecimento, não quebrar fluxo comercial.

### Nome e telefone não devem ser inventados
O provedor Evolution Go suporta Info.Sender/SenderAlt como JID alternativo e o changelog descreve troca LID/telefone; consultar versao EFETIVA instalada e amostras redigidas. Se faltar telefone, nunca usar dígitos do LID como WhatsApp/CRMPhone. Não editar journal histórico para fabricar eventos, não buscar contatos por coincidência de nome nem fazer POST comercial para descobrir identidade.

## DNS e HTTPS
- DNS desejado: bot.etijucas.com.br A 169.58.210.14 CONFORME print; confirmar que o registro FOI SALVO e que este é o IP público REAL da VPS antes de executar mudanças.
- Caddy automático HTTPS exige DNS correto e portas 80/443 conforme documentação oficial: https://caddyserver.com/docs/quick-starts/https
- Template conservador: deploy/caddy/bot.etijucas.com.br.caddy.example. É EXEMPLO A SER INTEGRADO ao Caddyfile **existente**, não substitui configurações de outros sites/serviços. Upstream 127.0.0.1:8788 só é correto se Caddy estiver no mesmo namespace do host e a porta estiver no loopback; se Caddy for container, confirmar rede e usar upstream real.
- Publicar SOMENTE /dashboard, /painel e /api/dashboard/* sob HTTPS; ocultar publicamente /static/* e /webhook/* (Evolution deve continuar na rota interna autorizada). Não expor raw spool ou env. Bot já exige token Basic Auth no painel.
- Validação DNS / borda read-only: bash scripts/audit-ctwa-dns-readonly.sh 169.58.210.14 APENAS SE comprovado IP WAN; /dashboard, /painel e endpoints JSON exigem 401 sem senha, /static/* deve dar 404.
- Validação completa por endereço HTTPS depois de Caddy: python3 scripts/audit-ctwa-panel-readonly.py --env /private/.env.production --base-url https://bot.etijucas.com.br --panel-only. O script inclui credencial Basic no request, por isso só usar APÓS validar domínio e certificado com segurança. Não imprimir secrets no chat.
- Testar Browser real desktop/mobile e menu de contatos, JSON, timezone e filtros. Renderizar nomes e telefones existentes; conferir exemplos comparando com CRM read-only, sem mostrar PII no log.

## GO/NO-GO
- GO parcial: codigo novo revisado, tests/CI, Caddy validado, nome CRM verificado, LID indecidivel rotulado claramente.
- NO-GO para declarar "100%": faltam capturas de navegador usando HTTPS externo, API JSON externa autorizada e canario WhatsApp/CRM.
- Zero alterações no CRM, Evolution, worker adicional ou outras stacks; merge/deploy só depois de testes e checagem da produção real.
