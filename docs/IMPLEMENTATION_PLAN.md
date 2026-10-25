# FlagPilot — Plano de implementação

Data: 24/10/2026. Estado: **plano aprovado em 24/10/2026; fundação e incrementos 2A/2B validados localmente**.

Registro: Franklin autorizou a continuação após apresentação deste plano. Os limites e propostas abaixo descrevem o planejamento original; acompanhar o estado real em [FOUNDATION_STATUS.md](FOUNDATION_STATUS.md). A aprovação permite a fundação 0/1A/1B; não autoriza deploy ou alterações externas.

Especificação principal: [FlagPilot_Plano_Engenharia_Software.md](FlagPilot_Plano_Engenharia_Software.md), versão 1.0, lida integralmente (seções 1–17). Este plano detalha sua execução e explicita pontos que ela deixa em aberto. Não altera seus requisitos silenciosamente. O prompt exemplificado na seção 17 não autoriza implementação nesta etapa: prevalece a solicitação atual de planejar e aguardar aprovação.

## 1. Escopo e premissas

Entregar um monólito modular Go, um painel React e um SDK independente, com PostgreSQL como única dependência de execução obrigatória. Começar por flags booleanas; adicionar rollout percentual em incremento separado. Não adotar microserviços, Redis, Kubernetes, ORM, filas ou infraestrutura de monorepo adicional no MVP.

- Desenvolvimento inicial: Debian Linux, VS Code e Docker Compose v2. Preferir API/Vite no host com banco em container; oferecer execução completa em containers como alternativa reproduzível.
- Destino futuro: VPS Ubuntu, Docker, Coolify e Traefik, com API e frontend separados e acesso ao PostgreSQL somente por rede privada.
- Esta etapa entrega somente este plano e seu versionamento local. Não instala ferramentas, cria contas, configura DNS, modifica servidores, publica pacotes ou faz deploy.
- Núcleo obrigatório: identidade, organizações/membros, RBAC, projetos/ambientes, flags booleanas, tokens restritos, avaliação, auditoria, SDK, painel e operação local reproduzível.
- Revisão otimista entra junto das primeiras escritas, apesar da classificação P1: evita perda de alterações e já aparece como regra essencial.
- Liveness/readiness e logs JSON entram na fundação; métricas Prometheus entram antes da homologação. Não deixar segurança estrutural para a última fase.
- SDK Node usa token secreto. Browser usa BFF da aplicação consumidora no MVP proposto; chaves publicáveis e endpoint público ficam sujeitos a decisão explícita.
- Homologação exige autorização futura de alteração externa. Produção está fora da execução autorizada.

## 2. Estrutura do monorepo

```text
flagpilot/
├── apps/
│   ├── api/
│   │   ├── go.mod / go.sum
│   │   ├── cmd/
│   │   │   ├── server/main.go
│   │   │   └── admin/main.go             # bootstrap local/controlado
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── auth/
│   │   │   ├── organizations/
│   │   │   ├── projects/                # inclui ambientes
│   │   │   ├── flags/
│   │   │   ├── tokens/
│   │   │   ├── evaluation/
│   │   │   ├── audit/
│   │   │   ├── storage/postgres/
│   │   │   │   ├── queries/
│   │   │   │   └── sqlc/               # código gerado
│   │   │   ├── http/                   # router, middleware, erros
│   │   │   └── observability/
│   │   ├── tests/integration/
│   │   ├── sqlc.yaml
│   │   └── Dockerfile
│   └── web/
│       ├── src/{app,features,components,api,lib,test}/
│       ├── public/
│       ├── e2e/
│       ├── package.json
│       ├── vite.config.ts
│       └── Dockerfile
├── packages/sdk-js/
│   ├── src/
│   ├── tests/
│   ├── package.json
│   └── tsconfig.json
├── api/{openapi.yaml,.spectral.yaml}/
├── db/migrations/
├── testdata/bucketing/v1.json
├── deploy/
│   ├── compose.dev.yml
│   ├── compose.prod.yml                # modelo futuro, sem deploy
│   ├── web/                           # servidor estático/proxy
│   └── README.md
├── docs/
│   ├── FlagPilot_Plano_Engenharia_Software.md
│   ├── IMPLEMENTATION_PLAN.md
│   ├── adr/
│   ├── threat-model/
│   └── runbooks/
├── .github/workflows/
├── .env.example
├── .gitignore / .dockerignore
├── package.json / package-lock.json
├── Makefile
├── README.md
└── LICENSE                            # após confirmar Apache-2.0
```

Todos os caminhos acima são propostos; não serão criados como diretórios vazios nesta etapa. `compose.prod.yml` é um arquivo dentro de `deploy/`, corrigindo a notação ambígua da árvore original.

Usar npm workspaces para `apps/web` e `packages/sdk-js`, um lockfile raiz e `npm ci`. Um único módulo Go em `apps/api`, provisoriamente `github.com/franklinlem/flagpilot/apps/api`; não criar `go.work` sem um segundo módulo. Makefile raiz apenas coordena comandos. Não usar Nx/Turborepo inicialmente.

Versionamento: repositório privado pretendido `franklinlem/flagpilot`, commits pequenos por incremento, CI em PRs e tags SemVer para releases. Criação do remoto e push não fazem parte desta entrega. Não aplicar licença antes da confirmação.

Versões: escolher Go estável suportado, Node LTS e versões compatíveis de React/Vite/TypeScript ao executar a fundação; registrar versões exatas em manifests, ferramentas e CI. Propor PostgreSQL 16 no primeiro ciclo, conforme o mínimo da especificação, com mesma major em desenvolvimento/CI/destino e patch suportado fixado. Não usar imagens `latest`; registrar digest nas imagens de release. Não presumir ferramentas locais instaladas: Go não foi encontrado no PATH nesta inspeção.

## 3. Organização do backend Go

Um processo HTTP e módulos internos por responsabilidade. `cmd/server` compõe configuração, pool PostgreSQL, serviços e router. Sem regras de negócio em `main` ou handlers.

