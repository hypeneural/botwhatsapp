# Arquitetura multi-instância de baixo custo para Evolution Go

## Identidades sem confusão
Cada instância Evolution Go tem **UUID estável**, `apikey`, número comercial `E.164`, webhook privado, `role` e estado de conexão. Nenhum desses valores é o ID numérico `message_instances.id` no Evidência CRM.

```text
Evolution Go: ads / 48 3220-0440 ──┐
                                   ├─> Webhook seguro -> registry(UUID) -> journal -> worker
Evolution Go: greeting / 48 99642-5287 ─┘                     |
                                                             +-> Go HTTP (token da instância)
                                                             +-> CRM HTTPS (token interno, instance=1 validado)
                                                             +-> shared LeadClaim(tenant, phone, 416, BRT date)
```

`internal/instances/registry.go` é a biblioteca inicial para N instâncias e **não está conectada ao worker do canário**: isso deve ser feito depois do push/review do código do AntiGravity. Evitar reimplementar worker de produção às cegas.

### Registros e secrets
- Cada instância possui `slug`, `instanceUUID`, `businessPhoneE164`, `role` ads/greeting/observe, `enabled`, nomes de variáveis de ambiente `tokenEnv` e `webhookSecretEnv`, gatilhos exclusivos para ads e permissão de CRM.
- Nunca versionar tokens brutos. Ao iniciar, rejeitar UUID/telefone duplicados, instância habilitada sem token/secret, ads sem gatilhos e greeting com CRM permitido.
- `Registry.ResolveUUID` (map O(1)) valida instância. `Registry.VerifyRoute` exige caminho de webhook **exato** com segredo de alta entropia, sem segmentos adicionais.
- `Registry.Decide` reaproveita `core.Select` sem criar outro motor de parsing.
- `instances.EventKey(UUID,event,id)` é por transporte, separada de `instances.LeadClaimKey(tenant,nationalPhone,campaign,when,timezone)`. A claim não contém UUID, bloqueando dupla solicitação CRM no mesmo dia quando chega por dois números do mesmo estúdio.
- Para N instâncias, o worker terá um cliente Evolution por UUID/token com timeout e fila, mas um cliente CRM de saída de configuração explícita.
- Aplicar limite de concorrência inicial um worker por instância e uma claim global por lead/campanha, mediante medição local (não assumir que a VPS comporta tráfego ilimitado).

### Fan-out do webhook da instância ads já ocupada
**Não alternar o webhook `anderson` para o bot em produção.** O motor de status usa o endpoint atual. É necessário planejar UMA das opções abaixo:
1. Melhor quando possível: o consumidor `whatsapp-status` já recebe o evento, e um encaminhamento assíncrono autorizado para o novo bot preservaria o destino original; requer auditar suporte e contrato do status-engine, NÃO modificar esse serviço sem autorização.
2. Relay dedicado, com ingresso durável + fan-out para status-engine e bot, garantindo ACK e redelivery quando um destino falha. Só reconfigurar a instância uma vez após homologação do relay, com rollback testado e autorização. Não usar forwarder best-effort sem persistência.

Evitar ativar `WEBHOOK_URL` global na Evolution GO por conveniência: pode afetar TODAS as instâncias e requer mudanças na stack compartilhada. Evitar múltiplas mudanças `/instance/connect`.

### Identidade e entrega
- Mensagens de entrada podem conter `@lid`; jamais derivar número nacional de dígitos da LID. Resolver por JID telefone comprovado em `Info.Sender`/SenderAlt, ou bloquear campanha.
- `IsFromMe`, eventos `Receipt`, `SendMessage`, status, grupos e broadcast não iniciam fluxo comercial.
- Envio texto/áudio usa **apikey da instância ads**. Saudação imagem usa **apikey da instância greeting**. Campanha 416 usa **Authorization raw do CRM** e `instance` da tabela do CRM.
- Recibo GO em `data.Info.ID` não prova `delivered`; guardar ID e aguardar, quando houver, evento de recibo de status.

### Segurança de rede
- Melhor produção: novo contêiner do bot na rede privada Docker autorizada, sem porta pública. Não referenciar `127.0.0.1:14000` de dentro do contêiner; usar DNS/alias GO já existente na rede interna.
- Canário em Windows: escutar APENAS na interface Tailscale ou filtrar firewall. `0.0.0.0:8788` escuta demais interfaces mesmo com URL privada.
- Segredo no URL do webhook é bearer capability: usar URL opaca, não logar paths e não deixar reverse-proxy publicar access logs contendo o segmento. `instanceToken` do envelope não é assinatura.

### Envio de CRM
Padrão `campaign=416` sem nome, `contacts:DDD+numero`, `type:"campaign"`, `use_leads_system:false`. Endpoint real `POST /api/campaigns/schedule/execute`. Confirmação é ID do agendamento, não prova de entrega. Não executar outro POST para validar a prova do canário; pesquisar o agendamento existente read-only.

### Testes
`go test ./internal/instances ./internal/ingress ./internal/core ./internal/crm ./internal/evolution`.
Simular duas instâncias diferentes recebendo o mesmo contato; exigir mesma lead claim se marca/campanha/data iguais, apesar dos eventos terem chaves distintas. Testar caminho de webhook com subsegmentos extra. Testar `@lid`, `IsFromMe`, pausas e opt-out depois da reconciliação do worker.
