# Antigravity Flash Lite — preflight final dos dois anúncios Meta e preparação de produção

**Repositório único:** `hypeneural/botwhatsapp`; branch oficial `main` após esta PR.
**Escopo operacional agora:** somente validação local + preflight SOMENTE LEITURA da VPS se houver acesso, sem reconfigurar webhook.

## Contexto já resolvido pelo ChatGPT
- O registro do Antigravity de 22/10/2026 descreve 6 mensagens Meta CTWA nos últimos 50 webhooks do status-engine. Este dado **não prova que o bot respondeu**: a Evolution Go continua apontando ao status-engine até cutover autorizado.
- Gatilhos literais no arquivo `config/ads-triggers.txt`:
  1. `Quero conhecer os pacotes do Primeiro Natal em Tijucas`
  2. `Quero conhecer os pacotes de Natal em Tijucas`
- Para cada frase: texto com primeiro nome opcional → áudio PTT → imagem/vídeo opcionais → CRM 416 sem nome por `POST /api/campaigns/schedule/execute`.
- Sem match: saudação/menu no **mesmo número Ads 48 3220-0440** (imagem/caption ou texto). O número 48 99642-5287, ainda não pareado na GO, será saudação somente. CRM usa `instance` numérica própria.
- Eventos `@lid` sem JID telefônico válido NÃO enviam nada e NÃO são convertidos em contato CRM. Não assumir que a GO sempre os retransmitirá com telefone.
- Para evitar spam, o primeiro passo de texto Ads marca o início da tentativa no contato, com cooldown configurável de 24h; outra sequência comercial do mesmo contato fica suprimida durante a janela, inclusive áudio/mídia/CRM. A lead claim CRM 416 BRT continua independente.
- `UpdateContact` atômico mantém `OptOut` e `HumanPaused` mesmo se um envio e um webhook `PARAR` ocorrerem próximos.
- Go oficial recomenda `go vet`, `gofmt` e testes com race detector; CI já cobre `go test -race` em outbox/ingress/relay.

## Executar APENAS isto no Windows local
1. Na pasta `C:\Users\Anderson\Desktop\BotWhats`, verificar `git status -sb`. Preservar mudanças locais (NUNCA `reset --hard`, `clean -fd`, force-push).
2. Quando o PR estiver integrado, atualizar por `git pull --ff-only origin main`.
3. Ler `AGENTS.md`, `docs/CTWA-ROUTING-20261022.md`, `docs/RELAY-ROLLOUT-GATES.md` e `.agents/skills/ctwa-ads-funnel/SKILL.md`.
4. Conferir diferenças de formatação apenas dos arquivos modificados; aplicar `gofmt -w` somente nos Go novos/modificados caso necessário, revisando o diff antes de commitar. Evitar reformatar outros pacotes.
5. Executar exatamente **UMA** validação local integral:
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-local.ps1
```
Se um pacote falhar, corrigir o arquivo causal e repetir a suíte uma vez ao terminar. Não executar testes sem fim.
6. Testes mandatórios já versionados: `cmd/botwhatsapp/real_creatives_test.go`, `internal/outbox/ads_repeat_test.go`, `internal/core/ctwa_test.go`, `internal/outbox/ctwa_flow_test.go`. CI remota roda também race detector.
7. Conferir `.env.example`: `BOT_ADS_FLOW_COOLDOWN_HOURS=24`, `BOT_ADS_TRIGGERS_FILE=/app/config/ads-triggers.txt` no container, `BOT_ADS_FALLBACK_MENU=true`; `BOT_ENABLED`, `BOT_OUTBOUND_ENABLED`, `BOT_OPERATIONAL_AUTHORIZATION` e `RELAY_ENABLED` permanecem **false**.
8. Caso a VPS seja acessível, executar somente inventário de leitura e verificar **sem expor tokens**: status-engine idempotência/receipts, versão GO real, rede interna, disponibilidade de spool, tamanho de payload, observabilidade, configuração antiga do webhook e instâncias conectadas. Não executar `up`, reinício ou API de mudança.

## Gate para homologação e deploy FUTURO
- **NÃO** mover porta 8050 nem alterar Caddy, Asterisk, SmartDialer, Postgres, Evolution Go ou CRM.
- Instância Ads `anderson` atualmente pertence ao status-engine: nenhum cutover antes de prova de idempotência de status-engine, relay durável respondendo e rollback testado, quota/backlog monitorada.
- Áudio oficial OGG Opus e artes aprovadas devem existir em volume privado; sem assets o bot suprime a mídia em vez de inventá-la. Emissões de campanhas reais demandam aprovação operacional e contato teste autorizado.
- Não executar novo POST CRM 416 do canário anterior; lookup GET com timeout não autoriza repetição.
- O relatório original expôs um possível token GO; recomendar conferência e rotação privada caso ainda válido, sem reproduzir valor.

## Entrega compacta
SHA local, diferenças Go Windows vs CI, PASS/FAIL `go vet/test/build/compose`, resultados das duas frases, teste da proteção contra clique repetido e **lista objetiva dos blockers de deploy**. Não criar dezenas de documentos, branches ou PRs. Se tudo passar, encerrar a validação sem habilitar produção.
