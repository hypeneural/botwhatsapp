# Prompt Antigravity — funil unificado, fase 1 LOCAL, sem deploy
Use `hypeneural/botwhatsapp`, branch PR #10 `fix/ctwa-native-proposal-handoff-20261027` e **PR stacked da issue #11**. Preserve worktrees, branches e o P0. Leia primeiro `docs/FUNIL-UNIFICADO-BOT-CRM-META-20261027.md`.

1. `git status --short`; `git fetch origin`; confira ancestry e diferenças. Não force push, reset, clean, nem sobrescreva dados locais.
2. Valide a PR #10 primeiro, inclusive token Sanctum, spool, 5287, integração nativa CRM e seus testes; estado draft ≠ produção.
3. Inspecione payloads sintéticos da Evolution Go 0.7.2 e determine se o ad ID realmente chega em `referral.source_id`, `externalAdReply` ou outro campo. NÃO invente campo; texto do anúncio não comprova ad_id. Eventos @lid sem E.164 verificado não podem ser associados.
4. Faça um patch pequeno exclusivamente no bot que registre identificador do anúncio quando legítimo e fonte/prova, preservando schema histórico e não persistindo payload bruto, ctwa_clid ou telefone em Git.
5. Escreva testes para sem referral, source_type diferente de ad, trigger textual, entrada duplicada, from_me, LID-only e payloads malformados.
6. Execute `go test ./...`, `go vet ./...`, `go test -race ./internal/ingress ./internal/dashboard` se Go compatível no checkout. Registre contagem de testes e SHA. Em bloqueio, NO-GO objetivo, sem loop de testes.
7. Entregue somente relatório de evidências/diff e próxima fase. Não conecte Meta nem CRM LIVE, não envie mensagens, não altere servidores, campanha, orçamento, DNS, tokens ou estado da PR #10.

Codex é responsável por API read-only dedicada no CRM, separadamente. Não alterar Laravel nesta execução.
