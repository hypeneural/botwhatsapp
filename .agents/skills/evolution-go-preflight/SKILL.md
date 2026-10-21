---
name: evolution-go-preflight
description: Conferir versão instalada e contratos de webhook/HTTP da Evolution Go sem mexer nas sessões.
---
# Evolution Go preflight

Antes: ler `AGENTS.md`. A Evolution Go NÃO é Evolution API Node v2.

Código oficial: https://github.com/evolution-foundation/evolution-go
Arquivos: `pkg/routes/routes.go`, `pkg/sendMessage/service/send_service.go`, `pkg/instance/service/instance_service.go`, `pkg/events/webhook/webhook_producer.go`, `pkg/whatsmeow/service/whatsmeow.go`.

1. Apurar imagem/tag/digest/revisão real da instalação. Sem revisão identificável, escrever NÃO CONFIRMADO.
2. Comparar contra fontes oficiais da versão instalada, não usar tutoriais de Evolution Node.
3. No código auditado: `POST /send/text` {number,text,id?}; `POST /send/media` {number,type,url,caption?,id?}; `GET /instance/status`; `POST /instance/connect` {webhookUrl,subscribe:[\"MESSAGE\"]} — **MUTATIVO: NÃO EXECUTAR**.
4. Categoria `MESSAGE` gera evento `Message`. Envelope esperado tem `event,data,instanceId,instanceName,instanceToken`; `instanceToken` é segredo e não é assinatura!
5. Eventos duplicados podem vir de webhook GLOBAL + por instância. Investigar consumidores existentes por metadados, não mudar sua configuração.
6. Não inferir telefone real a partir de LID; mapear UUID de cada instância somente com evidência.
7. Relatório de contratos confirmados, dúvidas e riscos. Sem testes de envio real.
Referências: https://github.com/6justgotme/evolution-go-skill (MIT; conferir contra fonte da versão); https://github.com/estevao-tarifa/evo-chatwoot-bridge-go (MIT).


## Errata 2026-10-21
Segundo o inventário, versão anunciada na VPS é `0.7.2` com patch local para WebP animado; commit exato do patch ainda é desconhecido. Consulte `evolution-go-v072-adapter` e `docs/FORENSIC-AUDIT-2026-10-21.md`.

`WEBHOOK_URL` é a variável global do código oficial, NÃO `GLOBAL_WEBHOOK`. O relatório anterior não comprova sozinho que o webhook global esteja desativado. O campo de payload `instanceToken` é segredo, NÃO HMAC. Conferir volumes/consumidores e isolamento antes de instalar novas instâncias.
