---
name: ctwa-ads-funnel
description: Validar e manter o roteamento real CTWA Instagram e menu de saudação do Estúdio Evidência (Ads 48 3220-0440 e futura Saudação 48 99642-5287) sem enviar mensagens.
---
# Skill CTWA — Antigravity Flash Lite

Leia `docs/CTWA-ROUTING-20261022.md`, `AGENTS.md` e `config/ads-triggers.txt` antes de editar.

1. Frases editáveis em arquivo UTF-8, **uma frase completa por linha**; `BOT_ADS_TRIGGERS_FILE` tem prioridade sobre o env `BOT_ADS_TRIGGERS`. Jamais acrescentar match substring/regex sem aprovação: risco de enviar proposta 416 por falso positivo.
2. Instância Ads no presente: `anderson` / 48 3220-0440, UUID por env autenticada. Frases reais atualmente verificadas por relato de eventos Meta: **Quero conhecer os pacotes do Primeiro Natal em Tijucas** e **Quero conhecer os pacotes de Natal em Tijucas**. Ambas devem ser testadas; não presumir que o bot já recebeu os eventos só porque o status-engine recebeu. Esse payload usa `extendedTextMessage.text`, com `@lid` somente em SenderAlt — NÃO derivar número de LID.
3. Ads matched: texto com `{{nome}}` opcional, PTT, imagem opcional/caption, vídeo opcional MP4/caption, CRM 416 sem nome. Se não matched: saudação (imagem e caption; fallback textual se não houver imagem). `AdsFallback=true` precisa chegar até `instances.Registry.Decide`, não só `core.Config`.
4. Role greeting no futuro: 48 99642-5287 com token/UUID independentes; só saudação/menu, sem agendamento automático.
5. Testar `testdata/ctwa-first-natal.json` 100% sintético; não copiar dados privados do relatório em fixtures (há segredo real de instância no texto compartilhado).
6. Testar bloqueio de segundo clique dentro do `BOT_ADS_FLOW_COOLDOWN_HOURS`, atualização atômica de contato preservando opt-out e `@lid` sem telefone (ignorar sem supor replay). Testar ordem real `PreviousTaskID`, CRM por último, opt-out, pausa humana, cooldown, mídia ausente e timeout incerto SEM calls reais.
7. Antigravity executa `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-local.ps1` APENAS uma vez depois de `git pull --ff-only` com árvore limpa. Não fazer SSH, alterar webhook, repetir campanha 416, enviar canário, deploy nem instalar ferramenta na Contabo.
8. Reportar Git SHA, PASS/FAIL e bloqueios operacionais, sem reescrever motor ou criar testes intermináveis.