| Módulo | Responsabilidade |
|---|---|
| `config` | Ler ambiente, validar obrigatórios, limites e combinações inseguras; falhar na inicialização |
| `auth` | Argon2id, login/logout, sessões opacas, CSRF e política de autorização |
| `organizations` | Organizações, membros, papéis e invariantes do último Owner |
| `projects` | Projetos e ambientes, escopo e slugs |
| `flags` | Chave imutável, metadados, arquivamento e configurações transacionais |
| `tokens` | Emissão única, hash, prefixo, expiração, escopo e revogação |
| `evaluation` | Serviço de leitura e avaliador booleano/percentual puro |
| `audit` | Eventos redigidos, consultas e gravação na transação da alteração |
| `storage/postgres` | Pool pgx, SQL parametrizado/sqlc, transações e adaptação de erros SQL |
| `http` | Chi, limites, middleware, Problem Details, paginação e adaptação HTTP |
| `observability` | slog JSON, request ID, métricas e health checks |

Dentro de cada módulo, separar tipos/validação, serviço e transporte conforme houver código. Definir interfaces pequenas no consumidor quando necessárias para isolamento/testes; evitar uma interface genérica de repositório para cada tabela. Serviços não dependem de Chi. Avaliador não depende de HTTP, banco ou logger. Não criar pastas genéricas `utils`/`common` como depósito de regras.

Dependências: handlers → serviços → persistência; serviços usam avaliador e autorização explícitos. A composição injeta dependências, evitando imports circulares. SQL fica centralizado e revisável; código sqlc gerado não é editado manualmente. Transação de escrita é responsabilidade do caso de uso e inclui auditoria.

Medidas transversais desde os primeiros endpoints:

- `context.Context`, timeouts HTTP/SQL, limites de body/batch/contexto, validação de JSON e campos desconhecidos; consultas parametrizadas.
- Request ID, recover, logs JSON redigidos; nunca registrar senha, cookie, Bearer, corpo de login ou `bucketingKey`. Não usar identificadores de usuários/flags como labels Prometheus.
- Autorização por consulta e mutação; um ID do cliente nunca constitui prova de escopo. Negar por padrão. Usar 404 para recursos externos ao escopo e 403 para operação proibida dentro dele.
- Sessão opaca aleatória armazenada como hash; cookie HttpOnly, SameSite e Secure em HTTPS. Exceção HTTP local explicitamente limitada ao modo de desenvolvimento. Não aceitar essa exceção em modo de produção.
- CSRF em mutações, validação de origem, rotação de sessão no login e invalidação no logout/desativação. Rate limit de login independente de token SDK e proteção também contra login CSRF.
- Segredos de sessão/CSRF somente via ambiente quando necessários ao desenho escolhido; não introduzir assinatura redundante apenas porque há nomes de secrets na especificação.
- Liveness sem banco, readiness com consulta curta ao banco e compatibilidade do schema; graceful shutdown em SIGTERM/SIGINT com encerramento de requests/pool e prazo máximo.
- Rate limit em memória é aceitável com uma réplica no MVP; documentar que múltiplas réplicas exigem coordenação. Não confiar em `X-Forwarded-For` de origem arbitrária.

## 4. Estrutura do frontend React

```text
src/
  app/                 # providers, router, layout e tratamento global de erros
  features/
    auth/
    organizations/
    projects/
    environments/
    flags/
    tokens/
    audit/
  components/ui/       # componentes acessíveis e reutilizáveis
  api/                 # tipos gerados, transporte, Problem Details e CSRF
  lib/                 # pequenas funções de infraestrutura sem negócio
  test/                # setup e fixtures de testes
```

React + TypeScript em modo estrito, Vite, React Router, TanStack Query e React Hook Form/Zod. Escolher uma única biblioteca pequena de primitivas acessíveis ao implementar os primeiros formulários; não adotar um design system inteiro antes de avaliar necessidade. Sem Redux inicialmente.

- Estado remoto no TanStack Query, separado de estado local dos formulários. Query keys incluem organização/projeto/ambiente e caches são limpos no logout/troca de identidade.
- Tipos de transporte gerados de OpenAPI 3.1 com ferramenta compatível; validação Go permanece obrigatória. Validar respostas externas sensíveis também no SDK.
- Desenvolvimento: Vite encaminha `/api` para Go. Destino: servidor estático encaminha `/api` para a API na mesma origem. Evita CORS para o painel; não substitui CSRF.
- Nenhum segredo em `VITE_*`, bundle, localStorage ou logs. Cookie de sessão não é acessado por JavaScript. Valor de token recém-criado apenas em memória e mostrado uma vez.
- Rotas refletem organização, projeto e ambiente. Badge persistente com texto; confirmação com resumo de impacto ao alterar ambiente de produção. Não aplicar atualização otimista a uma mudança de flag antes da confirmação do servidor.
- Em conflito de revisão, mostrar configuração atual e solicitar nova ação do operador, sem sobrescrever automaticamente.
- Loading, erro, vazio, expiração de sessão, formulários acessíveis, foco/teclado e layouts responsivos entram junto da tela correspondente.

## 5. Estrutura do SDK TypeScript/JavaScript

Pacote proposto `@flagpilot/sdk-js`, escrito em TypeScript e consumível por JavaScript, com API `isEnabled`, `evaluate` e `close`. Disponibilidade do nome e publicação serão verificadas somente antes de publicar.

```text
src/
  index.ts             # exports públicos
  client.ts            # API e ciclo de vida
  transport.ts         # fetch, timeout, abort e retries
  cache.ts             # TTL, limite e isolamento
  types.ts             # opções, resultado e erros
  validation.ts        # validação das respostas
  internal/bucketing.ts # somente se necessário aos vetores de compatibilidade
```

