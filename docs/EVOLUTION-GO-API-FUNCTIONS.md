# Evolution Go 0.7.2 — funções para os dois números

**Não reconfigurar nem substituir a Evolution Go existente na Contabo.** O inventário do AntiGravity aponta imagem anunciada 0.7.2 com patch local de WebP; os UUIDs dos dois números-alvo ainda não existem naquela GO.

## Cliente Go isolado

Arquivo: `internal/evolution/client.go`. Não está conectado ao servidor de preview. Permite:
- `Client.Status(ctx)` -> `GET /instance/status` com header `apikey`, retorno `Status{Connected,LoggedIn,Name}`. Checagem informativa, não garante conexão até o envio.
- `Client.SendText(ctx,phone,text)` -> `POST /send/text`.
- `Client.SendTextWithID(ctx,phone,text,id)` -> igual, com `id` para correlação.
- `Client.SendMedia(ctx,phone,"image",bytes,caption)` -> `POST /send/media`, URL como base64 puro (sem `data:`).
- `Client.SendMediaWithID(ctx,phone,"audio",bytes,"",id)` -> `POST /send/media` com `type=audio`, equivalente a PTT na 0.7.2. Confirmar ffmpeg, MIME e duração em canário.

Todos utilizam **token da instância** no header `apikey`, não o token do CRM. Não repetir uma chamada que pode ter sido processada e gerou timeout. O `id` ajuda a correlacionar, mas não cria idempotência garantida.

## Topologia de rede da Contabo
- A Evolution Go é acessível pelo HOST via `127.0.0.1:14000`; em contêiner separado, `127.0.0.1` apontará para o próprio contêiner, NÃO para a GO.
- Futuro `botwhatsapp` deve utilizar endpoint Docker DNS interno verificado na rede própria/compartilhada aprovada, sem acessar o público `wa-api.etijucas.com.br`.
- Não alterar Caddy/SmartDialer/Asterisk/Status/Portal. Container do bot isolado e sem porta pública.
- Token A só pode enviar pelo número Ads, token B só pela instância de saudação. CRM tem **outro token** e outra instância de envio.

## Handlers e contratos

```http
GET /instance/status
apikey: <TOKEN_EVOLUTION_INSTANCIA>
```
Resposta conceitual:
```json
{"message":"success","data":{"Connected":true,"LoggedIn":true,"Name":"ads"}}
```

```http
POST /send/text
apikey: <TOKEN_EVOLUTION_INSTANCIA>
Content-Type: application/json

{"number":"5548999999999","text":"Recebemos seu contato!","id":"correlacao-unica"}
```

```http
POST /send/media
apikey: <TOKEN_EVOLUTION_INSTANCIA>
Content-Type: application/json

{"number":"5548999999999","type":"image","url":"BASE64_PURO","caption":"*Bem-vinda!*\n1 Natal\n2 Agendamento"}
```

```http
POST /send/media
apikey: <TOKEN_EVOLUTION_INSTANCIA>
Content-Type: application/json

{"number":"5548999999999","type":"audio","url":"BASE64_PURO","caption":""}
```

`data.Info.ID` no JSON de sucesso é o recibo do provedor. Resposta HTTP incompleta ou timeout vira `ErrUnconfirmed`. Evitar logs de conteúdo, base64 e tokens.

## Webhooks

Assinatura da categoria `MESSAGE`; evento recebido `"Message"`. Payload inclui `instanceId`, `data.Info.ID`, `data.Info.Chat`, `data.Info.IsFromMe` e `data.Message.conversation` ou `extendedTextMessage.text`.

`instanceToken` é um segredo transmitido no evento, **não prova autenticação criptográfica**. Receptor deve usar uma rota com segredo independente em rede fechada e validar UUID esperado.

Sem `WEBHOOK_URL` global novo; as duas futuras instâncias devem ter destinos particulares e não sobrescrever webhook de `Anderson`/status. O produtor pode fazer múltiplas tentativas; deduplicar por UUID+tipo+messageID.

## Estado operacional

Não criar sessões sem aprovação explícita. Antes de envio: confirmar conexões `Connected && LoggedIn` e se instância corresponde ao número planejado. O projeto não implementa reconexão automática de contas nem manipula QR code nesta fase.

**Performance:** bytes de áudio e imagem ficam em volume privado; método atual usa base64 até 8 MiB, com overhead ~33%. Se mídia real ultrapassar limite, avaliar multipart. Go API usa HTTP timeouts; worker futuro deve limitar concorrência e ter outbox persistente.

Fontes: https://github.com/evolution-foundation/evolution-go/tree/0.7.2 e `.agents/skills/evolution-go-v072-adapter/resources/verified-contract.md`.
