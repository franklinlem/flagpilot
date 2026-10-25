# Incremento 3A — projetos e ambientes

Data: 24/10/2026. Desenvolvimento local no Debian, após autorização de continuidade. Registro de validação manual anterior: Franklin confirmou cursor normal e seletor funcional em organizações/membros; os demais fluxos relatados continuaram funcionando.

## Entregas

- Módulo Go de projetos/ambientes com APIs de criação/listagem e detalhe de projeto, auditoria transacional e SQL tipado por sqlc.
- Owner/Admin criam; Owner/Admin/Editor/Viewer leem no próprio escopo. Releitura de papel depois de adquirir lock, com proteção contra permissões revogadas durante espera.
- Projetos novos recebem development, staging e production atomicamente. Ambientes adicionais permitem classificação explícita de produção, inclusive com slug personalizado.
- Migration 000003 aditiva, slugs únicos no escopo, FKs RESTRICT e revisão ambiental zero. Readiness exige schema 3. Contas, senhas e associações anteriores preservadas.
- Navegação organização → projetos → ambientes, formulários validados, listas paginadas, identificação textual de produção e cache separado por identidade/escopo.
- OpenAPI e tipos gerados atualizados. Guards HTTP e paginação compartilhados sem adicionar dependências.

## Validação

`make ci` aprovado localmente: gofmt/vet, ESLint, Spectral sem avisos, TypeScript, Go com race detector, 11 testes frontend, integração PostgreSQL real e builds. Playwright Chromium: 4 fluxos aprovados (autenticação, cursor, organizações/membros e projetos/ambientes), em stack isolada e descartável. Inclui verificação básica de ausência de overflow em viewport de 390 px; não representa auditoria completa de acessibilidade/responsividade. Não há execução remota de CI nem deploy externo.

Cenários confirmados pelos testes adicionados: upgrade preservando identidade, up/down descartável, integridade, permissões, isolamento, slugs, paginação, rollback de presets/auditoria, revogação durante lock, formulário de produção e fluxo Chromium com Owner/Viewer. A aceitação visual/manual das novas telas permanece com Franklin.

## Uso e continuidade

Ver [projetos e ambientes](runbooks/projects.md) e [ADR-009](adr/009-projects-environments.md). A conta existente continua válida em http://127.0.0.1:5173.

O escopo inicial oferece criação e consulta. Edição/arquivamento/exclusão precisam de políticas próprias; não declarar CRUD completo. Próximo incremento: 3B, flags booleanas, configurações por ambiente, auditoria/revisão e concorrência. Ainda sem avaliação, tokens, SDK funcional, rollout ou implantação externa.

Containers de desenvolvimento atualizados com schema 3; DB/API/web saudáveis. Readiness HTTP 200 e frontend HTTP 200 após a atualização. Volume existente preservado, sem provisionar contas/projetos reais automaticamente. Gitleaks sobre alterações staged sem segredos. A verificação adicional das permissões SQL e códigos de constraints também passou em PostgreSQL isolado.

Aceitação manual posterior: Franklin confirmou criação dos projetos e atribuição automática dos ambientes, incluindo a identificação visual de produção, e autorizou continuar. Estado seguinte em [FLAGS_STATUS.md](FLAGS_STATUS.md).
