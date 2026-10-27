# Próximos incrementos do FlagPilot

Atualizado em 27/10/2026. Franklin autorizou continuar a evolução após corrigir o ingresso da homologação. Prioridade: concluir a experiência do MVP e sua operação antes de segmentos/variantes ou novas dependências.

| Incremento | Entrega | Critério verificável |
|---|---|---|
| 8A — Segurança da conta | Trocar a própria senha pelo painel, com senha atual e confirmação da nova, encerramento de todas as sessões e auditoria organizacional | Senha atual incorreta/falha de auditoria não alteram conta; sessões antigas rejeitadas; login com senha nova; teste real em navegador |
| 8B — Backup supervisionado | Detectar backup atrasado/falha e acompanhar pelo Uptime Kuma existente | Alerta de atraso ensaiado com fixture; rotina normal recupera indicador; nenhum segredo/dump em alertas |
| 8C — Aplicação consumidora | Exemplo pequeno de backend/BFF com SDK e flag controlando uma função real | Alternância/rollout e fallback exercitados sem token secreto no browser; documentação de integração |
| 8D — Preparação de release | Consolidar critérios atuais do MVP, recuperação, capacidade e escolhas de licença/distribuição | Checklist atualizado com evidências e limites; decisões explícitas sobre publicação e produção |

A homologação permanece separada de produção. Publicação npm/registry, licença, produção e alterações no atualizador global do Traefik não são inferidas desta continuidade. Avaliação por SDK continua server-side. Sem A/B testing, segmentação complexa, cobrança ou SDK Go neste ciclo.

8A usa os componentes existentes (Go/Argon2id/sessão/CSRF, React Hook Form/Zod), sem migration ou dependência nova. UI em Conta → Segurança da conta, endpoint POST /api/v1/auth/password. Código/testes e decisões em [ACCOUNT_SECURITY_STATUS.md](ACCOUNT_SECURITY_STATUS.md) e [ADR-018](adr/018-self-service-password.md). Os demais incrementos são planejamento, não entregas já configuradas.
