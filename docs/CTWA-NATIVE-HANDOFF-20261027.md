# CTWA 5287 → Proposta Natal V4: correção cirúrgica 27/10/2026

## Evidência e estado

Main do bot auditada: `6af807225704c710b68a5a185c0725b26369cb3f`.
Main do CRM auditada: `60ba4e2de4fecee290f9b51ac85fac679e220315`.
No CRM, `NatalProposalEntryPoint::assertGenericAllowed` proíbe explicitamente 415/416 no POST genérico; 422 é rejeição definitiva, não timeout. Histórico relatado pelo operador/Antigravity: 13 ocorrências CTWA com PDF/áudio aceitos pela Evolution e CRM recusado com `natal_proposal_native_entry_required`. Sem acesso VPS a partir deste patch: não afirmar entrega final ou que o novo binário esteja em produção.

## Novo contrato, sem alterar o CRM

- Endpoint nativo já existente: `POST /api/v1/extension/proposals/schedule`.
- `Authorization: Bearer <Sanctum token do operador de serviço>` com ability **extension:proposal-write** e usuário ativo autorizado por `ProposalFlowPolicy::send`; diferente do token raw da API antiga.
- Body: `{"request_id":"UUIDv5 estável da task","phone":"DDDNÚMERO","name":null,"resend_mode":"normal"}`.
- Header `X-Idempotency-Key` é igual ao `request_id`.
- Receber apenas 200/201 `{"data":{"status":"scheduled","dispatch_uuid":"UUID"}}` como agendamento nativo confirmado. NÃO confundir com entrega Z-API/WhatsApp.
- 400/401/403/404/422: rejeição definitiva, não repetir automaticamente. Timeout/5xx/429/409/recibo inválido: **uncertain**, conciliar antes de qualquer tentativa.
- A variante 415/416 é resolvida somente no CRM segundo sua regra comercial (nome do WhatsApp é inseguro). Não liberar o caminho genérico.
- Flag secret `EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN` configurada APENAS em ambiente privado de produção. Contas, tokens e telemetria não devem ser commitados.
- O fluxo 5287 já usa opt-out, pausa humana e cooldown 24h; não zerar journal, não executar envio duplicado, não trocar webhook, portas, Evolution ou outros serviços.

## Painel já existente

O bot já expõe `/painel` e `/api/dashboard/contacts`. Nesta PR:
- Bloquear acesso sem `BOT_DASHBOARD_TOKEN`; exigir Basic Auth HTTPS no navegador, ou header `X-Dashboard-Token` em uso interno. Não usar `?token=` em links.
- Exibir `Rejeitado / não agendado`, `Incerto / revisar`, `Pendente` ou `Aceito no CRM (proposal-dispatch-UUID)`, sem confundir HTTP 2xx com mensagem recebida.
- O painel é leitor do journal local, **não consulta automaticamente o CRM**. Não rotular histórico não verificado como entregue. Não publicar diretamente na internet.

## Execução curta do Antigravity, com restrições

1. Ler esta PR, PR #9 (correção de identidade 5287 ainda draft), comparar main/head/produção. Resolver o default obsoleto `0440` antes do deploy; compor os patches de PR #9 sem reverter as garantias.
2. Validar `go test ./internal/crm ./internal/outbox ./internal/dashboard ./cmd/botwhatsapp`, `go vet ./...`, `gofmt` somente nos arquivos Go alterados. Fazer smoke local com `httptest`. Resultados não foram executados pelo autor GitHub remoto.
3. Revisar o token do painel e acesso HTTPS privado (sem query secret), e prover Sanctum **com escopo mínimo**. Instância de saída é selecionada pelo agendador nativo do CRM, não pelo `message_instances.id` hardcoded do legado.
4. Pré-produção: verificar configuração real do 5287, decisão pelo trigger literal, spool e 13 rejeições sem reinicializar/limpar journal. Consultar primeiro o CRM; separar dispatch já existente, rejeição definitiva e incerto. Não fazer replay automático.
5. Canário **um novo lead autorizado**: receber texto no 5287, observar etapas Evolution, confirmar recibo nativo `dispatch_uuid` e verificar no CRM que a proposta foi de fato agendada. Confirmar processamento posterior pela instância CRM sem inferir entrega a partir de aceite HTTP.
6. Rollout de somente bot após aprovação operacional, com health e rollback preservando volume, UUIDv5 e journal. Nunca reiniciar Evolution, Status Engine ou CRM para resolver esta PR.

## Histórico a reconciliar

Os 13 contatos relatados pelo Antigravity em 27/10 tiveram HTTP 422, mas **não foram reencaminhados por este patch**. Conferir elegibilidade, compra, opt-out e eventual dispatch antes de programar envios manuais com aprovação. A lista de 19 contatos dos prints não deve ser tratada inteira como falha CRM: são fontes diferentes, com sobreposições.

## Critérios de aceite

- NOVO lead: recibo `dispatch_uuid` persistido, proposta visível no CRM, sem duplicação, phone normalizado conforme `BrazilMobilePhone`.
- Rejeição 422 visível e sem novo POST.
- Timeout preserva `uncertain`; painel não informa sucesso falso.
- Painel restrito, sem URL token nem HTML não escapado de perfis.
- Confirmação de que PR #9 corrige 0440/5287 antes da implantação.