- Avaliação remota é a fonte de verdade no MVP; não baixar todas as configurações nem executar avaliação offline no browser. Código JS de bucketing para conformidade pode permanecer apenas nos testes, sem criar promessa de avaliação local.
- Node: `Authorization: Bearer` somente em backend confiável. Browser: BFF da aplicação consumidora mantém o token secreto e atende apenas operações/flags autorizadas. Não enviar o token do ambiente ao browser; proxy genérico aberto também seria inseguro.
- Adapter browser/BFF precisa de contrato e exemplo próprios. A construção que recebe `token` é exclusiva do uso servidor; separar opções para não sugerir que token no browser é aceitável.
- Usar `fetch`/`AbortController` nativos para evitar dependência de cliente HTTP; fixar matriz de runtimes na fundação. Gerar ESM, CJS e `.d.ts` com build simples e testar import/require do pacote produzido.
- Cache em memória por instância, com capacidade limitada e chave contendo base URL normalizada, identidade do ambiente/credencial, flag e bucketingKey. Não compartilhar entre usuários ou credenciais. Não usar hash de token como autenticação.
- TTL inicial proposto 30 s, timeout 1.500 ms, no máximo um retry transitório com jitter dentro do orçamento total; nunca retry de 401/403/422. Tratar 429 respeitando `Retry-After` somente se couber nesse orçamento.
- Fallback padrão false, configurável por chamada. Falha de transporte não se confunde com retorno válido false: `evaluate` informa motivo local tipado, sem inventar revisão remota. O contrato local pode ter `revision: null` e razões próprias, a confirmar no OpenAPI/SDK.
- Não cachear falhas de autenticação/transporte como resultado válido. Ao expirar TTL, falha retorna fallback por padrão; uso de último valor conhecido habilitado exige opção explícita e documentação do risco.
- Mudança remota/revogação pode levar até o TTL para refletir em cache; cache não equivale a revogação imediata. `close` aborta operações, limpa caches e timers; chamadas posteriores têm comportamento documentado.
- Deduplicar requests concorrentes equivalentes, respeitar abort externo, limitar batch e não persistir contexto. Testar isolamento, relógio controlado, expiração, resposta inválida, 404/401, timeout/offline, retry e fechamento.

## 6. Modelo inicial de dados

UUIDs gerados pela aplicação, timestamps `timestamptz`, restrições nomeadas e SQL explícito. Não exigir extensões para a primeira migração. Toda tabela recebe os campos essenciais da seção 6 da especificação; abaixo estão complementos e invariantes propostos.

| Tabela | Modelo e integridade inicial |
|---|---|
| `organizations` | UUID, nome, slug único, created_at |
| `users` | UUID, email, password_hash Argon2id, disabled_at, created_at; índice único em lower(email) |
| `memberships` | PK (organization_id,user_id), role CHECK Owner/Admin/Editor/Viewer; FKs e transação impedindo remover último Owner |
| `projects` | UUID, organization_id, nome, slug; UNIQUE(organization_id,slug) |
| `environments` | UUID, project_id, nome, slug, revision bigint default 0 CHECK >=0; UNIQUE(project_id,slug) e UNIQUE(id,project_id) |
| `flags` | UUID, project_id, key, descrição, archived_at, created_at; chave validada/imutável; UNIQUE(project_id,key) e UNIQUE(id,project_id) |
| `flag_configs` | UUID, project_id, flag_id, environment_id, enabled default false, rollout_bps 0..10000 default 10000, salt, updated_at, version >=1; UNIQUE(flag_id,environment_id) |
| `api_tokens` | UUID, project_id, environment_id, nome, token_prefix, token_hash único, scope CHECK evaluation, expires_at, revoked_at, created_at e created_by nullable |
| `audit_events` | UUID, organization_id, actor_id nullable, action, resource_type/id, before/after JSONB redigidos, created_at; snapshot mínimo do ator para histórico após desativação |
| `sessions` | UUID, user_id, token_hash único, expires_at, revoked_at, created_at; vínculo com usuário ativo em cada uso |

Consistência de projeto no banco: `flag_configs(flag_id,project_id)` referencia `flags(id,project_id)` e `flag_configs(environment_id,project_id)` referencia `environments(id,project_id)`. Token usa a mesma FK composta de ambiente/projeto. Isso impede associar flag ou token a ambiente de outro projeto mesmo se houver erro na aplicação. FKs compostas são suportadas pelo PostgreSQL [S2].

Índices adicionais: memberships(user_id), projects(organization_id), environments(project_id), flag_configs(environment_id,flag_id), tokens(prefix), sessões por usuário/expiração e audit_events(organization_id,created_at DESC,id DESC). Reaproveitar índices de PK/UNIQUE; não duplicar flags(project_id,key). Avaliar índices extras com consultas reais.

Políticas iniciais: arquivar flags; não reutilizar chave arquivada no mesmo projeto; não oferecer hard delete de organização/projeto/ambiente no MVP sem definir retenção. FKs RESTRICT por padrão para dados de domínio e auditoria; sessões podem ser removidas por limpeza controlada. Auditoria sem UPDATE/DELETE para o usuário runtime, exceto papel específico de retenção futura. Não alegar imutabilidade contra administrador do banco.

Tokens: segredo aleatório de alta entropia, comparação segura e hash SHA-256 de token completo; Argon2id reservado a senhas humanas. Prefixo serve para identificação, não como segredo. Expiração padrão proposta 90 dias, configurável; confirmação pendente. Nunca salvar token em claro em auditoria ou banco.

### Revisões e escrita atômica

1. Autorizar ator e abrir transação.
2. Bloquear a linha do ambiente e comparar revisão recebida, ou realizar UPDATE condicional equivalente.
3. Validar e alterar configuração, incrementando `flag_configs.version`.
4. Incrementar `environments.revision` uma vez por mutação lógica no ambiente.
5. Inserir auditoria na mesma transação; falha em qualquer passo faz rollback integral.
6. Responder com revisão e ETag coerentes com a representação retornada.

Criação/arquivamento de flag afeta todos os ambientes do projeto: atualizar snapshots/revisões correspondentes na mesma transação, bloqueando ambientes em ordem estável. Criar ambientes gera configurações desativadas para flags ativas existentes; criar flag gera configurações desativadas nos ambientes existentes. Testar concorrência entre ambas as criações; serializar essas operações no projeto para impedir lacunas. Propor bloqueio projeto → ambientes para operações que alteram o conjunto; escrita de configuração bloqueia somente seu ambiente.

