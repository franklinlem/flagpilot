# Incremento 2B — organizações, membros e autorização

Data: 24/10/2026. Implementação e validação locais; CI configurada, sem execução no GitHub nesta sessão. Nenhum deploy/servidor externo alterado.

## Entregas

- GET/POST `/api/v1/organizations`, GET `/api/v1/organizations/{orgId}`, GET/POST de membros e PATCH/DELETE da associação por userId.
- Criador vira Owner da nova organização; somente associações próprias listadas/consultáveis. Owner gerencia membros; demais papéis somente leem. Desconhecidos negados por padrão.
- Último Owner ativo protegido por transação e lock; autorização relida depois da espera, sem confiar em papel antigo do middleware.
- Alteração e auditoria atômicas; cadastro local de usuário como Viewer via CLI create-user. Sem cadastro público ou diretório global de emails.
- Paginação cursor vinculada ao usuário/escopo, validação de JSON/UUID/papel/limites, sessão + Origin + CSRF nas mutações e erros padronizados.
- Navegação React Router, TanStack Query com cache por identidade/organização, nova organização, listagem, adição por seletor de email, alteração de papel e confirmação de remoção. Sem retry automático de mutações.
- Nenhuma migration nova: schema 2 preservado, próximo número SQL reservado ao módulo de projetos/ambientes. Conta, senha e organização locais de Franklin preservadas; não provisionar usuários reais automaticamente.

## Evidências

- Go unitário/race e matriz Owner/Admin/Editor/Viewer/desconhecido; cursor e limites.
- PostgreSQL real: isolamento em leitura/escrita, 404 externo/403 interno, todos os papéis, associação duplicada, paginação sem duplicação, cursor externo, rollback de auditoria e cadastro.
- Concorrência: duas autodespromoções deixam um Owner; operação aguardando lock rejeita papel revogado durante espera. Owner desativado não permite remover o último Owner ativo.
- Cookie existente perde acesso à organização removida sem perder outra organização; HTTP exige sessão/CSRF e não revela dados em erro de escopo.
- Vitest/Testing Library: login e erros preservados, logout e troca de conta não reaproveitam organização/papel da identidade anterior.
- Playwright Chromium, stack/contas descartáveis: login, cadastro local, nova organização, membros, alteração de papel, remoção/reinclusão, último Owner, Viewer sem controles de gestão, 403 em tentativa direta de promoção e 404 para organização externa.
- Lint/tipos/Spectral, builds, código gerado, scans de segredos e dependências e teste de terminal CLI. Scanner Go segue sem achado alcançável; aviso do módulo sobre OpenPGP não importado permanece documentado em AUTH_STATUS.

## Aceitação e próximos passos

Franklin validou manualmente a criação de senha/login do incremento 2A: captura exibiu conta autenticada, organização franklem e papel Proprietário. Isso confirma apenas esses cenários. Franklin também exercitou as telas 2B e relatou que os demais fluxos funcionaram, solicitando correção do cursor ao salvar papel e um seletor de usuários existentes. Isso não representa validação manual de todos os cenários de segurança/acessibilidade.

Atualize http://127.0.0.1:5173. A conta existente continua válida. Ver [uso de organizações/membros](runbooks/organizations.md).

Próximo incremento: 3A, projetos e ambientes. Não implementados nesta etapa: flags/rollout/avaliação, SDK funcional, gestão global de usuários/desativação, hard delete de organizações, convites, auditoria consultável ou operação de produção.

## Correções após uso manual

- Cursor padrão em botões sem alteração; progresso e texto Salvando… somente enquanto a operação está pendente. Causa reproduzida: CSS aplicava cursor wait a todo botão desabilitado, mesmo depois de salvar.
- Seletor por email e endpoint Owner-only de candidatos, com paginação e exclusão de membros atuais/desativados. Visibilidade limitada a organizações compartilhadas com o ator ou histórico de associação do destino (ADR-008); permite reincluir membro removido sem copiar ID.
- Contrato OpenAPI e cliente gerado atualizados; sem migration ou alteração de contas/senhas existentes.
- Validação adicionada: regressão do cursor no Chromium com CSS real, ciclo de salvamento e liberação após erro, isolamento/permissões/paginação de candidatos no PostgreSQL e reinclusão pelo seletor no E2E.

Verificação desta correção: `make ci` aprovado (9 testes frontend, Go/race, integração PostgreSQL, lint/contrato/tipos e builds); 3 cenários Playwright aprovados em stack descartável, incluindo cursor e reinclusão pelo seletor. Gitleaks staged sem segredos. Containers locais atualizados e saudáveis, preservando o volume existente. Sem execução remota de CI.

Aceitação manual posterior: Franklin confirmou que o ponteiro voltou ao normal e que o seletor funciona, e autorizou continuar. Próxima entrega em [PROJECTS_STATUS.md](PROJECTS_STATUS.md).
