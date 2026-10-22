# Handoff obrigatório para Antigravity — canário, CRM 416 e multi-instância

**Escopo exato:** concluir auditoria dos três commits locais, conciliar PRs e preservar todos os serviços Contabo. Este roteiro **não autoriza novo disparo de campanha, mudança de webhook ou deploy**.

## Passo 1 — proteger os commits locais do canário
Execute no clone Windows, *sem perder o working tree*:

```powershell
git status -sb
git log --oneline --decorate --all -20
git show --stat --oneline 16514e4
git show --stat --oneline 10348a6
git show --stat --oneline ae40f97
```

Antes de criar branch, confira `git diff`, `git diff --cached` e arquivos untracked. Se houver alterações locais, preserve numa branch/commit seguro sem sobrescrever; não use `git reset --hard`, `git clean -fd`, `git push --force`. Não enviar mídia privada, `data/spool`, tokens ou `run_canary.cmd`.

Publique a árvore do canário na branch `feat/antigravity-outbox-canary-audit` e abra PR contra `main` (não fazer merge automático). Isso é necessário porque os 3 SHAs do relatório não estavam no GitHub remoto na auditoria.

## Passo 2 — auditoria do código REAL e do teste
Leia a skill `canary-forensic-review` e `docs/AUDIT-CANARY-AND-MULTIINSTANCE-2026-10-21.md`. Confronte worker/ingress/cmd da árvore local com resultados do canário. Verifique:
- A rota **real** de CRM no código deve ser `/api/campaigns/schedule/execute`; `/api/campaigns/schedule` do relatório está incompleta.
- Confirmação CRM 416: `id` na raiz do JSON + `campaign=416` + `instance.id` esperado.
- ID de texto/áudio confirma aceite do endpoint GO, não recibo de entrega WhatsApp.
- Replay do mesmo evento e segundo evento do mesmo lead são testes sintéticos; distinguir de entrada natural pelo webhook.
- Journal de intents por etapa, fsync, efeitos `attempt_started`, `uncertain`, proteção em crashes e concorrência.
- Bloqueio durável de opt-out (`PARAR`), cooldown e pausa humana, antes e DEPOIS do restart.
- PTT OGG/Opus é asset real? Revalidar metadados de formato localmente sem repetir envio.

## Passo 3 — confirmar rollback e CRM READ-ONLY
Sem executar POST:
- Conferir backend GO via consulta administrativa autorizada, somente comparando webhook atual da instância `anderson` com o destino anterior; **não imprimir tokens nem URLs com segredos**.
- Confirmar status-engine ativo e que continua recebendo eventos. UI snapshot de URL não é prova.
- Consultar `GET /api/campaigns/schedule/search` com campanha 416 e telefone canário autorizado, confirmar registro do ID de agendamento já recebido, instância/estado. Se não aparecer, marcar INCONCLUSIVO, nunca reenviar POST.
- Não tentar teste de tráfego do WhatsApp fora da autorização específica.

## Passo 4 — consolidar multi-instância e performance (OFFLINE)
Leia `multi-instance-contract` e `docs/MULTIINSTANCE-ARCHITECTURE.md`.
- PR #1 do CRM 416 ainda não estava integrada; nova PR do registry usa sua branch como base. Primeiro conciliar PR #1, depois PR do registro, e por último worker local após resolução de conflitos.
- Reutilizar `internal/instances.Registry`; NÃO reescrever worker nem mapear instance por nome/celular em cada handler.
- Isolar TokenEnv/SecretEnv/UUID/número/role de cada instância. Nunca registrar segredos.
- Dedup evento por UUID+event+message ID, claim comercial *compartilhado por todas instâncias* por tenant+nationalPhone+campaign416+data BRT.
- Integrar registry na camada de recebimento após conhecer exatamente o worker local; não manter 2 engines concorrentes.
- Propor distribuição durável dos eventos para status engine e bot, sem trocar webhook `anderson` em testes.
- Adicionar retenção para journal (sem varredura por evento), métricas de estados e limites de disco.

## Passo 5 — validação mínima
Execute localmente uma vez `go test ./...`. Se houver falha, resolver especificamente o pacote e rodar suite completa uma vez ao final. Não executar bateria de testes VPS, disparos CRM reais ou mudanças em Caddy/Tailscale/firewall.

Entregáveis:
1. Link/sha da PR do código original do canário.
2. Diagnóstico PASS/RISCO/BLOQUEIO com evidência de arquivo+linha; não usar `SUCESSO TOTAL` para aceite de API.
3. Resultado do teste local, revisão da rota real e GET read-only do CRM.
4. Plano de integração das branches sem apagar o trabalho do Codex/AntiGravity.
5. Rollback dos componentes NOVOS, preservando Status, Evolution GO e CRM.