`version` é a versão da configuração individual; `revision` é a versão do snapshot completo do ambiente. Histórico de auditoria não implica rollback de configuração nem snapshots históricos reconstituíveis; essa capacidade precisa de escopo separado.

## 7. Docker Compose de desenvolvimento

O arquivo futuro será `deploy/compose.dev.yml`; nesta etapa descrevemos seu contrato sem criar serviços.

| Serviço | Execução e dependências | Exposição local |
|---|---|---|
| `db` | PostgreSQL, volume nomeado, healthcheck `pg_isready` | 127.0.0.1:5432 para modo host; porta configurável |
| `migrate` | Job golang-migrate com SQL montado somente leitura; aguarda db saudável | Nenhuma; execução explícita e encerramento com status |
| `api` | Profile `full`; build Go e execução; exige job migrate concluído | 127.0.0.1:8080, opcional para debug |
| `web` | Profile `full`; Vite/HMR, proxy `/api` para api; dependências/cache próprios | 127.0.0.1:5173 |

Modo leve padrão: subir db, executar job de migração e rodar Go/Vite no host. Modo completo: subir profile `full`, com banco saudável → migração concluída → API → web. `depends_on` simples não garante prontidão: usar `service_healthy` e `service_completed_successfully` conforme [S1]. A API também trata indisponibilidade após o startup; dependência Compose não substitui reconexão/readiness.

Vite escuta `0.0.0.0` apenas dentro do container; porta publicada fica restrita a loopback. Usar URLs distintas para DB no host (`localhost`) e no container (`db`), e para proxy Vite (`localhost:8080`/`api:8080`). Scripts selecionam configuração sem copiar credenciais para manifests. Bind mounts com UID/GID compatíveis evitam arquivos root no notebook. Definir alvos distintos de build de desenvolvimento e runtime; sem hot reload obrigatório de Go, reinício manual basta inicialmente.

`.env.example` raiz terá placeholders/documentação para: `APP_ENV`, `DATABASE_URL`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `HTTP_ADDR`, `PUBLIC_BASE_URL`, `ALLOWED_ORIGINS`, parâmetros de sessão/CSRF, log, timeouts, rate limit e portas locais. Credenciais administrativas e segredos não terão valor funcional padrão. Validar variáveis antes do startup. `.env`, dumps, logs e artefatos sensíveis no `.gitignore` e `.dockerignore`; distinguir interpolação `--env-file` do envio efetivo de variáveis ao container.

Comandos planejados (somente disponíveis após fundação):

```sh
cp .env.example .env                    # preencher valores locais
make dev-db                            # db saudável + migration explícita
make dev-api                           # outro terminal
make dev-web                           # outro terminal
make dev-containers                    # alternativa completa com profile full
make test
make down                              # preservar volume
```

Remover volumes exigirá comando separado e deliberado; `down` não usa `-v`. Prometheus/Grafana, observadores externos e Traefik não são necessários no notebook. Métricas continuam disponíveis internamente sem stack de monitoramento. Containers de release multi-stage, não-root, sem toolchain e com shutdown correto; limites e modo read-only serão validados antes da homologação.

## 8. Estratégia de migrations

Usar golang-migrate com versões sequenciais e SQL `NNNNNN_nome.up.sql` / `.down.sql`. Nada de auto-migrate na inicialização normal da API ou script de schema paralelo que divirja das migrations.

Sequência proposta, adicionada junto de cada incremento:

1. `000001_identity`: users, organizations, memberships, sessions e restrições.
2. `000002_audit`: eventos e permissões de auditoria, antes das primeiras mutações administrativas de 2B.
3. `000003_projects_environments`: projetos, ambientes e revisões.
4. `000004_flags`: flags, configurações e FKs compostas; nenhum endpoint de escrita de flag antes da auditoria existir.
5. `000005_evaluation_tokens`: credenciais de ambiente e índices.
6. Migrations posteriores apenas quando necessário, incluindo alterações aprovadas do modelo.

A primeira fundação pode validar o mecanismo usando `000001` sem implementar login. Usuário bootstrap e ambientes são criados por comando/caso de uso, não inseridos com senha padrão na migration. Rollout pode ter campos preparados desde `000004`, mantendo valores 0/100% até sua fase.

Cada migration tem transação explícita quando suportada; índices concorrentes exigem tratamento próprio. Não editar migration já aplicada/compartilhada. Job único/controlado com credencial DDL separada do usuário runtime, timeout de lock e proteção contra execuções concorrentes. Não executar migration automaticamente em cada réplica.

Testar banco vazio → head, esquema anterior → head, restrições, falha e estado dirty. Down/up apenas em banco descartável para alterações reversíveis; `.down.sql` com perda de dados precisa de advertência e nunca deve ser rotina de rollback operacional. Não usar `force` para esconder falha. Preferir expand/contract e roll-forward; backup/restore e rollback de imagem são procedimentos distintos.

## 9. Contrato HTTP e avaliação

Completar `api/openapi.yaml` para todos os endpoints da tabela original e também as lacunas aprovadas de membros, bootstrap, listagem de tokens e arquivamento. Base canônica `/api/v1`; exemplos e documentação do SDK usarão essa base. Health/metrics terão caminho único explicitado no contrato e no proxy.

OpenAPI 3.1, Spectral, limites, autenticação cookie/CSRF e Bearer, paginação cursor estável com desempate por ID, exemplos e Problem Details RFC 9457. Validar ferramenta de geração para 3.1 antes de fixá-la. Não interpretar tipos gerados como validação automática em runtime.

Proposta de concorrência: header `If-Match` obrigatório em configuração; ausência 428, ETag desatualizado 412, conflito de domínio 409. HTTP define 412 para precondição falsa de `If-Match` [S3]; a especificação pede 409 para concorrência, portanto esta alteração precisa ser aprovada e refletida na fonte. ETag pertence à representação identificada, com revisão ambiental coerente; decidir endpoint de leitura/configuração e sintaxe do ETag antes dos handlers. Para metadados de flag, usar controle de versão próprio ou equivalente, sem fingir que um único ambiente cobre todo o projeto.

