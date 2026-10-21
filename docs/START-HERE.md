# Primeira tarefa Antigravity Flash Lite — auditoria, NÃO implantação

Este repositório foi criado em 21/10/2026 para dois bots simples usando Evolution Go existente na VPS Contabo.

**Primeiro resultado esperado: relatório sanitizado da versão da Evolution Go e inventário de dependências da VPS.** Não instalar serviços, não executar testes com números reais e não mudar o CRM.

1. Ler `AGENTS.md` e executar a skill `.agents/skills/vps-readonly-audit/SKILL.md`.
2. Identificar contêiner/imagem/tag/digest Evolution Go, recursos, portas, redes e demais serviços da VPS por metadados; nunca copiar env/logs/DB/segredos.
3. Executar a skill `.agents/skills/evolution-go-preflight/SKILL.md`. Conferir `/send/text`, `/send/media`, `/instance/status`, envelopes de webhook e seus consumidores *na versão instalada*; `POST /instance/connect` é mutativo e proibido na auditoria.
4. Executar `.agents/skills/crm-readonly-contract/SKILL.md` apenas por consulta GitHub e APIs de leitura autorizadas.
5. Preencher um relatório local em `reports/vps-inventory.redacted.md` (ignorado pelo Git) e pendências abaixo.
6. Parar. Somente depois da análise haverá desenvolvimento seguindo `.agents/skills/two-bot-workflows/SKILL.md`.

## Pendências não resolvidas
- Versão/tag/digest, configuração e quantidade de instâncias reais Evolution Go.
- UUID seguro de cada instância ligada aos dois telefones.
- Webhooks globais e por instância e seus consumidores atuais.
- Texto literal do anúncio do Meta Ads.
- Áudio gravado, arquivo de imagem JPG/PNG e legenda final.
- URL de serviço, campanha Natal 2026 e ID numérico da instância de saída do CRM; validar política de disparos.
- SSH autorizado e quais serviços da VPS compartilham rede/proxy/portas.

## Matriz de identidade
| Telefone | Papel | No CRM |
|---|---|---|
| 48 3220-0440 | Inbound Meta Ads, texto+áudio e requisição ao CRM | Solicitar campanha configurada de Natal pelo número de SAÍDA já existente no CRM |
| 48 99642-5287 | Saudação: foto+caption e menu | Não acionar campanha automaticamente |

Os telefones **não identificam** por si só UUID Evolution Go nem instância de saída do CRM.
