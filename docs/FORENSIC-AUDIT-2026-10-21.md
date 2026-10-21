# Auditoria forense consolidada — 21/10/2026

## Método e níveis de evidência
- **RELATADO PELO ANTIGRAVITY:** inventário em modo passivo da Contabo, 20:20 BRT. O relatório veio por texto; este projeto NÃO teve acesso direto à VPS nem à imagem local.
- **VERIFICADO NO GITHUB:** contratos e código público Evolution Go 0.7.2, CRM em leitura, projetos de referência e testes compilados via CI.
- **PENDENTE:** comparação do Dockerfile local/patch in-tree com commit exato, arquivos privados .env, UUIDs futuros, estado ativo das campanhas no DB de produção, áudio/imagem autorizados e teste em WhatsApp real.

## Inventário sanitizado relatado
Ubuntu 24.04.4, Kernel 6.8, KVM; 11 GiB RAM total, aproximadamente 9 GiB disponíveis; disco ~193 GB, 78% usado, ~44 GB livres. Docker 29.7.2, Compose v5.5.0, ffmpeg presente no host; Go ausente no host, portanto preferir **Docker multi-stage build**. Não estimar capacidade por RAM livre apenas.

Evolution Go em `evolution-go-evolution-go-1`, imagem local `evoapicloud/evolution-go:0.7.2` e image SHA relatado `80386a78...`; patch local animado/WebP no Dockerfile. Host bind `127.0.0.1:14000 -> :4000`. PostgreSQL 15 dedicado. Caddy partilha a rede `edge`. `smartdialer` (telefonia/Asterisk), `whatsapp-status`, `captive-portal`, `flow-media`, `video-call`, `christmas-games`, `telegram-campaign` e `sms-gateway` partilham o host: **nenhum serviço pode ser reiniciado para este projeto**.

Instância Evolution Go detectada em /instance/all: `Anderson`, UUID `178c3f22-8c7a-43db-91b2-6c7d830503cf`, `connected=false`, com webhook para `whatsapp-status`. Não reutilizar essa instância. Nenhuma instância foi encontrada para `554832200440` ou `5548996425287` **naquele inventário**, não concluir que os números estão desconectados de TODOS os provedores WhatsApp.

## Correções de interpretação do relatório
1. **Falha crítica de autenticação proposta:** o texto diz 'middleware ... validando instanceToken'. ERRADO. `instanceToken` vem no payload e pode ser falsificado. Receptor usa segredo externo independente na URL interna (sem log), mapeia rota à instância e valida UUID. Proibir exposição pública do endpoint.
2. **Cooldown invertido:** o diagrama envia greeting 'Dentro do cooldown 24h'. ERRADO. Enviar apenas se última saudação AUSENTE ou fora da janela; respeitar pausa humana.
3. **Webhook global sem prova:** o relatório afirma 'GLOBAL_WEBHOOK não configurado'. No código oficial a variável é `WEBHOOK_URL`; só se pode afirmar ausência global após comprovar `WEBHOOK_URL` vazio/desabilitado em forma sanitizada.
4. **Loopback e contêiner:** `127.0.0.1:14000` vale no **host**, não dentro de um NOVO contêiner. No futuro, preferir URL DNS interna da GO na rede Docker já existente, após confirmar aliases e autorização de anexação; não expor acesso público.
5. **Imagem e versão não são iguais a source commit:** digest local não prova a revisão Git exata nem a compatibilidade do patch animado. Citar a versão 0.7.2 como a anunciada, comparar handlers e patch antes do deploy.
6. **Campanhas 415/416:** constantes foram confirmadas em `Natal2026ProposalManifest::VARIANTS`, mas não a situação real no DB/ativo; confirmar antes de emitir POST.
7. **CRM instance=null:** favoritar automaticamente outra instância pode mudar o número remetente. Exigir `message_instance_id` explícito e validado antes do envio real.
8. **Áudio PTT:** código constrói `AudioMessage PTT=true`, mas êxito de transcodificação e execução de ffmpeg requer canário; não presumir funcionamento pelo simples fato de o binário existir.
9. **WhatsApp não oficial:** Evolution Go é via cliente WhatsApp Web/whatsmeow; integração depende de sessões estáveis e pode apresentar desconexões/restrições. Monitorar status antes de envios, sem alterar serviços.

## Decisões arquiteturais
- Go puro (`go 1.23`) com `net/http`, sem redis adicional, sem n8n, sem OpenClaw/Chatwoot. A execução futura fica em container separado, leve, sem Go no host.
- Fase presente: parser+roteador+adapter HTTP não conectados a saídas + journal de eventos preview-only, `BOT_ENABLED=false`.
- O journal por evento é um artefato de **prévia**, NÃO um motor de outbox pronto para produção. Antes de ativar disparos: implementar transições `pending -> intent_persisted -> attempt -> confirmed/uncertain`, controle por telefone/campanha, pausa humana, cooldown durável, checkpoint/recovery e retenção com observabilidade. Não ligar os clientes de saída antes disso.
- Para escala de duas instâncias e tráfego local moderado, evitar infraestrutura extra; medir latência/memória/durabilidade com fixtures. Não prometer métricas sem benchmark.
- Contato explicitamente opt-out, sem telefone confiável, trigger incompleto ou campanha inativa: fail closed.
- Créditos e referências em `docs/THIRD-PARTY-RESEARCH.md`.

## P0 bloqueios para produção
- Duas sessões/UUIDs NÃO EXISTEM no inventário; precisam de provisionamento/pairing aprovado e confirmação de política de números.
- Texto exato do anúncio, imagem, áudio e modelo de menu definitivo não foram enviados.
- Não há instância explícita de saída CRM validada; IDs 415/416 do código não são prova de campanha ativa.
- Engine de saída durável ainda não implementada; código atual NÃO envia mensagens.
- Rotas privadas, isolamento da rede, verificação de secret e execução pós-pareamento carecem de canário.