Proposta de bucketing a registrar na ADR-003:

- Interpretar oito bytes iniciais do SHA-256 como `uint64` big-endian; `bucket = n % 10000`; resultado parcial é `bucket < rollout_bps`. JS usa `BigInt`, sem conversão para Number antes do módulo.
- Entrada UTF-8 na ordem exata definida na fonte. IDs UUID em forma canônica, salt fixo aleatório codificado em hex e chave validada; somente bucketingKey pode conter `:`. Não aplicar normalização Unicode implícita.
- Ordem das regras: flag inexistente/arquivada → NOT_FOUND e false proposto; disabled → false; rollout 0 → false; rollout 10000 → true; parcial sem contexto → fallback; parcial com contexto → comparação.
- Salt fica estável entre mudanças percentuais; não recalcular a cada edição. Mesmo usuário pode ter bucket distinto entre ambientes, conforme IDs no hash.
- Compartilhar vetores Go/JS com Unicode, delimitadores, extremos 0/10000, buckets de borda e valor acima de 2^53. Testar monotonicidade do conjunto liberado ao ampliar percentual com salt fixo.
- Decidir resultado HTTP/SDK para flag inexistente, motivo de rollout 0 e propriedade de fallback antes de implementar. Proposta: defaultValue pertence à chamada SDK, e servidor sem contexto retorna false/MISSING_CONTEXT; SDK aplica default configurado nessa razão. Não confundir DISABLED válido com erro/fallback.

## 10. Estratégia de testes

Testes acompanham cada incremento e seus riscos; não criar suites vazias ou testes que apenas repetem getters. Meta ≥80% separadamente para lógica de avaliação e autorização, com cenários obrigatórios independentemente da cobertura.

| Camada | Ferramenta e cenários essenciais |
|---|---|
| Go unitário | testing/httptest, race detector; validação, RBAC, evaluator, revisão e middleware |
| PostgreSQL integração | Testcontainers-go com banco descartável; migrations, FKs de projeto, rollback com auditoria, concorrência, isolamento de tenants e revogação |
| Contrato | Spectral + requests positivos/negativos contra API real; status, erros, ETag, schemas, cookie/CSRF e SDK compatível |
| SDK | Vitest com fetch/relógio controlados; falhas, TTL, isolamento, budget de retry, cancelamento e artefatos ESM/CJS |
| Frontend | Vitest + Testing Library/axe; formularios, permissões visuais, estados e conflito; backend ainda valida tudo |
| E2E | Playwright, API e PostgreSQL reais: login → projeto → ambiente → flag → ativação → avaliação → auditoria → revogação → logout |
| Segurança | Secret scan, govulncheck/SCA, SAST e scan de imagens; CSRF, IDOR, limites de payload/rate, XSS e logs sem contexto/segredos |
| Carga/resiliência | k6 10/50/100 req/s, API/banco indisponíveis, reinício e cache; restore em banco isolado antes de produção |

Não reutilizar banco de desenvolvimento nos testes destrutivos. Testcontainers exige Docker acessível; CI deve falhar explicitamente se integração obrigatória não puder executar. Permitir DATABASE_URL de teste explicitamente separada para diagnóstico, mantendo o mesmo conjunto de verificações.

Chromium no fluxo rápido de PR; Chromium/Firefox e viewports mobile/tablet/desktop na validação de release. Headless não substitui aceitação visual/manual com teclado. Matriz SDK em Node LTS suportados e browser via BFF de demonstração. Benchmark registra CPU/RAM, versão, dataset, concorrência, aquecimento e erro; p95 <100 ms é meta não verificada até medir em ambiente de referência. Não rodar carga em produção.

## 11. Pipeline inicial de CI

GitHub Actions em PR e push para main, sem credenciais de deploy e sem auto-homologação nesta etapa. Ativar jobs conforme surgirem artefatos: não manter checks verdes artificiais para componentes ausentes.

1. **Higiene:** scan de segredos, lockfiles e formatação; permissões mínimas `contents: read`, actions fixadas por SHA e cancelamento de runs antigas por branch.
2. **Go:** gofmt, go vet, unitários/race e build; lint focado, sem conjunto excessivo de regras. govulncheck para dependências alcançáveis.
3. **Banco:** migrations em PostgreSQL real e testes integração, com isolamento e limpeza; mesma major PostgreSQL do desenvolvimento.
4. **TypeScript:** npm ci, lint, tsc, testes web/SDK e build; verificar import/require/tipos do SDK empacotado, sem publicar.
5. **Contrato:** Spectral, regenerar tipos/sqlc e falhar quando houver diferença não versionada; testes de contrato quando API disponível.
6. **Containers/E2E:** build das imagens sem push e smoke health/readiness desde a fundação; E2E completo passa a bloquear PR quando o fluxo existir.

Cachear módulos Go e cache npm com chaves por lockfile, nunca `.env` ou segredos. Upload somente de relatórios sanitizados, com retenção curta; traces/screenshots E2E usam credenciais sintéticas. Não usar `pull_request_target` para executar código de PR com secrets.

Fase de release futura: SBOM, scan de imagens e publicação GHCR somente em contexto confiável autorizado, tags/digests imutáveis e changelog. Gate de release bloqueia falhas críticas/altas sem decisão formal; banco de vulnerabilidades indisponível não equivale a scan aprovado. Compatibilizar a menção a apenas críticas na seção 9 com o gate mais restritivo da seção 10.

CD fica desabilitado: a previsão de homologação automática na seção 13 exige autorização futura, não supera a regra atual de não alterar servidores externos. Produção não terá deploy executado durante este trabalho.

## 12. Sequência incremental do MVP

Cada incremento termina com documentação, testes correspondentes e resultado verificável. Só avançar após aprovação deste plano; depois, manter incrementos pequenos dentro do escopo autorizado.

