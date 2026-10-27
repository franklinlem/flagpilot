# FlagPilot — prontidão do MVP

Atualizado em 27/10/2026, após 8A–8D. O MVP funciona na homologação autorizada em https://flagpilot.franklem.uk. Este registro não declara release 1.0, produção ou garantia de SLO/RPO/RTO. Detalhes históricos permanecem nos status de cada incremento e no Git.

## Entregue e verificado

| Área | Evidência atual |
|---|---|
| Identidade e organizações | Sessão segura, CSRF/origem, rate limit, bootstrap/recuperação via CLI, papéis organizacionais, membros e proteção do último Owner. Aceitações manuais anteriores registradas. |
| Segurança da própria conta | Troca de senha no painel com senha atual, confirmação da nova, revogação de todas as sessões e auditoria organizacional atômica. Integração de falhas/concorrência e browsers aprovados; formulário ainda sem aceitação manual de Franklin. [8A](ACCOUNT_SECURITY_STATUS.md). |
| Projetos/ambientes | Criação/consulta, presets atômicos e isolamento. Edição/exclusão não fazem parte das entregas atuais. |
| Flags e avaliação | Booleanos, arquivamento, estado/rollout por ambiente, precondição If-Match e confirmação de produção, tokens com expiração/revogação, single/batch e bucketing Go/JS determinístico. |
| SDK | Pacote privado Node/TypeScript/JavaScript 0.1.0, ESM/CJS/tipos, timeout/retry limitado/cache/fallback; 16 testes e uso contra API real descartável. Exemplo local de cotação com BFF, rollout e fallback aprovado contra API real descartável/Chromium em [8C](CONSUMER_STATUS.md). |
| Dashboard | Filtros de catálogo, histórico organizacional com paginação/filtros e projeção de campos conhecidos. Foco, títulos, erros, teclado e mobile exercitados; não equivale a certificação WCAG. |
| Testes atuais | make ci completo local: Go/race/vet, integração PostgreSQL, 24 testes frontend, 16 SDK, OpenAPI/tipos/lint e builds. Segurança da conta passou em Chromium/Firefox; suíte administrativa Chromium passou. CI GitHub final de 8A (40538304391, ae94310) aprovada, incluindo quatro suítes administrativas/segurança Chromium/Firefox. Novos testes 8B/8C têm resultado próprio. |
| Imagens | Gate final de 27/10 sem HIGH/CRITICAL, incluindo correção de curl/libcurl no web. API mantém dois UNKNOWN de OpenPGP não importado, triados em 6F e preservados no relatório. [Inventário final](evidence/images-account-final-2026-10-27.json). |
| Infraestrutura | Coolify/Contabo, projeto/ambiente/banco próprios de homologação; TLS público e interno do DB, roles separadas, sem DB/API publicados. Apenas web no ingresso coolify, explicitamente selecionado no router Traefik. [7A](HOMOLOGATION_STATUS.md), [7B](INGRESS_STATUS.md). |
| Preservação na atualização | Em 8A, somente API/web recriados; schema 5, DB/migrator/proxy e demais containers preservados. [Verificação externa](evidence/account-homolog-2026-10-27.json). |
| Backups | Dump custom criptografado com GPG, pull SSH restrito ao frank-note2, cron diário 03h15 (-03), retenção de 14 dias. Execuções agendadas de 26 e 27/10 observadas e cópia adicional antes de 8A. Chave privada GPG de recuperação fora dos servidores. Supervisão 8B e rotação do log instaladas; ensaio de atraso/falha/recuperação aprovado, [estado](BACKUP_SUPERVISION_STATUS.md). [Runbook](runbooks/backups.md). |
| Recuperação | Backup real de 25/10 restaurado isoladamente com as 11 tabelas idênticas; destino removido. Ensaios sintéticos adicionais cobrem invalidação de sessões/tokens e rollback compatível. Não equivale a RTO produtivo. |
| Monitoramento | Uptime Kuma no frank-note2, HTTPS/API e backup supervisionado UP, com canal backup hermes vinculado. Listener Prometheus interno e logs estruturados disponíveis; coletor/dashboards de métricas ainda não configurados. |
| Capacidade | Ensaio local k6 de 6D com cotas e limites ativos, rejeições separadas de sucessos; resultados em CAPACITY_STATUS.md. Não mede capacidade sustentada da VPS. |
| Versionamento | Repositório GitHub privado, histórico verificado com Gitleaks. Sem deploy automático, publicação npm/imagens ou release pública. |

## Critérios ainda abertos

| Prioridade | Trabalho necessário |
|---|---|
| Operação independente | Supervisão 8B entregue, porém frank-note2 precisa estar ligado/conectado; backup/Kuma compartilham host e ainda não há segunda cópia externa automática. |
| Integração do produto | Exemplo local 8C entregue; aceitação humana e integração numa aplicação de uso real, com autenticação própria, ainda são etapas diferentes. |
| Operação | Provar entrega real de alertas, renovar certificado interno/chave GPG antes do vencimento, acompanhar espaço/logs e definir procedimento de recuperação operacional. Coleta Prometheus e métricas/alertas agregados permanecem pendentes. |
| RPO/RTO/capacidade | RPO ≤24 h e RTO ≤4 h são alvos propostos sem comprovação produtiva. Medir recuperação e carga no ambiente de destino com autorização específica. |
| Segurança/release | Assinaturas/proveniência, digests de registry, política de release/rollback de schema incompatível e avaliação de segurança mais ampla. Não há afirmação de ausência absoluta de vulnerabilidades. |
| Publicação e produção | Escolher licença e estratégia de distribuição, aprovar produção explicitamente e manter banco/volume/segredos/rotas separados. Não promover homologação automaticamente. |
| Produto e UX | Aceitação manual da troca de senha, revisão WCAG/leitor de tela/zoom e estados restantes; política de edição/exclusão e de retenção/limpeza de sessões/auditoria. Auditoria global de identidade ainda não existe; eventos de senha são organizacionais. |

Checklist consolidado em [RELEASE_READINESS.md](RELEASE_READINESS.md); preparação 8D não equivale a liberação produtiva.

Segmentação complexa, variantes, A/B testing, cobrança, SDK Go, SSO/MFA e aprovações de mudanças continuam pós-MVP. Sequência em [ROADMAP.md](ROADMAP.md).

## Reprodução local

```sh
make ci
python3 scripts/auth-e2e.py --browser chromium
python3 scripts/auth-e2e.py --browser firefox
python3 scripts/auth-e2e.py --browser chromium --suite security
python3 scripts/auth-e2e.py --browser firefox --suite security
```

Executar harnesses em sequência, com stacks/credenciais/volumes descartáveis próprios. Instalação de browsers e ensaios de recuperação/capacidade/release seguem os respectivos runbooks. Não usar dados reais do desenvolvimento ou credenciais de Franklin nos testes. A aceitação humana cobre somente os cenários relatados, separadamente das verificações headless.
