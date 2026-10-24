# Fundação — estado e evidências

Data: 24/10/2026. Incrementos 0, 1A e 1B entregues e validados **localmente no Debian**. CI do GitHub configurada, ainda não executada no GitHub: não há remoto associado/publicação nesta entrega.

## Entregas

- Monorepo npm workspaces, módulo Go único, manifests e lockfiles.
- Go/Chi com configuração validada, slog JSON, request ID gerado pelo servidor, erros Problem Details e graceful shutdown. Logs não incluem headers, query, corpo, path arbitrário ou detalhes de panic/DB.
- Live e ready em `/api/v1/health/*`. Ready exige banco e schema versão 1, dirty=false; nenhum auto-migrate no processo da API.
- PostgreSQL 16.13, pgx/sqlc e migration de identidade (organizations/users/memberships/sessions), sem usuários de login pré-criados. Usuário runtime sem DDL, papel migrador separado.
- React/TypeScript/Vite com página inicial e estado de conexão. SDK privado vazio, build ESM/CJS e declarações; nenhum cliente de avaliação antecipado.
- OpenAPI 3.1 para o escopo inicial com endpoints planejados identificados, tipos gerados e validação Spectral.
- Compose leve/completo, imagens multi-stage não-root, healthchecks, secrets apenas em .env local e CI sem deploy/publicação.
- ADR-001 a ADR-007, threat model, fluxos UX e runbook de migrations. Especificação atualizada para 1.1 conforme decisões aprovadas.

## Verificações executadas

| Verificação local | Resultado |
|---|---|
| `make generate` e comparação com arquivos gerados versionados | Sem diferenças |
| `make ci` | Passou: gofmt/vet, ESLint, Spectral sem warnings, tipos, testes Go/race, Vitest, PostgreSQL real/Testcontainers e builds |
| Integração de migrations | Up vazio, no-change, down/up descartável, dirty/incompatível e perda de acesso ao DB detectados |
| Integridade e privilégio runtime | Email case-insensitive único, FKs, hash de sessão validado; CREATE TABLE pelo runtime negado |
| Build SDK e import/require | ESM/CJS importáveis; pacote ainda sem avaliação |
| `govulncheck ./...` | Sem vulnerabilidades após atualização de x/text |
| `npm audit --audit-level=high` | Zero vulnerabilidades reportadas |
| Gitleaks sobre histórico e diff staged | Sem vazamentos detectados |
| `make dev-containers` | DB, API e web healthy; portas publicadas somente em loopback |
| `python3 scripts/smoke.py` | Passou: live/ready, SIGINT com exit 0, reinício e queda real do DB; recursos descartáveis removidos |
| Imagem web runtime | Build passou; HTML/CSP, proxy da API e bloqueio de readiness/metrics verificados em container temporário |
| Vite não-root | Página e proxy respondem em 127.0.0.1:5173 |

O scan detectou inicialmente [GO-2026-6629](https://pkg.go.dev/vuln/GO-2026-6629) em golang.org/x/text 0.40.0, alcançável via pgx. A dependência foi fixada em 0.41.0; scan e testes subsequentes passaram. TypeScript 5.9.3 foi mantido por compatibilidade declarada do gerador OpenAPI; não usamos force/legacy-peer-deps.

## Operação local deixada disponível

- Painel inicial: http://127.0.0.1:5173
- API: http://127.0.0.1:8080/api/v1/health/live
- DB: loopback 5432, volume de desenvolvimento preservado.
- Encerrar: `make down`, sem remover volume.

Ferramentas em `.tools` e credenciais em `.env` (0600) não são versionadas. Smoke/Testcontainers usam recursos e credenciais sintéticos separados; não alteram outros projetos.

## Limites e próximos incrementos

Não há login, bootstrap CLI, RBAC em execução, CRUD, flags, auditoria operacional, avaliação, rollout, métricas Prometheus ou cliente SDK. Os contratos e ADRs dessas áreas descrevem a direção futura. Nenhum teste de isolamento de organizações é reportado como aprovado antes de existir autorização.

A interface foi verificada por build/HTTP; aceitação visual manual, teclado/a11y e Playwright de fluxos de negócio ficam para as telas funcionais. Não houve benchmark, backup/restore, validação de homologação ou scan de imagens de release. Imagens base estão fixadas por tag; digest/SBOM e hardening do destino ficam para release. Nenhuma licença foi aplicada, pacote/imagem publicado, deploy feito ou servidor externo alterado.

Próximo incremento: 2A, identidade administrativa com bootstrap auditável, Argon2id, sessões, login/logout/me, CSRF e testes. Auditoria deve ser antecipada antes da primeira mutação administrativa; não implementar bootstrap não auditado.