| Incremento | Entrega | Critério de saída |
|---|---|---|
| 0 — Decisões | ADRs iniciais, threat model, fluxos UX, contrato completo e resolução de pendências bloqueantes | Escopo, RBAC, BFF, bootstrap e semântica de avaliação aprovados |
| 1A — Fundação | Git/ignore, módulo Go, Chi, config, slog, shutdown, esqueleto web/SDK, Compose DB e Makefile | API live/ready, Vite e comando local documentados; sem negócio |
| 1B — Persistência/CI | pgx/sqlc, mecanismo migrate, schema identidade, usuário runtime restrito e CI inicial | Banco vazio migrado, ready falha sem DB, build/testes reais verdes |
| 2A — Identidade | Bootstrap CLI, Argon2id, login/logout/me, sessão e CSRF | Login limitado, expiração/logout/desativação invalidam acesso |
| 2B — Organizações/RBAC | Organizações, membros e matriz de papéis | Usuário A não acessa B; último Owner protegido; mutações auditadas |
| 3A — Projetos/ambientes | CRUD aprovado, slugs e associação por organização | Integridade/isolamento e permissões testados |
| 3B — Flags booleanas | Chaves imutáveis, configurações por ambiente, arquivamento, auditoria e revisão atômicos | Alterar staging não muda produção; corrida não perde escrita |
| 4A — Tokens/avaliação | Tokens, revogação/expiração, single/batch, limites e evaluator booleano | Token nunca administra; avaliação consistente e sem vazamento |
| 4B — Rollout P1 | Bucketing v1, salt estável e vetores Go/JS | Vetores idênticos; casos de borda e concorrência aprovados |
| 5 — SDK | Node, adapter BFF aprovado, cache limitado, timeout/retry/fallback, builds | Contrato, cache isolado, offline e compatibilidade do pacote aprovados |
| 6 — Painel | Login, organização/projeto/ambiente, flags, tokens, membros e auditoria | E2E principal, a11y e confirmação de produção; erros/conflitos úteis |
| 7 — Preparação operacional | Métricas, imagens, modelo Coolify, runbooks, benchmark, backup/restore local | Operação local reproduzível e artefatos revisáveis; sem alterar VPS |
| 8 — Homologação autorizada | Inventário VPS, stack isolada, TLS, smoke/E2E, restore e rollback | Evidência em homologação e checklist; depende de autorização futura |

Auditoria básica entra antes do primeiro incremento com mutações administrativas (2B), usando `000002_audit`; telas de consulta e cobertura completa do fluxo de flags chegam nos incrementos posteriores. Não publicar mutações administrativas sem registro; bootstrap CLI também gera evento quando criar a organização e seu primeiro Owner. UI mínima de cada fluxo pode acompanhar o backend; incremento 6 consolida o painel e os E2E, sem exigir um grande lote final.

MVP booleano pode ser validado antes de 4B. Rollout P1 entra antes de 1.0 se aprovado; não incluir variantes, segmentos complexos, cobrança, SSO/MFA ou SDK Go. A estimativa original de 34–51 dias úteis inclui produção e deve ser revista após 1B; não é prazo assumido por este plano.

## 13. Inconsistências e decisões pendentes

As propostas abaixo ainda não são mudanças aprovadas da especificação.

| ID | Inconsistência/lacuna | Proposta e momento de decisão |
|---|---|---|
| D01 | Papéis falam em projeto/flags autorizadas, mas memberships só tem escopo organizacional | MVP com papéis organizacionais herdados em todos os projetos: Owner tudo; Admin projetos/ambientes/flags/tokens; Editor flags/configurações; Viewer leitura sem valores secretos. Somente Owner gerencia membros. Se precisar restringir por projeto/flag, incluir grants próprios antes de implementar RBAC. **Bloqueia 2B.** |
| D02 | Login existe, mas criação inicial de usuário, membros e recuperação não têm contrato | CLI admin local lê senha de entrada segura, cria usuário/org/Owner de forma explícita; sem signup público nem senha padrão. Owner associa usuários existentes; convite por email fica fora do MVP. Recuperação por CLI controlada revoga sessões. Adicionar endpoints de membros e invariantes do último Owner. **Bloqueia identidade/contrato.** |
| D03 | SDK browser permitido sem definir chave pública/BFF | BFF como caminho inicial; endpoint publicável fica fora do MVP. Documentar integração e contrato do adapter browser. **Bloqueia contrato SDK, não a fundação.** |
| D04 | Fórmula SHA-256 não define como mapear uint64 em 10.000 buckets | Propor módulo 10000 com comparação estrita e BigInt em JS; fixar vetores/encoding/salt. **Bloqueia rollout.** |
| D05 | If-Match versus 409 em concorrência; não há GET individual da configuração explícito | Propor 428/412/409 e leitura com ETag correspondente; completar contrato antes da escrita. **Bloqueia 3B.** |
| D06 | “fallback configurado” no servidor não aparece no schema/request; exemplo só tem default no SDK | Default por chamada SDK; servidor false/MISSING_CONTEXT, com aplicação explícita do default no SDK. Definir também representação de fallback sem revisão e flag inexistente. **Bloqueia avaliação/SDK.** |
| D07 | História P1 mistura revisão, auditoria e histórico restaurável | Revisão otimista e auditoria são obrigatórias desde as escritas; restauração/snapshots completos não estão implicitamente incluídos. **Aprovar limite antes de 3B.** |
| D08 | Rollout é caso de uso principal, P1 e opcional antes de 1.0 | Booleano primeiro, incremento 4B separado; confirmar se 4B integra a primeira release. **Não bloqueia fundação.** |
| D09 | Esboço SQL permite cruzar projetos e omite tabelas | FKs compostas, migrations completas por incremento e testes de integridade; antecipar auditoria às primeiras mutações. **Antes da primeira migration correspondente.** |
| D10 | CRUD promete operações ausentes na tabela HTTP | Definir edição/arquivamento, listagem de tokens, membros, rotação/recuperação e política de exclusão; não inventar hard delete. **Antes do contrato de cada módulo.** |
| D11 | Arquitetura mostra `/v1/evaluate`, contrato define `/api/v1`; health pode parecer fora da base | Padronizar `/api/v1`, inclusive health; caminhos internos e regras de proxy explícitos. `/metrics` pode ficar em listener interno separado documentado. **Na fundação.** |
| D12 | readiness interno também serve a healthchecks/roteamento | Em dev acesso por loopback; destino permite probes pela rede interna e bloqueia readiness/metrics na borda pública; sem detalhes sensíveis no corpo. **Antes da homologação.** |
| D13 | Não definidos retenção, expiração, limites, email e identificação de produção | Tokens implementados em 4A com prazo explícito até 365 dias e sugestão de 90 dias na UI (ADR-011). Sessão absoluta 12 h; avaliação com body 64 KiB, batch 100 e bucketingKey até 256 bytes. Classificação de produção implementada em 3A/3B, sem renomeação. Auditoria 90 dias ainda proposta, sem limpeza automática antes de aprovação; revisar demais limites nas etapas correspondentes. |
| D14 | Cache de 30 s versus suspensão imediata e revogação | Documentar atraso máximo por TTL, permitir TTL zero para necessidade de consulta imediata e não prometer revogação de resultado local já cacheado. Sem cache de autenticação no servidor inicialmente. **Antes do SDK.** |
| D15 | “Nenhum identificador pessoal persistido” versus cache/bucketing e logs de acesso | Contexto somente em memória limitada pelo TTL; não registrar em banco/logs/traces. Evitar IDs pessoais na origem e rever proxy. Hash não transforma automaticamente dado pessoal em anônimo. **No threat model.** |
| D16 | Licença, pacote npm e remoto ainda propostos | Confirmar Apache-2.0 antes de LICENSE/publicação; manter repositório privado inicialmente; verificar nome npm antes da publicação. **Não bloqueia desenvolvimento local.** |
| D17 | Homologação automática/roadmap de produção conflita com autorização atual | CI local/remoto sem CD; alteração externa e homologação exigem autorização futura; produção fora do escopo. **Limite de execução obrigatório.** |
| D18 | Performance e capacidade VPS sem ambiente medido | Definir fixture e máquina de referência; não prometer SLO nem RPO/RTO antes de carga/restore. **Antes de aceitação operacional.** |

