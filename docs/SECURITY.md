# Segurança, privacidade e rollback
- Não versionar tokens, segredos, dados de clientes, áudios privados, fotos, bancos nem inventário bruto da Contabo.
- GO inclui `instanceToken` no corpo do evento: não constitui assinatura de origem, e não deve ser logado. Proteger receptor por rede interna ou segredo independente.
- Cada instância tem seu token; o número de entrada não implica que a proposta seja enviada pela Evolution Go: o CRM utilizará a própria integração já existente.
- Evitar duplicidade: o produtor de webhook da GO pode entregar global e por instância, repetir após erro de HTTP; dados devem ser persistidos antes do ACK.
- Resultado incerto de envio deve permanecer `uncertain`. Não enviar outra campanha/mensagem às cegas.
- Não alterar a VPS ou outros serviços antes do inventário.
- Rollback do futuro bot deve afetar apenas o novo serviço. Não reiniciar nem reconfigurar GO/CRM.
