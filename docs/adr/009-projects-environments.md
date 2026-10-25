# ADR-009 — Projetos e ambientes iniciais

Data: 24/10/2026. Estado: implementada no incremento 3A, após autorização de continuidade de Franklin.

## Contexto e escopo

Organizações e membros foram validados manualmente, incluindo a correção do cursor e o seletor de usuários. O próximo passo do MVP é organizar projetos e ambientes sem antecipar flags, avaliação ou tokens. As operações de edição, arquivamento e exclusão ainda precisam de regras de retenção e concorrência; o contrato atual autoriza inicialmente criação/listagem. Este incremento entrega criação/listagem e detalhe de projeto para navegação, sem declarar CRUD completo.

## Decisões

- Módulo `internal/projects` contém projetos e ambientes. HTTP reutiliza os guards de sessão/Origin/CSRF existentes; paginação passa ao módulo interno compartilhado `pagination`, sem dependência entre domínios.
- Papéis organizacionais herdados: Owner/Admin criam projetos e ambientes; todos os papéis válidos leem. A associação ativa é consultada no banco em toda operação. Projeto externo retorna 404; papel insuficiente no próprio escopo retorna 403.
- Operações adquirem SHARE na organização e releem o papel depois de eventual espera. Criação de ambiente bloqueia também o projeto com NO KEY UPDATE, na ordem organização → projeto. Alterações de membros bloqueiam a organização com NO KEY UPDATE, preservando a autorização durante a operação. Criações futuras de flags precisam seguir a mesma serialização no projeto para inicializar configurações sem lacunas.
- PostgreSQL exige UPDATE em ao menos uma coluna para SELECT FOR NO KEY UPDATE. O runtime recebe UPDATE somente em `projects.id` para viabilizar esse lock, além de SELECT/INSERT; não recebe UPDATE de nomes/slugs/classificação nem DELETE nas novas tabelas. Não há endpoint de atualização de ID. As FKs RESTRICT preservam referências.
- Criar projeto cria automaticamente `development` (Desenvolvimento), `staging` (Homologação) e `production` (Produção), cada um com UUID próprio, revisão zero e auditoria. Qualquer falha reverte projeto, ambientes e eventos da transação inteira.
- Ambiente adicional exige `name`, `slug`, `isProduction` booleano explícito. O slug reservado `production` exige true no serviço e no banco; outros slugs também podem representar produção. Essa classificação fica persistida e destacada na interface, sem edição nesta etapa. A interface orienta marcar ambientes que atendam usuários reais. Não inferir classificação apenas pelo texto do nome.
- Nome/slug são imutáveis pela API inicial; slugs únicos por organização (projeto) e por projeto (ambiente). Sem renomeação que contorne classificação. Futuras mudanças de flags em produção exigirão confirmação baseada no booleano persistido, além da autorização do servidor.
- `environments.revision` é BIGINT não negativo, inicialmente zero. Nenhum contador é incrementado artificialmente antes de existirem configurações de flags. UNIQUE(id,project_id) prepara FKs compostas do próximo módulo. Revisão continua sendo do snapshot ambiental, não versão de edição de metadados.
- Migration 000003 aditiva; migrations anteriores preservadas. Readiness passa a exigir exatamente schema 3. Não apagar volumes nem executar down no banco de desenvolvimento para atualizar.

## Consequências e limites

Nenhuma nova dependência ou variável secreta. UI contém navegação, formulários, validação, estados vazios/erros, paginação, cache por identidade/escopo e marcação textual de produção. Confirmação de alterações de flags, políticas de edição/exclusão e revisão de configurações pertencem aos próximos incrementos. Não há projetos demonstrativos criados na conta real automaticamente.

Testes usam PostgreSQL real isolado e Chromium com contas descartáveis: isolamento, matriz de papéis, perda de permissão durante espera, unicidade no escopo, classificação de produção, paginação, rollback de presets/auditoria, upgrade com identidade preservada e navegação. Testes headless não substituem aceitação visual/manual.