Pendências imediatas para aprovar o desenho: modelo de permissões (D01), bootstrap/membros (D02), opção BFF (D03), semântica de contrato/concorrência/fallback (D05–D07) e limite da primeira release (D08). As demais têm proposta padrão e momento explícito de decisão.

## 14. ADRs e documentação planejadas

| ADR | Conteúdo a registrar |
|---|---|
| ADR-001 | Monólito Go/Chi/pgx/sqlc, módulo único e npm workspaces |
| ADR-002 | Cookie/sessões opacas, CSRF, bootstrap, expiração e recuperação |
| ADR-003 | Bucketing v1, encoding, fórmula, salt e vetores compartilhados |
| ADR-004 | Cache/fallback/retries, revogação e limites da avaliação offline |
| ADR-005 | Tenancy, papéis, FKs compostas, transações e revisões |
| ADR-006 | Containers, futura topologia Coolify, proxy, isolamento e migrations |
| ADR-007 | Token secreto servidor e BFF browser; avaliação pública adiada |

ADRs terão contexto, decisão, alternativas, consequências e status; nenhuma será marcada como aceita antes da aprovação pertinente. README documentará instalação Debian, comandos, versões e fluxo VS Code. Runbooks cobrirão migrations/dirty state, bootstrap/recuperação, rotação, incidentes, backup/restore e rollback. Atualizar OpenAPI e threat model em cada incremento afetado.

## 15. Aceitação desta etapa e próximo passo

Esta etapa termina com análise e plano documentados, sem arquivos executáveis de negócio, containers iniciados ou mudanças externas. O documento original permanece preservado. A validação desta entrega é documental; testes de aplicação ainda não são aplicáveis.

Após aprovação, iniciar somente o incremento acordado, recomendado **0 + 1A + 1B**, com comandos reproduzíveis, validação de health/DB/CI e ADRs iniciais. Não iniciar autenticação, CRUD ou avaliação junto da fundação sem passar pelos respectivos critérios. Aguardar aprovação de Franklin antes de produzir código.

## 16. Referências técnicas consultadas

Consultadas em 24/10/2026; orientam as propostas, sem substituir a especificação do projeto.

