---
name: bot-offline-simulator
description: Simula sem rede real os workflows de anúncio Meta Ads, saudação, menu e opt-out com fixtures sintéticas da Evolution Go 0.7.2. Use para validar rapidamente a lógica no Antigravity Flash Lite.
---
# Simulador offline — rápido para Flash Lite

**Nunca envia WhatsApp, nunca chama o CRM, não usa tokens nem a VPS.**

Da raiz do clone:
```bash
go test ./...
go run ./cmd/simulate ./testdata/ads.json
go run ./cmd/simulate ./testdata/greeting.json
go run ./cmd/simulate ./testdata/menu-human.json
go run ./cmd/simulate ./testdata/opt-out.json
```
Resultados esperados:
- `ads.json`: `ads_proposal` e 3 passos (texto, áudio, CRM); payload apenas DEMONSTRAÇÃO com `instance=999999` (fictício, jamais enviar).
- `greeting.json`: `greeting` com imagem+caption planejada (a mídia real ainda não foi fornecida).
- `menu-human.json`: `menu_option=5`, resposta e pausa humana planejada.
- `opt-out.json`: `ignore` com motivo de opt-out; a persistência definitiva da supressão é pendência de produção.

Se o Go local não existir, ler as fixtures e os testes; não instalar ferramentas na VPS compartilhada. Não confundir simulação com canário de produção.
