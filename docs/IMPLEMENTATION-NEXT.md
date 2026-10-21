# Próxima fase de execução para Antigravity Flash Lite

## Estado ATUAL verificado no GitHub
O código `internal/core/`, `internal/workflow/`, `internal/crm/`, `internal/evolution/`, `internal/ingress/` e `cmd/botwhatsapp/` compõe um **receptor de PREVIEW, sem efeitos externos**. `go test ./...` deve rodar localmente e no GitHub Actions. O endpoint `/readyz` sinaliza que o PREVIEW está ligado, não que está apto a disparar.

### Passo A — validar a base, SEM acessar VPS
1. `git status --short` e `git pull --ff-only origin main` **após salvar eventuais alterações locais** (não descartar mudanças).
2. `go version` ou usar container efêmero de build local; `go test ./...` UMA vez.
3. Comparar evento artificial e fixtures: `internal/core/core_test.go`, `internal/ingress/server_test.go`.
4. Ler `docs/FORENSIC-AUDIT-2026-10-21.md`, `docs/CRM-CONTRACT.md`, `.agents/skills/evolution-go-v072-adapter/SKILL.md`.

### Passo B — desenvolver engine durável (sem envios reais)
1. Separar **por telefone+campanha** idempotência de lead da dedup **por messageID**.
2. Criar journal de efeitos/outbox com passo e estado: `pending` -> gravar `attempt_started` em fsync -> enviar -> `confirmed` ou `uncertain`. Depois de crash em `attempt_started`, manter bloqueado até reconciliação; NÃO retransmitir.
3. Worker de baixa concorrência (1–2 workers) com limite por instância/telefone; webhook registra de forma durável antes do ACK e NÃO efetua HTTP de saída dentro do handler.
4. Persistir cooldown de saudação, estado de atendimento humano e opt-out, e criar ferramenta de diagnóstico que **não exibe números**.
5. Separar chamadas: texto, áudio, CRM. A falha no áudio não deve bloquear CRM, mas primeira resposta deve ser tentada antes de campanha. `campaign=415` apenas quando nome validado; `416` caso contrário.
6. Não enviar `instance=null` no CRM real; confirmar o ID de `message_instances` que deve disparar proposta. Sem isso, deixar bloqueado.
7. Mídias reais: formato/bytes/hash e local seguro. Áudio PTT usa `type=audio`; imagem `type=image,caption`. Não versionar fotos privadas nem áudio do gestor.
8. Tests mocks do ciclo inteiro, crash/restart, timeout ambíguo, duplicados e concorrência. Não fazer integração real ainda.

### Passo C — provisionamento CONTROLADO e CANÁRIO
Somente após autorização explícita do gestor e conferência da rede Docker:
- Confirmar se os números podem ser pareados nesta instalação sem conflito com outro provedor e se recebem mensagens.
- Criar duas instâncias próprias, sem modificar a instância `Anderson`.
- Configurar webhook interno de cada instância; `WEBHOOK_URL` global verificado, e nenhum consumidor existente modificado.
- Testar mensagens reais para número autorizado, com flags da campanha inicialmente OFF.
- Confirmar campanha no CRM e instância de SAÍDA antes de um único POST de teste aprovado.
- Documentar rollback que remove SOMENTE a nova stack/rotas, nunca Evolution/SmartDialer/Status.

### Gates
**P0:** nenhum segredo no Git; teste offline PASS; dedup/crash handling PASS; UUIDs/instâncias verificadas; CRM campaign e instance live verificados; mídia hash e autorização; rede isolada.
**Não fazer na próxima rodada:** reconfigurar Caddy, instalar Redis ou MySQL extra, interagir com clientes reais, editar o CRM, reiniciar qualquer serviço existente.
