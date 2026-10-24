# Incremento 2B — organizações, membros e autorização

Data: 24/10/2026. Implementação e validação locais; CI configurada, sem execução no GitHub nesta sessão. Nenhum deploy/servidor externo alterado.

## Entregas

- GET/POST `/api/v1/organizations`, GET `/api/v1/organizations/{orgId}`, GET/POST de membros e PATCH/DELETE da associação por userId.
- Criador vira Owner da nova organização; somente associações próprias listadas/consultáveis. Owner gerencia membros; demais papéis somente leem. Desconhecidos negados por padrão.
- Último Owner ativo protegido por transação e lock; autorização relida depois da espera, sem confiar em papel antigo do middleware.
- Alteração e auditoria atômicas; cadastro local de usuário como Viewer via CLI create-user. Sem cadastro público ou diretório global de emails.
- Paginação cursor vinculada ao usuário/escopo, validação de JSON/UUID/papel/limites, sessão + Origin + CSRF nas mutações e erros padronizados.
- Navegação React Router, TanStack Query com cache por identidade/organização, nova organização, listagem, adição por ID, alteração de papel e confirmação de remoção. Sem retry automático de mutações.
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

Franklin validou manualmente a criação de senha/login do incremento 2A: captura exibiu conta autenticada, organização franklem e papel Proprietário. Isso confirma apenas esses cenários. As novas telas 2B ainda dependem da aceitação visual/manual do usuário; testes headless não substituem teclado/acessibilidade e browser humano.

Atualize http://127.0.0.1:5173. A conta existente continua válida. Ver [uso de organizações/membros](runbooks/organizations.md).

Próximo incremento: 3A, projetos e ambientes. Não implementados nesta etapa: flags/rollout/avaliação, SDK funcional, gestão global de usuários/desativação, hard delete de organizações, convites, auditoria consultável ou operação de produção.
