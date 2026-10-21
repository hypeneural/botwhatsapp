# Contrato selecionado da Evolution Go 0.7.2
Verificado por leitura dos handlers e structs oficiais e confrontado com o inventário do AntiGravity. A imagem customizada instalada NÃO foi obtida por este agente.

- `POST /send/text`: header `apikey: <TOKEN_DA_INSTANCIA>`; body `{"number":"55DDD...","text":"..."}`.
- `POST /send/media`: mesmo header; `{"number":"55DDD...","type":"image","url":"base64-ou-url","caption":"..."}` ou `type:"audio"` para voz PTT. Validar MIME, ffmpeg e arquivo real em canário; base64 puro e multipart são suportados no handler consultado.
- Envio bem-sucedido: `data.Info.ID` não vazio. Resultado HTTP incerto/não parseável não autoriza novo envio automático.
- `POST /instance/connect`: `{"webhookUrl":"...","subscribe":["MESSAGE"]}`; operação MUTATIVA, somente na fase de pareamento aprovada.
- Webhook: `event:"Message"`; `data.Info.ID`, `data.Info.Chat`, `data.Info.IsFromMe`, `data.Message.conversation` e `data.Message.extendedTextMessage.text`. Pode conter `instanceToken` secreto no envelope, que NÃO é autenticação de origem. Não armazenar/logar payload cru.
- O produtor pode postar tanto no `WEBHOOK_URL` global quanto no webhook por instância; até 5 tentativas quando há falha. Atenção: variável canônica `WEBHOOK_URL`, **não `GLOBAL_WEBHOOK`**.
- UUID de instância e mapeamento de números não são dedutíveis do telefone informado pelo usuário.
- `@lid`, `@g.us`, `status@broadcast`: não converter em telefone de CRM.

Fontes:
https://github.com/evolution-foundation/evolution-go/blob/0.7.2/pkg/routes/routes.go
https://github.com/evolution-foundation/evolution-go/blob/0.7.2/pkg/sendMessage/service/send_service.go
https://github.com/evolution-foundation/evolution-go/blob/0.7.2/pkg/events/webhook/webhook_producer.go
https://github.com/6justgotme/evolution-go-skill/tree/90d23690f7fa79bc252ee006675b4c7e50e0a176/reference
https://github.com/estevao-tarifa/evo-chatwoot-bridge-go/tree/7ba5bb1a69734c8f486e7515bd1b3b9b3f82571c/backend/internal

Atribuição: os dois projetos especializados de terceiros têm licença MIT. Este resumo é redação própria; implementações atuais em `internal/` foram escritas para este projeto e não contêm cópia literal de seus arquivos.
