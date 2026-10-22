# Fluxo Meta Ads e saudações — contrato do bot em 22/10/2026

## Evidência do anúncio inspecionado
O relatório read-only do Antigravity informa que a instância GO `anderson`, conectada ao **48 3220-0440**, recebeu às 07h26 BRT um contato pelo Reel/CTWA Instagram. O texto completo é **"Quero conhecer os pacotes do Primeiro Natal em Tijucas"**, no campo `data.Message.extendedTextMessage.text`. O campo `Chat` traz JID telefônico `@s.whatsapp.net`; `SenderAlt` é `@lid`. `externalAdReply` e `referral` indicam a origem do anúncio, mas são metadados contextuais e NÃO são evidência criptográfica nem gatilho sozinho.

**Esta auditoria leu o relatório enviado pelo usuário e o GitHub, não fez nova consulta ao banco da VPS.** IDs de mensagens, nomes/telefones de clientes, thumbnail, tokens do webhook e CTWA click IDs foram intencionalmente excluídos deste repositório. IMPORTANTE: o anexo original do relatório NÃO está totalmente sanitizado — contém campo `instanceToken` e outros campos sensíveis. Revisar/rotacionar credencial se estiver ativa e nunca versionar o JSON bruto.

## Regras efetivas
1. Instância Ads (48 3220-0440): a frase exata normalizada (lowercase, espaços repetidos e pontuação de borda) deve constar em `config/ads-triggers.txt` OU `BOT_ADS_TRIGGERS`. **O arquivo tem precedência** quando `BOT_ADS_TRIGGERS_FILE` não está vazio. Máximo 50 frases, máximo 240 caracteres por frase, uma linha de texto completo por gatilho. Não usar substring ou termos amplos como "natal". Mudanças aplicadas somente após reinício autorizado DO BOT.
2. Frase reconhecida: `core.AdsProposal`, nome primeiro extraído de PushName opcional e seguro; `workflow.CreateWithOptions` cria passos ordenados:
   - Texto de acolhimento `{{nome}}` (genérico quando não houver nome);
   - Áudio PTT OGG/Opus (se ausente, tarefa suppressed; não inventar arquivo);
   - Imagem oficial e legenda, **somente se `BOT_ADS_IMAGE_FILE` configurado**;
   - Vídeo MP4 e legenda, **somente se `BOT_ADS_VIDEO_FILE` configurado** (limite atual 8 MiB base64 na GO);
   - `POST /api/campaigns/schedule/execute` da campanha **416 SEM NOME**, `contacts` string em DDD+telefone nacional, `instance` ID numérico do CRM. O nome do WhatsApp NÃO é enviado ao CRM.
3. A saída do CRM é assíncrona: ID retornado confirma **agendamento**, não entrega; timeout/resultado incerto não recebe retry automático. Audio/mídia incerta é registrada e não bloqueia CRM, pois a conversão depende do lead válido + autorização, não do upload de material secundário.
4. Mensagem não correspondente recebida no mesmo número de anúncios: com `BOT_ADS_FALLBACK_MENU=true` (padrão), segue `core.Greeting`. Se houver imagem `BOT_GREETING_IMAGE_FILE`, envia UMA imagem com caption do menu; senão envia o mesmo texto da legenda como **mensagem de saudação textual**, nunca silêncio. Cooldown persistente 24 horas.
5. O outro número **48 99642-5287** é role `greeting` APENAS. Nunca dispara automaticamente CRM a partir de uma saudação e nunca usa a APIKEY do número de anúncios. Ainda depende de pareamento/sessão confirmada. O número emissor da proposta CRM é **uma integração à parte**.
6. Opções 1–4: respostas informativas e URLs existentes; opção 5: pausa humana DURÁVEL; `PARAR`: opt-out persistente, inclusive se o contato veio do anúncio. Contatos em pausa/opt-out não podem seguir novos fluxos.
7. Limite de corpo do webhook tanto no relay como no bot é **1 MiB**, pois JSON de CTWA pode carregar miniatura/referral. Nunca persistir esse JSON bruto desnecessariamente no journal do bot; relay somente guarda até encaminhar, em volume privado.

## Etapas e garantias
O outbox grava o lote e cada dependência `PreviousTaskID`. O processador não executa etapa posterior enquanto anterior estiver pending/attempt_started; confirmed/uncertain/suppressed contam como término. Isso garante **ordem de tentativas/registro**, não exatamente-once físico em WhatsApp/CRM após falhas ambíguas. O CRM tem lead-claim diária por número + 416 + tenant (BRT). Um segundo clique do mesmo contato ainda pode gerar outro texto/áudio: avaliar política comercial de cooldown do funil antes de ativar tráfego alto; não afirmar deduplicação integral de TODOS os efeitos.

## Configuração rápida
- Edite `config/ads-triggers.txt` e inclua UMA FRASE COMPLETA por linha, sem as aspas. Comentários começam com `#`.
- Configure `BOT_ADS_TRIGGERS_FILE=/app/config/ads-triggers.txt` no contêiner; `config` deve ser read-only mounted. Se desejar controlar só por env, remova `BOT_ADS_TRIGGERS_FILE` e use `BOT_ADS_TRIGGERS="frase A|frase B"`.
- `BOT_ADS_FALLBACK_MENU=true` mantém saudação para mensagens não reconhecidas.
- A imagem dos pacotes usa `BOT_ADS_IMAGE_FILE` e `BOT_ADS_IMAGE_CAPTION_FILE`; áudio usa `BOT_ADS_AUDIO_FILE`. Vídeo opcional usa `BOT_ADS_VIDEO_FILE` e `BOT_ADS_VIDEO_CAPTION_FILE`.
- Arquivos de mídia NÃO vêm junto deste código; nunca criar imagens artificiais em produção. O worker suprime apenas a mídia faltante.
- Flags de envio continuam OFF em staging; **não** mudar webhook `anderson`, que pertence ao motor de status. O relay precisa de homologação separada.

## Fontes oficiais consultadas
- Evolution GO 0.7.2, `pkg/sendMessage/service/send_service.go` e `pkg/routes/routes.go`: `/send/media` suporta `audio`, `image`, `video`, PTT para áudio, payload base64.
- https://github.com/evolution-foundation/evolution-go/releases/tag/0.7.2
- https://github.com/evolution-foundation/evolution-go/blob/main/docs/wiki/guias-api/api-messages.md (versão v1.0: consultar apenas como referência)
- CRM privado: `routes/api.php`, `SingleAuthentication.php`, `CampaignScheduledResource.php`, `AppServiceProvider.php`, `Natal2026ProposalManifest.php` (somente leitura).
