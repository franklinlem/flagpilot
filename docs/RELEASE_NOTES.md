# Notas internas — ciclo 8A–8D

27/10/2026. Registro para revisão do MVP em homologação; não representa release pública.

- Conta: troca da própria senha no painel, confirmação da senha atual, revogação de todas as sessões e auditoria organizacional.
- Operação: backups supervisionados no Kuma, detecção de falha/atraso/corrupção e ausência de heartbeat; rotação limitada do log próprio.
- Integração: aplicação local de cotação com BFF/SDK, alternância/rollout e fallback, sem token secreto no browser.
- Release: checklist consolidado, evidências e limites de recuperação/capacidade, licença/distribuição/produção explicitamente pendentes.

Sem migration neste ciclo; schema permanece em 5. Apenas 8A alterou runtime da aplicação implantada. 8B atualizou scripts/cron próprios do backup e adicionou monitor no host físico; 8C é exemplo local; 8D é documentação. Nenhum deploy de produção, publicação de imagem/pacote ou alteração de proxy global.

Compatibilidade do SDK continua 0.1.0 privado, sem alteração de API pública. Rollback de API foi ensaiado apenas com schema compatível; não executar migration down ou restore sobre banco ativo como atualização normal. Estado atual e critérios em [RELEASE_READINESS.md](RELEASE_READINESS.md).
