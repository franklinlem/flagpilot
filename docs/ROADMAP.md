# Próximos incrementos do FlagPilot

Atualizado em 27/10/2026. Franklin autorizou continuar a evolução após corrigir o ingresso da homologação. Prioridade: concluir a experiência do MVP e sua operação antes de segmentos/variantes ou novas dependências.

| Incremento | Entrega | Critério verificável |
|---|---|---|
| 8A — Segurança da conta | Trocar a própria senha pelo painel, com senha atual e confirmação da nova, encerramento de todas as sessões e auditoria organizacional | Senha atual incorreta/falha de auditoria não alteram conta; sessões antigas rejeitadas; login com senha nova; teste real em navegador |
| 8B — Backup supervisionado | Detectar backup atrasado/falha e acompanhar pelo Uptime Kuma existente | Alerta de atraso ensaiado com fixture; rotina normal recupera indicador; nenhum segredo/dump em alertas |
| 8C — Aplicação consumidora | Exemplo pequeno de backend/BFF com SDK e flag controlando uma função real | Alternância/rollout e fallback exercitados sem token secreto no browser; documentação de integração |
| 8D — Preparação de release | Consolidar critérios atuais do MVP, recuperação, capacidade e escolhas de licença/distribuição | Checklist atualizado com evidências e limites; decisões explícitas sobre publicação e produção |

A homologação permanece separada de produção. Publicação npm/registry, licença, produção e alterações no atualizador global do Traefik não são inferidas desta continuidade. Avaliação por SDK continua server-side. Sem A/B testing, segmentação complexa, cobrança ou SDK Go neste ciclo.

**8A–8D entregues nos seus respectivos escopos.** 8A está na homologação; 8B está no host físico/Kuma; 8C é exemplo local; 8D consolida critérios e decisões, sem aprovar produção.

- [8A — Segurança da conta](ACCOUNT_SECURITY_STATUS.md): sem migration/dependência nova; aceitação manual do formulário ainda pendente.
- [8B — Supervisão de backups](BACKUP_SUPERVISION_STATUS.md): monitor push ativo no canal backup hermes; atraso/falha/recuperação ensaiados com fixtures, backup real preservado.
- [8C — Aplicação consumidora](CONSUMER_STATUS.md): cotação com BFF/SDK, rollout e fallback aprovados contra API real descartável e Chromium. [Executar exemplo](../examples/consumer/README.md).
- [8D — Preparação de release](RELEASE_READINESS.md): critérios consolidados; licença/distribuição pública e produção continuam pendentes. [Notas internas](RELEASE_NOTES.md).

Próximo ciclo sugerido, ainda sem implantação produtiva autorizada: aceitação humana de conta/consumidor, confirmar recebimento dos alertas e planejar cópia/monitor independente; depois ensaiar recuperação e capacidade no destino autorizado. Somente após resolver critérios de release, definir versão candidata e promoção. Não antecipar segmentos/A-B testing como substituição destas pendências.
