# Fluxo Meta Ads e saudações — contrato do bot em 23/10/2026

## Evidência do anúncio inspecionado
O relatório read-only do Antigravity informa que a instância GO `anderson`, conectada originalmente ao **48 3220-0440**, recebeu contato pelo Reel/CTWA Instagram. Há dois textos completos comprovados no criativo: **"Quero conhecer os pacotes do Primeiro Natal em Tijucas"** (reel) e **"Quero conhecer os pacotes de Natal em Tijucas"** (imagem/carrossel), no campo `data.Message.extendedTextMessage.text`. O campo `Chat` traz JID telefônico `@s.whatsapp.net`; `SenderAlt` é `@lid`. `externalAdReply` e `referral` indicam a origem do anúncio, mas são metadados contextuais e NÃO são evidência criptográfica nem gatilho sozinho.

**Auditoria e Histórico de Linhas:** Em 22/10/2026 às 20:20 BRT, o WhatsApp desconectou/bloqueou a linha `48 3220-0440`. A auditoria forense confirmou que 100% dos leads recebidos antes da desconexão foram processados, respondidos e agendados no CRM 416 com 0% de falha. A operação de tráfego pago foi então transferida para o número **48 99642-5287** (`5548996425287` / instância `evydencia`), mantendo o menu orgânico como fallback automático.

## Regras efetivas
1. Instância Ads Vigente (48 99642-5287 / `evydencia`): a frase exata normalizada (lowercase, espaços repetidos e pontuação de borda) deve constar em `config/ads-triggers.txt` OU `BOT_ADS_TRIGGERS`. **O arquivo tem precedência** quando `BOT_ADS_TRIGGERS_FILE` não está vazio. Máximo 50 frases, máximo 240 caracteres por frase, uma linha de texto completo por gatilho. Não usar substring ou termos amplos como "natal".
2. Frase reconhecida: `core.AdsProposal`, nome primeiro extraído de PushName opcional e seguro; `workflow.CreateWithOptions` cria passos ordenados com durabilidade no outbox:
   - Reação natalina com emoji na mensagem do lead (padrão `🎄` via `BOT_ADS_REACTION_EMOJI`);
   - Texto de acolhimento `{{nome}}` (genérico quando não houver nome, arquivo `config/ads-welcome-text.txt`);
   - Áudio PTT OGG/Opus (`BOT_ADS_AUDIO_FILE`);
   - Proposta em PDF com envio por URL absoluta segura (`BOT_ADS_DOCUMENT_URL`, servida pelo bot em `/static/Proposta_Natal.pdf`) ou binário base64 direto, acompanhada de legenda comercial (`BOT_ADS_DOCUMENT_CAPTION`);
   - Texto de encerramento (`config/ads-closing-text.txt`);
   - `POST /api/campaigns/schedule/execute` da campanha **416 SEM NOME**, `contacts` string em DDD+telefone nacional, `instance` ID numérico do CRM. O nome do WhatsApp NÃO é enviado ao CRM.
3. A saída do CRM é assíncrona: ID retornado confirma **agendamento**, não entrega; timeout/resultado incerto não recebe retry automático. Audio/mídia incerta é registrada e não bloqueia CRM, pois a conversão depende do lead válido + autorização, não do upload de material secundário.
4. Mensagem não correspondente recebida no mesmo número de anúncios: com `BOT_ADS_FALLBACK_MENU=true` (padrão), segue `core.Greeting`. Se houver imagem `BOT_GREETING_IMAGE_FILE`, envia UMA imagem com caption do menu; senão envia o mesmo texto da legenda como **mensagem de saudação textual**, nunca silêncio. Cooldown persistente 24 horas.
5. Painel Web de Visualização em Tempo Real: o bot disponibiliza a interface `/painel` (e APIs JSON `/api/dashboard/stats` e `/api/dashboard/contacts`), permitindo acompanhar em tempo real todos os contatos atendidos, distinguir tráfego pago de orgânico e auditar o status dos agendamentos no CRM 416.
6. Opções 1–4 do Menu: respostas informativas e URLs existentes; opção 5: pausa humana DURÁVEL; `PARAR`: opt-out persistente, inclusive se o contato veio do anúncio. Contatos em pausa/opt-out não podem seguir novos fluxos.
7. Limite de corpo do webhook: configurável via `BOT_MAX_BODY_BYTES` (padrão 32 MiB), permitindo acomodar com folga payloads de CTWA e anexos de mídia sem saturar o spool. Nunca persistir esse JSON bruto desnecessariamente no journal do bot.

## Etapas e garantias
O outbox grava o lote e cada dependência `PreviousTaskID`. O processador não executa etapa posterior enquanto anterior estiver pending/attempt_started; confirmed/uncertain/suppressed contam como término. Isso garante **ordem de tentativas/registro**, não exatamente-once físico em WhatsApp/CRM após falhas ambíguas. O CRM tem lead-claim diária por número + 416 + tenant (BRT). O worker marca duravelmente `LastAdsFlowAt` ao iniciar a primeira tentativa de texto; uma segunda sequência do mesmo contato dentro de `BOT_ADS_FLOW_COOLDOWN_HOURS` (1–168h, padrão 24h) tem o texto suprimido e as demais etapas em cascata suprimidas, inclusive CRM. Isso é uma janela **por contato no journal local** (não locks entre réplicas) e não garante exatamente-once físico. O lead claim CRM 416 continua como segunda proteção por dia BRT.

## Limites de evidência
- Eventos exclusivamente com `@lid` e sem JID telefônico validado devem ser ignorados e contabilizados sem tentativa de enviar ao LID ou converter para CRM. NÃO supor retransmissão garantida com telefone.
- Não salvar PushName, número, identificadores Meta/CTWA nem tokens reais nos testes ou no Git.

## Configuração rápida
- Edite `config/ads-triggers.txt` e inclua UMA FRASE COMPLETA por linha, sem as aspas. Comentários começam com `#`.
- Configure `BOT_ADS_TRIGGERS_FILE=/app/config/ads-triggers.txt` no contêiner; `config` deve ser read-only mounted. Se desejar controlar só por env, remova `BOT_ADS_TRIGGERS_FILE` e use `BOT_ADS_TRIGGERS="frase A|frase B"`.
- `BOT_ADS_FALLBACK_MENU=true` mantém saudação para mensagens não reconhecidas no mesmo número.
- A proposta em PDF usa `BOT_ADS_DOCUMENT_URL` (recomendado para alta compatibilidade de entrega), `BOT_ADS_DOCUMENT_FILENAME` e `BOT_ADS_DOCUMENT_CAPTION`. Áudio usa `BOT_ADS_AUDIO_FILE`.
- Arquivos de mídia comerciais reais permanecem restritos no volume local `assets/` e são protegidos pelo `.gitignore`.

## Fontes oficiais consultadas
- Evolution GO 0.7.2, `pkg/sendMessage/service/send_service.go` e `pkg/routes/routes.go`: `/send/media` suporta `audio`, `image`, `video`, `document`, PTT para áudio, payload base64 ou URL.
- https://github.com/evolution-foundation/evolution-go/releases/tag/0.7.2
- CRM privado: `routes/api.php`, `SingleAuthentication.php`, `CampaignScheduledResource.php`, `AppServiceProvider.php`, `Natal2026ProposalManifest.php` (somente leitura).
