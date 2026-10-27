# Preparação de release — incremento 8D

27/10/2026. Situação: **MVP em homologação; preparação concluída, liberação de produção pendente**. Este checklist não é uma release 1.0, tag de distribuição, autorização de produção nem promessa de disponibilidade.

## Checklist rastreável

| Critério | Estado e evidência |
|---|---|
| Fluxos do MVP implementados | Identidade/RBAC, organizações/membros, criação/consulta de projetos/ambientes, flags booleanas, rollout, avaliação single/batch, SDK e auditoria. [Prontidão](MVP_READINESS.md). “CRUD” não significa exclusão/edição completa de todo recurso. |
| Segurança da conta | Troca da própria senha e revogação transacional de sessões, testes de falha/concorrência e browsers. [8A](ACCOUNT_SECURITY_STATUS.md). Aceitação humana do formulário ainda pendente. |
| CI da base implantada | GitHub [40538304391](https://github.com/franklinlem/flagpilot/actions/runs/40538304391), ae94310, concluída com sucesso; inclui suítes administrativa/segurança em Chromium/Firefox. CI dos incrementos seguintes exige resultado próprio, não herda esse verde. |
| Imagens da base implantada | [Scan final 27/10](evidence/images-account-final-2026-10-27.json): zero HIGH/CRITICAL, dois UNKNOWN na API com triagem registrada; IDs exatos em [implantação](evidence/account-homolog-2026-10-27.json). Validade limitada àquele inventário/data, não scan permanente. |
| Isolamento/TLS/ingresso | [Homologação](HOMOLOGATION_STATUS.md), [ingresso](INGRESS_STATUS.md). Sem produção ou mudança de proxy global neste ciclo. |
| Backup supervisionado | [8B](BACKUP_SUPERVISION_STATUS.md): cron, retenção, arquivo real recente, checksum, alerta de atraso/falha e recuperação ensaiados. Kuma e backup compartilham host; leitura do alerta pelo destinatário não confirmada. |
| Restore/rollback | [Restore real 25/10](evidence/backup-homolog-2026-10-25.json), 11 tabelas idênticas; [reversão de API com schema 5](evidence/rollback-2026-10-25.json). Novo restore e tempo no destino produtivo ainda necessários; nenhuma reversão incompatível presumida. |
| Consumidor/BFF | [8C](CONSUMER_STATUS.md): cotação real controlada por flag, rollout e fallback com API parada; browser sem token no ensaio. Exemplo local não implementa autenticação produtiva. |
| Capacidade/SLIs | [Benchmark local](CAPACITY_STATUS.md); métricas internas disponíveis. Capacidade sustentada da VPS, coletor/alertas de métricas e SLO operacional não comprovados. |
| Distribuição | Repositório e pacotes continuam privados; sem licença pública, registry/npm público, assinatura ou proveniência de release. Decisões abaixo. |

## Decisões e fronteiras de publicação

- **Manter agora:** GitHub privado, distribuição por checkout/artefato local, atualização manual supervisionada da homologação. Código raiz mantém versão interna 0.0.0 e SDK privado 0.1.0; não inventar equivalência com release pública 1.0.
- **Licença pendente:** Apache-2.0 é proposta da especificação, ainda sem escolha expressa. Não criar LICENSE nem declarar código open source antes da decisão de Franklin. Inventariar licenças de dependências e obrigações de distribuição antes de publicação.
- **Distribuição pendente:** escolher entre uso privado e publicação autorizada. Para publicação, confirmar nome/escopo npm e registry, alinhar versões/OpenAPI/changelog, gerar SBOM e scan do commit final, assinar/vincular proveniência e registrar digests de registry dos artefatos exatos. Image ID local não é digest publicado.
- **Produção pendente:** requer autorização explícita de Franklin e ambiente próprio com domínio, banco/volume, roles, segredos e rotas independentes. Não converter homologação em produção nem reaproveitar seu banco/tokens.

## Ordem para liberar produção posteriormente

1. Franklin exercita troca da própria senha e exemplo consumidor; registrar cenários e eventuais correções. Confirmar escopo funcional, inclusive política futura de edição/exclusão e retenção de auditoria/sessões.
2. Definir SLO/RPO/RTO aceitos, segunda cópia/cofre de recuperação e monitor independente do frank-note2. Confirmar entrega dos alertas e calendário de restauração/renovação CA/GPG.
3. Com ambiente de destino autorizado, medir restore completo e fluxo funcional/carga limitada; registrar duração, integridade, limites e recursos. Alvos RPO ≤24 h/RTO ≤4 h ainda precisam ser demonstrados, não herdados do ensaio local.
4. Fixar commit candidato limpo e versões, CI completa verde, scans/SBOM atualizados, cobertura de avaliação/autorização e revisão das pendências de segurança/acessibilidade. Preparar matriz de compatibilidade de schema, backup anterior, comandos de promoção/reversão e responsável pela janela.
5. Preparar artefatos de release conforme licença/distribuição aprovadas. Submeter checklist e evidências concretas à aprovação final de produção; somente então provisionar/promover os artefatos verificados, acompanhar métricas e executar smoke.

O objetivo de cobertura ≥80% e WCAG 2.2 AA da especificação continua meta a medir/revisar; não há certificação ASVS/WCAG. Mais funcionalidades pós-MVP não resolvem estes critérios operacionais.

## Inconsistências esclarecidas

A especificação contém esboços e comandos históricos: “CRUD” excede as operações atualmente entregues; browser usa BFF e não SDK com segredo; campos de sessão/CSRF de exemplos antigos não são requisitos adicionais de configuração; retenção vigente é 14 dias e não o exemplo de 7 diários/4 semanais. `api/openapi.yaml`, `.env.example`, ADRs e runbooks atuais prevalecem como contratos implementados. Nenhum registry/deploy automático está habilitado apesar da direção futura descrita no plano. Publicação exige revisar esses exemplos e alinhar versões/hosts antes da tag final.
