# FlagPilot

Plataforma self-hosted de feature flags em desenvolvimento. Incremento atual: tokens e avaliação booleana com autorização sobre a fundação Go/Chi, PostgreSQL/migrations e React/Vite. **Login/logout, bootstrap, organizações, membros, projetos, ambientes e gestão de flags, tokens auditados e avaliação booleana já funcionam.** Rollout percentual e cliente SDK virão nos próximos incrementos.

## Requisitos e versões

Debian Linux, Docker Engine com Compose v2+ acessível pelo usuário, Python 3.12+ e GNU Make. Node **24.22.1** (também em `.nvmrc`), Go **1.27.3**, PostgreSQL **16.13**, sqlc **1.31.1**, golang-migrate **4.20.1**. TypeScript **5.9.3** foi escolhido por compatibilidade com o gerador OpenAPI; React **19.3.1** e Vite **8.3.5**. Dependências npm e Go fixadas em lockfiles.

Não é necessário instalar Go globalmente. O instalador abaixo baixa Go/sqlc/migrate/gitleaks para `.tools`, verifica SHA-256 e não usa sudo; suporta Linux amd64. Se já usar Go global, mantenha a versão indicada. Node pode ser selecionado com seu gerenciador habitual (`nvm use`, se instalado). Não instalamos nem reconfiguramos o Docker do sistema.

## Iniciar no notebook

```sh
python3 scripts/install-tools.py
make setup
make install
make dev-db
make dev-api                    # terminal 1
make dev-web                    # terminal 2
```

Painel: http://127.0.0.1:5173. Após subir a stack, execute `make bootstrap` para criar o primeiro administrador com senha oculta. Sem Go no host, use `docker compose --env-file .env -f deploy/compose.dev.yml exec api /admin bootstrap`. Não há conta/senha padrão. API: http://127.0.0.1:8080. `make setup` cria `.env` com permissão 0600 e senhas aleatórias, sem sobrescrever arquivo existente. O template `.env.example` não tem senha funcional. Não use esses segredos fora do desenvolvimento.

No VS Code, `.vscode/extensions.json` recomenda Go/ESLint; tasks executam os comandos Make e a configuração de debug lê `.env`. Se Go não estiver no PATH do editor, configure a extensão para o Go em `.tools/go` ou abra o VS Code com esse diretório no PATH. O debugger Delve é instalado pela extensão quando solicitado.

O comando da API usa `go run`: a compilação acontece no primeiro início. Para depurar ou testar SIGINT diretamente no binário, use `go build -o /tmp/flagpilot-api ./cmd/server` dentro de `apps/api`, depois `python3 scripts/run-local.py /tmp/flagpilot-api` na raiz.

Alternativa completa em containers, sem Go/Node no host:

```sh
make setup
make dev-containers
make down
```

O modo completo constrói API runtime e web de desenvolvimento. Ainda não usa bind mounts/HMR de arquivos do host: após alterar código, refaça `make dev-containers`. No modo host, Vite tem HMR; Go requer reinício manual. Essa escolha evita volumes de node_modules e problemas de UID no primeiro incremento.

Todos os ports publicados são loopback. PostgreSQL usa um volume próprio `flagpilot-dev_pgdata`. `make down` preserva esse volume. Não compartilhe volume nem credenciais entre ambientes. Não inicie modos host e containers nas mesmas portas ao mesmo tempo.

Para mudar portas: edite `DB_PORT`, `API_PORT`, `WEB_PORT`, `HTTP_ADDR` e a porta host de `DATABASE_URL` em `.env`; ao mudar API_PORT em modo host, defina também `API_PROXY_TARGET=http://127.0.0.1:<porta>` para Vite. Nos containers os endereços internos ficam inalterados. As senhas locais geradas são hexadecimais; valores personalizados inseridos em URLs precisam de percent-encoding. O parser local de `.env` aceita `CHAVE=valor` sem aspas ou expansão shell.

## Health e contrato

```sh
curl -i http://127.0.0.1:8080/api/v1/health/live
curl -i http://127.0.0.1:8080/api/v1/health/ready
```

