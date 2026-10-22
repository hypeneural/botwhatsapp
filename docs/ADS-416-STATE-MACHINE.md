# Ordem recomendada — Meta Ads 48 3220-0440 → CRM 416

## Antes do envio
- A instância Ads deve estar pareada e conectada, mapeada por UUID confirmado.
- Texto exato de anúncio autorizado e recebido por webhook autenticado.
- Lead com telefone confiável (JID telefone/associação provada, nunca `@lid` bruto).
- Sem opt-out, pausa humana ou duplicidade de lead/campanha dentro da janela definida.
- ID numérico `message_instances.id` explícito para o número de saída do CRM.
- Campanha 416 ativa em produção e adequada para a proposta de Natal 2026.
- Flag de efeito externo ON somente depois de autorização de canário.

## Sequência
1. **Ingresso**: receptor responde HTTP 202 apenas após gravar o evento; `(instanceUUID,event,messageID)` para dedup do transporte.
2. **Elegibilidade**: texto exatamente correspondente aos gatilhos do anúncio, com normalização conservadora; opt-out e pausa humana têm precedência.
3. **Claim comercial**: transação durável `(phoneNational,campaign=416,window)` previne reagendamento por novas mensagens do mesmo lead.
4. **Efeito 1 (resposta)**: persistir intenção+ID, marcar `attempt_started`, enviar texto pelo número Ads. Confirmar `data.Info.ID` para `confirmed`; timeout -> `uncertain`.
5. **Efeito 2 (áudio)**: enviar áudio pré-gravado pelo número Ads se arquivo aprovado. Persistir confirmação/estado. Falha do áudio não deve bloquear solicitação CRM, mas manter registro independente.
6. **Efeito 3 (CRM)**: persistir intenção e tentativa; `POST /api/campaigns/schedule/execute` com `campaign:416`, `contacts:DDD+numero`, `instance:<id CRM explicitamente validado>`, `type:"campaign"`, `use_leads_system:false`.
7. **Agendamento confirmado**: gravar ID retornado + campanha + instância + status do CRM. HTTP 2xx sem recibo inequívoco -> `uncertain`, não repetir POST. O CRM cuida do envio pelo número distinto e de seus lembretes; o bot não deve enviar PDF por fora.
8. **Respostas posteriores**: não reaplicar campanha pelo menu; delegar atendimento humano ou menu onde apropriado.
9. **Reinício**: efeitos `attempt_started` não podem ser reexecutados automaticamente se status for desconhecido. Reconciliar manualmente usando GET do CRM, sem interpretar busca vazia como prova de não envio.

## Modelo de estados (futuro motor; ainda não executado)
```
received -> qualified -> lead_claimed
  -> text_pending -> text_attempt_started -> text_confirmed | text_uncertain
  -> audio_pending -> audio_attempt_started -> audio_confirmed | audio_uncertain
  -> crm_pending -> crm_attempt_started -> crm_confirmed | crm_uncertain
```
Os três efeitos são independentes; áudio pode falhar sem impedir CRM. No caso de resposta de texto desconhecida, priorizar ausência de duplicação em vez de repetir cegamente.

## Saudação 48 99642-5287
`Message` novo + não pausado + cooldown expirado => uma imagem com caption/menu. Opções 1–5 recebem texto específico, com opção 5 interrompendo o bot para humano. Não disparar campanha 416 por padrão na instância B. Cooldown/estado humano devem ser persistidos no executor.

## Avisos
O projeto ainda está em preview-only; esta é a especificação para o worker do Antigravity, não afirmação de produção funcionando. Não enviar mensagens ou campanhas até aprovar o canário.
