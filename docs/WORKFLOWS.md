# Matriz funcional — dois bots independentes

## A — Tráfego pago: 48 3220-0440
Gatilho: novo evento `Message` de chat privado, vindo de cliente, na instância cujo UUID for confirmado para esse telefone. Extrair texto do payload GO; reconhecer somente frases permitidas (configuração literal ainda pendente).

Ações:
1. Registrar ingresso idempotente por `(instance_uuid, event_type, message_id)`.
2. Responder texto acolhedor pela **mesma** instância de recebimento.
3. Enviar **áudio pré-gravado** pelo mesmo WhatsApp; confirmar suporte local a mídia/codec.
4. Registrar uma única intenção para `POST /api/campaigns/schedule/execute` no CRM. O CRM executará campanha Natal 2026 na própria instância de saída preexistente e aprovada.
5. Se falhar o áudio, não bloquear a solicitação CRM. Se a requisição CRM teve timeout após transmissão, não repetir automaticamente; reconciliar. Se CRM ainda não confirmou, não afirmar que o outro WhatsApp já enviou a proposta.
6. Texto desconhecido: saudação/menu opcional, com cooldown. Atendimento humano interrompe o bot.

## B — Saudação: 48 99642-5287
Gatilho: evento novo e elegível vindo de pessoa, em janela sem saudação recente e sem pausa humana.
Ação: `POST /send/media` `type=image` com `caption` de `config/greeting-caption.txt`.
Menu determinístico: 1 Natal 2026; 2 agendamentos; 3 fotos e pedidos; 4 localização `https://ir.fotosdenatal.com/`; 5 humano. Respostas completas dos itens 1–3 dependem de confirmação do gestor e dos destinos oficiais. Não adivinhar links. `menu` pode reabrir lista.

## Regras compartilhadas
- Números são públicos e **não devem ser usados como IDs de instância** no backend.
- Diferenciar GO instance UUID, JID do cliente, telefone com DDI da GO e telefone só DDD exigido pelo CRM.
- Rejeitar mensagem própria, grupo e eventos de status; tratar LID sem número mapeado como bloqueio do disparo comercial.
- Persistir estado, cooldown, pausa humana, recibos e efeitos incertos.
- Manter flags de execução OFF até canário autorizado.
- Sem webhook novo sobrescrever assinaturas existentes. Primeira tarefa do Antigravity é entender a topologia da VPS.