Live verifica o processo. Ready verifica conectividade, schema versão 5 e estado não-dirty; banco vazio, indisponível ou schema incompatível retorna 503 sem detalhes internos. A API continua viva enquanto o banco está indisponível. Nunca executa migrations sozinha.

[OpenAPI 3.1](api/openapi.yaml) descreve o contrato inicial completo. `x-implementation: available` identifica health, autenticação e organizações/membros; os demais endpoints retornam 404 até seus incrementos. Métricas ainda não implementadas. Nginx de runtime bloqueia readiness e `/metrics` na borda; o painel consulta a sessão em `/me`, sem expor a saúde interna do banco.

## Verificar

```sh
make generate                   # sqlc e tipos TypeScript OpenAPI
make lint                       # gofmt/vet, ESLint, Spectral e tipos
make test                       # testes unitários Go com race e Vitest
make test-integration           # PostgreSQL descartável via Testcontainers
make build                      # API, web, SDK ESM/CJS e import/require
python3 scripts/smoke.py         # stack isolada, SIGINT e falha real do banco
python3 scripts/check-admin-terminal.py # senha oculta/SIGINT, sem DB
PLAYWRIGHT_BROWSERS_PATH="$PWD/.tools/playwright" npx playwright install chromium
python3 scripts/auth-e2e.py      # login real em stack/conta descartáveis
```

`make ci` agrega lint/testes/integração/build. Docker é obrigatório para integração: indisponibilidade falha o teste, sem skip silencioso. Smoke usa project name e volume próprios e remove somente esses recursos ao terminar; não remove dados de desenvolvimento. A CI também verifica código gerado, scans de segredos/vulnerabilidades e build das imagens. Não publica imagens nem executa deploy.

Para scan local equivalente:

```sh
.tools/bin/gitleaks git --redact --no-banner .
.tools/bin/gitleaks git --pre-commit --staged --redact --no-banner .
npm audit --audit-level=high
# Com Go no PATH:
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
# Executar govulncheck ./... dentro de apps/api.
```

Testes de integração usam somente credenciais sintéticas em banco efêmero. Não há contas administrativas ou senhas padrão de login. O usuário runtime `flagpilot_app` tem DML no schema inicial, sem DDL; o papel de migração é separado. O PostgreSQL oficial usa o usuário de inicialização com superuser no Compose **local**; produção exigirá provisionamento de papel migrador sem superuser conforme ADR-006.

## Documentação e sequência

- [Especificação principal](docs/FlagPilot_Plano_Engenharia_Software.md)
- [Plano de implementação aprovado](docs/IMPLEMENTATION_PLAN.md)
- [Decisões arquiteturais](docs/adr/README.md)
- [Modelo de ameaças inicial](docs/threat-model/initial.md)
- [Migrations e recuperação de falha](docs/runbooks/migrations.md)
- [Operação de desenvolvimento](deploy/README.md)
- [Evidências da fundação](docs/FOUNDATION_STATUS.md)
- [Estado da autenticação](docs/AUTH_STATUS.md)
- [Estado de organizações/membros](docs/ORGANIZATIONS_STATUS.md)
- [Estado de projetos/ambientes](docs/PROJECTS_STATUS.md)
- [Estado de flags booleanas](docs/FLAGS_STATUS.md)
- [Estado de tokens/avaliação](docs/EVALUATION_STATUS.md)
- [Uso de tokens/avaliação](docs/runbooks/evaluation.md)
- [Uso de flags](docs/runbooks/flags.md)
- [Uso de projetos/ambientes](docs/runbooks/projects.md)
- [Cadastrar usuários e gerenciar membros](docs/runbooks/organizations.md)
- [Bootstrap e recuperação](docs/runbooks/identity.md)

Próximo incremento: rollout percentual (4B), seguido do SDK (5). Avaliação HTTP de servidor disponível com tokens por ambiente; o SDK ainda não está funcional. Flags possuem revisão otimista e confirmação de produção. Projetos/ambientes oferecem criação e consulta; edição/exclusão continuam pendentes de política. Organizações/membros já possuem isolamento e proteção do último Owner. Bootstrap, login e recuperação já são auditados. Nenhum DNS, VPS, Coolify ou servidor externo foi alterado. Licença Apache-2.0 e publicação do pacote/remoto continuam pendentes; não há licença atribuída automaticamente.
