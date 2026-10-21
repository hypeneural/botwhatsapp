# Auditoria passiva — VPS Contabo com múltiplos serviços

## Escopo permitido
Apenas inventário técnico sanitizado de SO, recursos, serviços, contêineres, tags, redes e portas. Não acessar mensagens nem logs brutos. Não ler `.env`, dados de bancos, chaves ou `docker inspect` integral.

## Procedimento local (Windows + Antigravity)
1. Na raiz do clone, ler `AGENTS.md` e o script `scripts/vps-readonly-inventory.sh` antes de executar.
2. Só com alias SSH já configurado e autorizado, testar `ssh -o BatchMode=yes -o ConnectTimeout=8 SEU_ALIAS_SSH 'printf ok'`.
3. Com acesso autorizado e após revisar o script: em terminal que suporte redirecionamento stdin para SSH, executar `ssh -T SEU_ALIAS_SSH 'bash -s' < scripts/vps-readonly-inventory.sh`. Em PowerShell antigo sem redirecionamento de stdin compatível, usar Git Bash; não improvisar envio de credenciais.
4. Revisar saída local; não colar bruta no chat nem enviar ao Git. Registrar apenas inventário sanitizado e riscos em `reports/vps-inventory.redacted.md` (ignorado pelo Git).
5. Mapear serviços não relacionados à GO para evitar colisão de porta, proxy, rede e reconfiguração de webhook. O script não realiza alterações.
6. Para identificar digest/revisão exata, solicitar nova aprovação para comandos de metadados específicos do container GO; não executar `docker exec` nem abrir secrets.
7. Se não houver acesso: deixar versão/UUID NÃO CONFIRMADOS e finalizar auditoria offline do repositório.

## Proibições
`docker compose up/down`, `docker restart`, `docker stop`, `docker prune`, `apt upgrade`, `systemctl restart`, reiniciar a VPS, alterar firewall, executar `/instance/connect`, apagar/recriar sessões, enviar WhatsApp ou campanha.

## Entrega
Inventário sanitarizado: horário, SO, capacidade, contêineres pertinentes, infraestrutura compartilhada, riscos e próximas validações; nunca registrar tokens ou payloads.
