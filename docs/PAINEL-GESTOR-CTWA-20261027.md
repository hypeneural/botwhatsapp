# Painel do gestor — WhatsApp Evydência (revisão 27/10/2026)

> Estado: **PR #10 em revisão**, não equivale a implantação. O painel real consulta **somente os arquivos duráveis deste bot**, não a Meta Ads nem o CRM diretamente.

## Visão rápida (inclusive no celular)

Endereço do bot na rede privada: `/painel` (o domínio/porta dependem do deploy; **não** expor publicamente).
Entrar via Basic Auth no **HTTPS privado**; `BOT_DASHBOARD_TOKEN` é a senha, e o nome de usuário pode ser `gestor`. Não passar senha na URL ou registrar capturas com informações reais de clientes.

A página abre no **Histórico de atendimentos** e traz quatro contadores:
- **Mensagens recebidas**: eventos na janela atual, não o total de pessoas.
- **Identificadas como anúncio**: correspondência literal com o gatilho CTWA ou metadado `externalAdReply` / `referral.sourceType=ad`. É um sinal operacional, **não prova independente da Meta**.
- **Propostas aceitas pelo CRM**: despacho nativo com recibo registrado; **não indica entrega no WhatsApp do segundo número**.
- **Precisam de atenção**: eventos com recusa pelo CRM ou resultado incerto; conferir antes de qualquer reenvio.

O filtro de origem **Sem marcação de anúncio** não deve ser chamado de "orgânico": anúncios sem metadados conhecidos podem entrar nele. A busca procura nome, telefone, trecho do texto e gatilho **dentro da janela carregada**; selecione 50/150/500 para ampliar. Atualização padrão de 30 segundos, pausa automática quando a aba não está visível.

Ao clicar **Ver N etapa(s)** aparecem as etapas realmente persistidas: ingresso, texto, áudio, PDF, CRM, horários de tentativa, aceite/rejeição e identificadores. Na visão de celular as informações de cada evento aparecem em cartão vertical, sem tabela larga.

O bloco **Ver contatos agrupados por telefone** é opcional e só consulta o resumo após abrir; isso poupa trabalho de disco no acesso principal. Contato pode ter vários eventos, de modo que os totais dos dois painéis divergem legitimamente.

## Glossário do gestor

| Visualização | Interpretação | Próxima decisão |
|---|---|---|
| CRM aceitou | Retorno nativo com `dispatch_uuid` | Acompanhar processamento posterior da proposta |
| CRM rejeitou | Regra/validação recusou, incluindo HTTP 422 legado | Analisar elegibilidade, permissão ou contrato; não reenviar automaticamente |
| Resultado incerto | Timeout, 429, 5xx ou resposta inconclusiva | Consultar CRM/provedor antes de qualquer retentativa |
| Pendente / Em andamento | Tarefa durável não encerrada | Aguardar trabalhador, conferir saúde se persistir |
| Sem execução registrada | Evento recebido, nenhuma etapa na outbox (pode ser preview, supressão, lacuna histórica) | Consultar registro, não supor mensagem enviada |
| Atendimento local finalizado | Resposta do bot pela Evolution foi aceita; pode não haver CRM | Não equivale à entrega ou leitura |
| Ignorado / Contato pediu para parar | Sem ação por regra, texto sem gatilho ou opt-out | Respeitar a supressão |

**Aviso de origem:** o painel guarda até 280 caracteres do texto de entrada e tipo, e não armazena payload bruto, anexo ou credenciais do webhook. Registros históricos anteriores a este código podem não ter texto, tipo ou prova do primeiro ingresso.

## Preview em Windows — 100% fictício (sem VPS)

Na worktree `fix/ctwa-native-proposal-handoff-20261027`, basta executar **uma vez**:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1 -Format -Preview
```

O script confirma SHA/branch, formata **apenas** arquivos Go da PR, roda `go test -count=1 ./...`, `go vet`, compilação de bot + preview, confere `git diff --check` e, se Node instalado, `node --check` do JS extraído da página. Gera relatório sanitizado no diretório TEMP sem imprimir ou transmitir segredos de produção.

Só após PASS abre o servidor de demonstração `cmd/dashboard-preview`. O programa cria registros **fictícios** dos cinco cenários (CRM aceitou, CRM rejeitou 422, incerto, saudação normal, mensagem ignorada), escuta **apenas** 127.0.0.1 em porta aleatória e mostra no console URL e login **demo / demo-local**. Nunca aponta para `BOT_DATA_DIR` real nem para Evolution/CRM. Abrir URL no Chrome/Edge, testar:
1. Desktop: contadores, filtro **Precisam de atenção**, filtro de origem, gatilho e expansão do histórico de APIs.
2. Mobile: DevTools → largura 390 px; sem rolagem horizontal da tabela, cartões completos, labels legíveis, botões acessíveis.
3. Abrir **contatos agrupados**: só então carrega o resumo.
4. Fechar pelo **Ctrl+C**, que desliga o listener e apaga o diretório sintético.

Se faltar Node, a sintaxe JS retorna **SKIP** (nunca PASS). O script retorna **FAIL** em erro de teste, formatação sem `-Format` ou build. Em caso de falha, não executar deploy.

## Segurança e escalabilidade

- `/painel` e `/api/dashboard/*` exigem autenticação; CSP bloqueia recursos externos, `no-store` evita cache, página não pode ser embutida por terceiros.
- Endpoints são somente GET; não possuem botões de reenviar, reagendar ou apagar mensagens.
- Histórico de entrada: manutenção limita a 90 dias/30 mil arquivos; detalhes da outbox: limite independente de 30 dias/20 mil tarefas, conforme configuração. Nunca apagar manualmente arquivos de supressão/claims.
- O agregado de eventos tem snapshot em cache por até **8 segundos** por quantidade solicitada, evitando varreduras repetidas durante acessos simultâneos. Não afirma streaming em tempo real; o botão Atualizar relê o snapshot quando expira.
- O sistema de atendimento, a Evolution e o CRM podem ter status posteriores não observados por este bot; não inventar entrega.
- A UI é HTML/CSS/JS local sem dependências de frontend externas. Conteúdo do cliente é sempre escapado ao renderizar.

## Próximo passo operacional

Homologar localmente os cinco cenários no preview, revisar diferenças com PR #9 (código 5287 já incorporado na #10), comparar a instalação de produção em modo somente leitura e concluir gates de token Sanctum e autenticação privada. O canário de **um novo contato autorizado** deverá ser feito apenas após aprovação operacional. **Não reconstruir a interface, não fazer replay dos 13 erros 422 e não reiniciar a Evolution.**