- [S1 — Docker Compose: ordem de startup e condições de dependência](https://docs.docker.com/compose/how-tos/startup-order/).
- [S2 — PostgreSQL: constraints e foreign keys](https://www.postgresql.org/docs/current/ddl-constraints.html).
- [S3 — RFC 9110: If-Match e precondições HTTP](https://www.rfc-editor.org/rfc/rfc9110.html#name-if-match).
- [S4 — Go: organização de módulos e internal/cmd](https://go.dev/doc/modules/layout).

## Continuidade após validação de organizações

Franklin aprovou o funcionamento manual de organizações/membros, incluindo cursor e seletor, e autorizou a continuidade. O incremento 3A implementa criação/listagem de projetos e ambientes, detalhe de projeto, presets development/staging/production, autorização e auditoria. Edição/exclusão não estão implicitamente aprovadas: aguardam política de retenção/concorrência. Decisões em ADR-009 e estado em PROJECTS_STATUS.md. Próxima implementação funcional: 3B.

## Continuidade após validação de projetos

Franklin confirmou criação de projetos e ambientes automáticos e autorizou a etapa 3B. Flags booleanas, configurações por ambiente, auditoria/revisão atômicas, chave imutável, arquivamento e confirmação de produção estão implementados conforme ADR-010; resultados em FLAGS_STATUS.md. Filtros de catálogo e consulta de auditoria permanecem para a consolidação do painel. Próximo incremento: 4A, tokens e avaliação.

## Continuidade após validação de flags

Franklin confirmou criação, ativação e arquivamento de flags e autorizou 4A. Tokens por ambiente, emissão única, expiração/revogação auditadas e avaliação booleana single/batch foram implementados conforme ADR-011. Prazo UI inicial de 90 dias, obrigatório e limitado a 365; rate limit local e contexto somente em memória. Evidências em EVALUATION_STATUS.md. Próximas etapas: 4B (rollout) e 5 (SDK).

## Continuidade após validação de tokens

Franklin confirmou manualmente emissão com validades diferentes, cópia, ocultação e revogação no painel e autorizou 4B. Essa aceitação cobre os cenários exercitados de gestão, sem presumir testes manuais da API. Rollout percentual determinístico, edição decimal por ambiente, confirmação de produção e vetores Go/TS compartilhados estão implementados conforme ADR-003. Sem migration, dependências ou variáveis novas; schema 5 preservado. Próxima etapa: 5, cliente SDK HTTP com cache/timeout/fallback. Evidências e limites em EVALUATION_STATUS.md.

## Continuidade após validação de rollout

Em 25/10/2026, Franklin confirmou visualização e salvamento de porcentagens diferentes e autorizou a etapa 5. O SDK privado HTTP para Node 24.x implementa evaluate/isEnabled/close, cache LRU limitado, timeout total, retry transitório máximo 1, deduplicação/cancelamento e fallback por chamada. ESM/CJS/tipos e integração com API real são verificados. Contrato em ADR-004, evidências em SDK_STATUS.md e exemplo no runbook/sdk.md. Sem novas dependências, migrations ou credenciais reais. A próxima etapa é 6, consolidação do dashboard (filtros, consulta de auditoria e acessibilidade); homologação continua sujeita à autorização externa.

## Consolidação do dashboard — 6A

Continuidade autorizada em 25/10/2026 após entrega do SDK. Filtros de chave/estado do catálogo antes da paginação, histórico organizacional somente leitura com projeção de campos conhecidos, filtros de ação/tipo, paginação temporal e controles acessíveis estão implementados conforme ADR-012. Schema 5, sem novas dependências ou alterações externas. Evidências e limites em DASHBOARD_STATUS.md; uso em runbooks/dashboard.md. Próximo incremento: 6B, fechamento da matriz de UX/acessibilidade e critérios operacionais do MVP antes de planejar homologação. Não há aprovação implícita de deploy/servidores externos.

## Fechamento de UX e recuperação — 6B

Franklin confirmou filtros/histórico e autorizou continuidade em 25/10/2026. Navegação por teclado, orientação/foco por tela, retorno ao topo e timeout comum de 8 s com sinais de cancelamento implementados. Matriz de seis fluxos por navegador validada em Chromium/Firefox, sem flexibilizar limites de login; smoke de falhas/reinício aprovado em stack sintética. Evidências em DASHBOARD_STATUS.md e MVP_READINESS.md. Não houve migration, dependência do produto, deploy ou modificação externa. Próxima entrega operacional deve priorizar backup/restore local e observabilidade, mantendo homologação/produção separadas e dependentes de autorização específica.

## Ensaio local de recuperação e observabilidade — 6C

Continuidade autorizada após 6B. Script sintético de backup/restore destrói origem própria, compara todas as tabelas no destino e aplica invalidação auditada de sessões/tokens antes de iniciar API restaurada. Listener Prometheus separado, opt-in, registry próprio e labels agregados; Compose local publica somente em loopback. ADR-013, runbooks/recovery.md e observability.md descrevem uso/limites. Schema 5, sem backup recorrente, alertas externos ou deploy; nenhum dado real de Franklin é usado pelo ensaio. Próxima entrega deve medir capacidade local e preparar observabilidade/coleta e plano operacional antes de qualquer autorização de homologação externa.

## Medição local de capacidade — 6D

Continuidade autorizada após 6C. Cenários k6 10/50/100 e sobrecarga 150 req/s executados em stack sintética com cotas, mantendo limites de segurança. Percentis dos sucessos e de todos os resultados, 429/erros/descartes, amostras de recursos e condições registrados em CAPACITY_STATUS.md e evidence/load-2026-10-25.json. Nenhuma alteração de negócio, schema, dependências da aplicação ou infraestrutura externa. Método em runbooks/capacity.md e ADR-014. Não certifica SLO sustentado, saturação máxima ou sizing da VPS. Próximo incremento proposto: artefatos de release (scans/SBOM/digests) e ensaio local de rollback, mantendo licença/publicação/deploy pendentes.

## Evidências de release e rollback compatível — 6E

Continuidade autorizada após 6D. Bases/imagens upstream fixadas por digest; SBOM CycloneDX e scans dos quatro runtimes com manifest de hashes, gate fail-closed e workflow manual sem push/deploy. Ensaio de reversão da API 6B/candidata, ambas schema 5, preservou DB/tabelas/sessão/revogação/avaliação/escritas e reaplicou candidata sem down migration/restore. Fontes históricas reconstruídas, sem alegar release publicada. Gate de imagens bloqueado por HIGH/CRITICAL; remediação/triagem é próximo incremento antes de release. Nenhuma dependência de negócio, migração, publicação ou servidor externo alterado. Método em runbooks/release.md, decisão ADR-015, evidências/limites em RELEASE_STATUS.md.

## Remediação dos runtimes — 6F

Correções autorizadas após 6E. PostgreSQL 16.15 com helper gosu recompilado/depêndencias corrigidas, NGINX estável 1.30.6 com APKs fixados e migrator scratch PostgreSQL-only sobre golang-migrate 4.20.1. Novos scans/SBOM aprovam gate sem HIGH/CRITICAL; dois UNKNOWN OpenPGP não importados na API continuam registrados, com análise de binários. Upgrade sintético do mesmo volume, recuperação, rollback, integração, builds e Chromium validados. Desenvolvimento local atualizado com cópia privada prévia e conteúdo preservado; schema 5 e dados mantidos. Sem novos recursos de negócio, alteração externa, publicação ou licença automática. Decisão ADR-016; evidências e limites em SECURITY_REMEDIATION_STATUS.md. Próximo incremento: artefatos operacionais/release ainda pendentes, sem inferir autorização externa.
