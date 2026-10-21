---
name: two-bot-workflows
description: Projetar e implementar dois bots determinísticos sobre Evolution Go com integração CRM externa.
---
# Workflows — depois do preflight

Não iniciar esta skill antes de conclusão do inventário VPS e confirmação de versão/UUID da Evolution Go. Desenvolvimento somente em `hypeneural/botwhatsapp`.

**ADS, 554832200440**: receber evento Message; validar UUID, IsFromMe=false, conversa privada, mensagem nova e texto elegível; comparador normalizado e conservador com frase exata configurada (não inventar frases). Registrar dedup durável por UUID+tipo+ID. Ações independentes: (1) texto de recepção; (2) áudio gravado; (3) solicitar campanha existente no CRM via POST. Não bloquear ação CRM por falha apenas do áudio. Cada efeito tem estado próprio, sucesso/ambíguo/falha e política segura de reconciliação. Caso não seja anúncio, menu fallback opcional.

**GREETING, 5548996425287**: conversa nova ou cooldown vencido → enviar uma imagem com `caption` de `config/greeting-caption.txt` pelo `/send/media`. Menu 1 Natal; 2 agendamento; 3 pedidos; 4 localização https://ir.fotosdenatal.com/; 5 humano e pausa do bot. Reconhecer números/sinônimos determinísticos, `menu` reexibe conforme regra. Não enviar saudação repetida a cada mensagem.

**Geral**: serviço pequeno Go em container separado; SQLite/WAL persistente ou DB existente somente se adequado; tokens distintos e flags OFF. Sem n8n, Redis, IA/LLM, painel. Não tocar core Evolution, não tocar CRM. Nunca associar LID sem telefone validado à campanha. Não retry cego de POST CRM ou envio de mídia com resultado desconhecido. Webhook autenticado por segredo próprio/rede interna, não instanceToken.

Testes locais mockados: mensagem do anúncio, fallback, imagem+caption, menu, duplicidade, concorrência, restart, timeout CRM incerto, falha áudio independente, humana pausa. Somente canário autorizado depois de revisão.
