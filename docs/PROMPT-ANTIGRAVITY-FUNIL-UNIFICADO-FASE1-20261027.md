# Antigravity — executar somente homologação da Fase 1: Evolution Go × Bot × Meta
**Referência:** https://github.com/hypeneural/botwhatsapp/pull/12 (stacked sobre PR #10). Leia primeiro `docs/AUDITORIA-FORENSE-EVOLUTION-CRM-META-20261027.md` e `AGENTS.md`. **SEM DEPLOY, SEM DISPARO, SEM SECRET EM LOG.**

## 1. Reconciliação do checkout
No repositório local existente: `git status --short`, `git branch --show-current`, `git fetch origin`. Preservar trabalho não commitado e PR #10 (não force push/reset/clean/rebase destrutivo). A PR #12 contém parser `internal/ingress/ad_referral.go`, ingresso `audit.go`, API `events.go` e UI `ui.go` já escritos e com testes sintéticos. Não reimplementar ou criar outra branch. Se #10 avançou, comparar diff e integrar com merge/rebase supervisionados após inspeção de conflitos.

## 2. Gate local único
Executar uma vez `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./internal/ingress ./internal/dashboard`, a partir do checkout real, usando toolchain compatível com a PR #10; `git diff --check`. Se Go estiver indisponível, marcar `NO_GO_TOOLCHAIN` sem baixar pacote suspeito.
Validar fixture oficial Evolution Go **tag 0.7.2**, campos `data.referral.sourceType=ad` e `sourceID` em camel exato como `protojson`, com fallback nested `externalAdReply`, e negativos: sourceType post, sem sourceID, @lid-only, duplicated message, fromMe, mensagem sem trigger, texto copiado e JSON malformado.
Confirme que nenhum `ctwaClid`, source URL, miniatura, body bruto nem segredo é persistido. A UI deve mostrar apenas 'Referência de anúncio' (ainda não verificada) ou 'Indício de anúncio', inclusive mobile 390 px; origem sem evidência não pode ser 'orgânica comprovada'.

## 3. Gate de inventário Contabo (somente leitura, nenhuma alteração)
Identificar nome do container Evolution Go, imagem/tag/digest, status dos containers bot/relay/status-engine e timestamps de disponibilidade, apenas campos não secretos. Confirmar presença, NÃO valor, de `DATABASE_SAVE_MESSAGES` e se o journal ou banco mantém metadados de mensagens. Não imprimir `docker inspect` integral, `.env`, `docker logs` brutos, payload/telefone/token nem ativar debug.
Documentar se a instância 5287 tem `Message` encaminhado ao bot e/ou status-engine e se existe journal independente cobrindo horas nas quais o bot esteve OFF. Histórico 0440 é separado. Sem cobertura, `unknown` e não zero.
Se existir um evento de teste antigo **autorizado e identificável por ID**, verificar somente presença de `referral`, nomes dos campos, tipo, sourceID mascarado (não o valor total), event ID deduplicado e horário; não reproduzir mensagens ou exportar conversa.

## 4. Reconciliação read-only com Meta/CRM
Contar por ID na janela para Evolution e bot, sem telefone em logs. Consultar Meta Ads conta `act_1627262698736055` estritamente READ_ONLY, `level=ad`, e confirmar que sourceID tem correspondência com Ad ID real; ausência de correspondência = candidato não atribuído, não 'falha'.
Obter apenas contagens agregadas: gasto, `clicks`, `inline_link_clicks`, ações de conversa com `action_type` e janela explícitos; não equiparar cliques com leads únicos.
Para CRM, não fazer GET 360 por centenas de contatos nem usar `CampaignScheduledService::search` sem paginação. Registre necessidade/contrato da issue #442 Codex. Diferença entre `NatalCommercialConversionPolicy` e `NatalValidOrdersQuery` deve impedir usar "cliente convertido" como "pedido válido" no ROAS.

## 5. Relatório final
Uma página somente com SHA/branch, versões/digest, PASS/FAIL/SKIP do gate, cobertura das fontes, contagens SEM PII, localização dos artefatos e **GO/NO_GO**. Não enviar proposta de teste, fazer replay dos 13 casos 422, alterar campanha, orçamento, DNS, tokens, relé, webhook, volume, compose, cron ou imagem em produção. Se faltar acesso ou evidência, anotar `UNKNOWN`, nunca inventar. Só depois agendar decisão supervisonada de integração com Codex/Meta.
