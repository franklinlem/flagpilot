# FlagPilot — Plano de Engenharia de Software

**Versão:** 1.9 · **Data:** 25/10/2026 · **Estado:** implementação incremental

**Revisão 1.9:** incorpora as direções do [plano aprovado](IMPLEMENTATION_PLAN.md) e das [ADRs](adr/README.md). Fundação, identidade, organizações/membros, criação/consulta de projetos/ambientes e gestão de flags booleanas, tokens por ambiente e avaliação booleana/percentual single/batch estão implementados localmente. Ver [estado de avaliação](EVALUATION_STATUS.md). Rollout determinístico v1 está implementado com vetores Go/JS; o SDK HTTP privado para backend está implementado com cache/timeout/fallback (ver SDK_STATUS.md); `api/openapi.yaml` identifica a disponibilidade de cada endpoint. Filtros de catálogo e consulta de auditoria estão implementados em 6A; navegação/recuperação e testes Chromium/Firefox em 6B (DASHBOARD_STATUS.md). Ensaio sintético de recuperação e métricas Prometheus em listener separado entregues em 6C (OPERATIONS_STATUS.md). Critérios locais/pendentes em MVP_READINESS.md. Licença, publicação e alterações externas continuam pendentes.

## 1. Visão e limites

Plataforma self-hosted de **feature flags** para projetos e ambientes separados, com avaliação determinística, API administrativa, API de avaliação, painel React e SDK JavaScript. Objetivo: liberar ou suspender funcionalidades sem novo deploy, desde que a aplicação cliente consulte o serviço. Licença proposta: Apache-2.0 (confirmar antes de publicar). Público: desenvolvedores, equipes pequenas e operações DevOps.

**MVP:** organizações/projetos/ambientes; flags booleanas; autenticação administrativa; tokens por ambiente; CRUD com auditoria; avaliação determinística; SDK JavaScript; painel; Docker Compose; deploy em homologação. **Fora do MVP:** A/B testing estatístico, multivariantes, segmentação complexa, cobrança, SDK Go, Redis obrigatório, auto-hospedagem multi-região, aprovação de mudanças.

### Critérios de sucesso
- Avaliação correta em 100% dos casos de testes determinísticos.
- p95 da API de avaliação < 100 ms em ambiente de referência definido em benchmark (sem latência de rede externa).
- Nenhum vazamento entre organizações/ambientes nos testes de autorização.
- Recuperação de backup validada antes da produção.
- Implantação e rollback documentados e reproduzíveis.

## 2. Papéis e casos de uso

| Papel | Permissões |
|---|---|
| Owner | Gerenciar organização, membros, projetos e credenciais |
| Admin | Gerenciar projetos, ambientes, flags e tokens em toda a organização |
| Editor | Criar e alterar flags nos projetos da organização |
| Viewer | Consultar flags, histórico e métricas |
| SDK Client | Avaliar flags somente no ambiente autorizado |

No MVP, papéis são organizacionais e herdados por todos os projetos; somente Owner gerencia membros. Não há grants por projeto/flag. Bootstrap e recuperação administrativa serão CLI controlada, sem signup público ou senha padrão; último Owner protegido transacionalmente.

Casos principais: criar projeto; cadastrar ambiente `development`, `staging`, `production`; criar flag; ativar/desativar por ambiente; configurar rollout percentual; consultar flag no frontend/backend; auditar alterações; revogar token; restaurar serviço após falha.

Projetos novos recebem os ambientes development, staging e production na mesma transação. Ambientes adicionais indicam produção explicitamente; o slug production exige essa classificação. Nome/slug/classificação não têm edição neste incremento; exclusão/arquivamento aguardam política específica (ADR-009).

## 3. Requisitos funcionais

| ID | Prioridade | Requisito | Aceitação |
|---|---|---|---|
| RF-01 | P0 | Login administrativo | Sessão expira e logout invalida sessão |
| RF-02 | P0 | Organizações e membros | Cada operação valida escopo organizacional |
| RF-03 | P0 | Projetos e ambientes | Slugs únicos no escopo correto |
| RF-04 | P0 | Flags booleanas | Chave imutável após criação no MVP |
| RF-05 | P0 | Configuração por ambiente | Mudança em staging não altera produção |
| RF-06 | P0 | API de avaliação | Retorna valor, motivo e versão de configuração |
| RF-07 | P0 | Tokens restritos | Token de avaliação não administra flags |
| RF-08 | P0 | Auditoria | Registra ator, ação, alvo, timestamp e alterações não secretas |
| RF-09 | P0 | SDK JS | Cache, timeout e fallback configurável |
| RF-10 | P0 | Dashboard | CRUD, filtros, estados, confirmação de mudança |
| RF-11 | P1 | Rollout percentual | Bucketing estável por contexto e salt |
| RF-12 | P1 | Histórico de versões | Revisão otimista evita perda de atualizações |
| RF-13 | P1 | Métricas | Health, readiness e Prometheus |
| RF-14 | P2 | Variantes e segmentos | Projeto pós-MVP |
| RF-15 | P2 | SDK Go | Projeto pós-MVP |

### Regras essenciais
- Identificadores de flag usam `[a-z][a-z0-9_.-]{1,79}`.
- Flag desativada retorna `false` independentemente do rollout.
- Rollout 0% retorna `false`; 100% retorna `true`; sem `bucketingKey`, o servidor retorna `false`/`MISSING_CONTEXT` quando o rollout for parcial; o SDK aplica `defaultValue` configurado na chamada nessa razão. Flag ausente/arquivada retorna `false`/`NOT_FOUND`; o SDK também pode aplicar seu default nesse caso, sem sobrepor um `DISABLED` válido.
- Bucket: `SHA-256(projectId + ':' + environmentId + ':' + flagKey + ':' + salt + ':' + bucketingKey)`; interpretar os primeiros 8 bytes como uint64 big-endian, calcular `bucket = n % 10000` e retornar `bucket < rollout_bps` para rollout parcial habilitado. UTF-8, UUIDs canônicos, salt hexadecimal estável e sem normalização Unicode implícita; JS usa BigInt. Apenas bucketingKey pode conter o delimitador `:`. Fixar implementação e vetores de teste entre Go e JS.
- Configurações têm `revision` monotônica por ambiente; atualização exige `If-Match`/revisão para evitar escrita perdida.
- Flags não substituem autorização, verificação de assinatura, controle de acesso nem validação no servidor.

## 4. Requisitos não funcionais

**Segurança:** TLS, RBAC, segregação multi-tenant, mínimo privilégio, logs sem segredos, proteção CSRF em sessão cookie, CSP, CORS estrito, rate limiting, gestão de tokens, backups criptografados. **Confiabilidade:** timeouts, graceful shutdown, migrações reversíveis quando possível, backups e testes de restore. **Performance:** índices por ambiente/chave, consultas parametrizadas, cache por revisão. **Observabilidade:** logs JSON com request ID, métricas sem identificadores de usuário, traces opcionais. **Acessibilidade:** WCAG 2.2 AA como meta, teclado, contraste e mensagens de erro. **Responsividade:** desktop/tablet/mobile. **Privacidade:** minimização de contexto de avaliação, não persistir identificadores pessoais enviados pelo SDK.

## 5. Arquitetura

```text
React Admin (Vite) --> Go HTTP API (Chi)
                           |-- Auth/RBAC
                           |-- Admin API
                           |-- Evaluation API --> Evaluator determinístico
                           |-- Audit/Metrics
                           +--> PostgreSQL
SDK JS --> /v1/evaluate [token de ambiente]
Traefik/Coolify --> frontend e API por HTTPS
```

**Backend:** Go (versão estável suportada), Chi, pgx/sqlc (consultas tipadas), golang-migrate, OpenAPI 3.1, slog, Prometheus client. **Frontend:** React + TypeScript + Vite, TanStack Query, React Router, React Hook Form + Zod, biblioteca de componentes acessível, Vitest/Testing Library/Playwright. **Persistência:** PostgreSQL 16+ compatível com ambiente; Redis é opcional em fase posterior. **Infra:** Docker multi-stage, Compose para desenvolvimento, Coolify/Traefik para produção. Evitar acoplamento da avaliação ao dashboard.

### Estrutura do monorepo

```text
flagpilot/
  apps/api/cmd/server/
  apps/api/internal/{auth,flags,evaluation,projects,audit,storage,http}/
  apps/web/src/{app,features,components,api}/
  packages/sdk-js/src/
  api/openapi.yaml
  db/migrations/
  deploy/{compose.dev.yml,compose.prod.yml,README.md}/
  docs/{adr,threat-model,runbooks}/
  .github/workflows/
  README.md LICENSE .env.example
```

## 6. Modelo de dados PostgreSQL

| Tabela | Campos principais | Restrições |
|---|---|---|
| organizations | id UUID, name, slug, created_at | UNIQUE(slug) |
| users | id, email, password_hash, disabled_at, created_at | UNIQUE(lower(email)) via índice |
| memberships | organization_id, user_id, role | PK composta; role enum/check |
| projects | id, organization_id, name, slug | UNIQUE(organization_id,slug) |
| environments | id, project_id, name, slug, is_production BOOLEAN, revision BIGINT | UNIQUE(project_id,slug) |
| flags | id, project_id, key, description, archived_at | UNIQUE(project_id,key) |
| flag_configs | id, flag_id, environment_id, enabled, rollout_bps INT, salt, updated_at, version | UNIQUE(flag_id,environment_id); 0..10000 |
| api_tokens | id, project_id, environment_id, token_prefix, token_hash, scope, expires_at, revoked_at | Nunca guardar token em claro |
| audit_events | id, organization_id, actor_id nullable, action, resource_type, resource_id, before_redacted JSONB, after_redacted JSONB, created_at | Append-only lógico |
| sessions | id, user_id, token_hash, expires_at, revoked_at | Sessões revogáveis |

Índices: `flags(project_id,key)`, `flag_configs(environment_id,flag_id)`, `api_tokens(token_prefix)`, `audit_events(organization_id,created_at DESC)`. Chaves estrangeiras com políticas de exclusão explícitas; arquivamento preferido a deleção de flags usadas. Timestamps `timestamptz` em UTC. Aumentar `environments.revision` na mesma transação que modifica `flag_configs` e insere auditoria. Para organização/projeto, sempre filtrar pela associação autorizada (nunca apenas por ID recebido).

### Migração inicial (esboço)

```sql
CREATE TABLE organizations (
 id uuid PRIMARY KEY, name text NOT NULL,
 slug text NOT NULL UNIQUE, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE projects (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES organizations(id),
 name text NOT NULL, slug text NOT NULL,
 UNIQUE(organization_id, slug)
);
CREATE TABLE environments (
 id uuid PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects(id),
 name text NOT NULL, slug text NOT NULL, revision bigint NOT NULL DEFAULT 0,
 UNIQUE(project_id, slug)
);
CREATE TABLE flags (
 id uuid PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects(id),
 key text NOT NULL, description text NOT NULL DEFAULT '', archived_at timestamptz,
 UNIQUE(project_id, key)
);
CREATE TABLE flag_configs (
 id uuid PRIMARY KEY, flag_id uuid NOT NULL REFERENCES flags(id),
 environment_id uuid NOT NULL REFERENCES environments(id),
 enabled boolean NOT NULL DEFAULT false,
 rollout_bps integer NOT NULL DEFAULT 10000 CHECK (rollout_bps BETWEEN 0 AND 10000),
 salt text NOT NULL, version bigint NOT NULL DEFAULT 1,
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(flag_id, environment_id)
);
```

**Atenção:** migrations reais devem incluir as demais tabelas, índices, checagem de consistência entre projeto da flag e projeto do ambiente (via transação/trigger ou chave composta), e testes de integridade referencial.

## 7. Contrato HTTP / OpenAPI

Base: `/api/v1`. JSON UTF-8. Erros padronizados `application/problem+json` (RFC 9457): `type`, `title`, `status`, `detail`, `instance`, `requestId`. Paginação cursor para coleções; limites máximos. `401` sem credenciais; `403` sem permissão; `404` para recurso fora do escopo; `409` conflito; `422` validação; `429` limite. Autenticação admin por sessão segura cookie HttpOnly/SameSite e CSRF; SDK por `Authorization: Bearer <evaluation-token>` somente em ambientes confiáveis. **Não embutir token secreto em navegador público**: o MVP usa BFF da aplicação consumidora, com escopo e allowlist próprios. Endpoint público/chave publicável fica adiado. Ausência de If-Match obrigatório retorna 428; precondição falsa retorna 412; conflito de domínio continua 409.

| Método | Endpoint | Acesso | Objetivo |
|---|---|---|---|
| POST | `/auth/login` | Público | Criar sessão |
| POST | `/auth/logout` | Sessão | Revogar sessão |
| GET | `/me` | Sessão | Perfil e permissões |
| GET/POST | `/organizations` | Sessão | Listar/criar organizações |
| GET/POST | `/organizations/{orgId}/projects` | RBAC | Projetos |
| GET | `/projects/{projectId}` | Viewer+ | Detalhe e papel herdado no projeto |
| GET/POST | `/projects/{projectId}/environments` | RBAC | Ambientes |
| GET/POST | `/projects/{projectId}/flags` | RBAC | Listar/criar flags |
| GET | `/projects/{projectId}/flags/{flagKey}` | RBAC | Detalhe |
| PATCH | `/projects/{projectId}/flags/{flagKey}` | Editor+ | Metadados |
| PUT | `/environments/{envId}/flags/{flagKey}` | Editor+ | Configuração, revisão obrigatória |
| GET | `/environments/{envId}/flags` | Viewer+ | Configurações |
| POST | `/environments/{envId}/tokens` | Admin+ | Emitir token uma vez |
| DELETE | `/environments/{envId}/tokens/{tokenId}` | Admin+ | Revogar |
| GET | `/organizations/{orgId}/audit-events` | Viewer+ | Auditoria |
| POST | `/evaluate` | Token de avaliação | Avaliar uma flag |
| POST | `/evaluate/batch` | Token de avaliação | Avaliar várias flags |
| GET | `/health/live` | Público | Processo vivo |
| GET | `/health/ready` | Interno | Pronto, banco acessível |
| GET | `/metrics` | Rede interna | Prometheus |

### Exemplo de avaliação

```http
POST /api/v1/evaluate
Authorization: Bearer <token>
Content-Type: application/json

{"flagKey":"new_admin_dashboard","context":{"bucketingKey":"user-pseudonym-123"}}
```

```json
{
  "flagKey": "new_admin_dashboard",
  "value": true,
  "reason": "PERCENTAGE_ROLLOUT",
  "environment": "production",
  "revision": 18
}
```

### OpenAPI 3.1 — fragmento executável como ponto de partida

```yaml
openapi: 3.1.0
info:
  title: FlagPilot API
  version: 0.1.0
servers:
  - url: https://flagpilot-dev.franklem.uk/api/v1
paths:
  /evaluate:
    post:
      operationId: evaluateFlag
      security:
        - evaluationToken: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/EvaluationRequest'
      responses:
        '200':
          description: Evaluation result
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EvaluationResult'
        '401': {description: Invalid or missing token}
        '429': {description: Rate limit exceeded}
components:
  securitySchemes:
    evaluationToken:
      type: http
      scheme: bearer
  schemas:
    EvaluationRequest:
      type: object
      additionalProperties: false
      required: [flagKey]
      properties:
        flagKey: {type: string, minLength: 2, maxLength: 80}
        context:
          type: object
          additionalProperties: false
          properties:
            bucketingKey: {type: string, maxLength: 256}
    EvaluationResult:
      type: object
      additionalProperties: false
      required: [flagKey, value, reason, environment, revision]
      properties:
        flagKey: {type: string}
        value: {type: boolean}
        reason:
          type: string
          enum: [DISABLED, ENABLED, PERCENTAGE_ROLLOUT, MISSING_CONTEXT, NOT_FOUND]
        environment: {type: string}
        revision: {type: integer, format: int64}
```

**Definition of Done do contrato:** expandir `api/openapi.yaml` para **todos** os endpoints da tabela com request/response schemas, paginação, erros, exemplos, códigos HTTP, cookies/CSRF e permissões; validar com Spectral e gerar clientes/testes de contrato. O fragmento acima não é a especificação completa.

## 8. SDK JavaScript/TypeScript

**Pacote:** `@flagpilot/sdk-js` (nome proposto, verificar disponibilidade antes de publicar). Suportar Node.js e browser, com estratégias distintas de autenticação. API do pacote privado implementada para backend:

```ts
import { FlagPilotClient } from '@flagpilot/sdk-js';

const flags = new FlagPilotClient({
  baseUrl: 'https://flagpilot.franklem.uk/api/v1',
  token: process.env.FLAGPILOT_EVALUATION_TOKEN!, // somente backend
  timeoutMs: 1500,
  cacheTtlMs: 30000,
});

const enabled = await flags.isEnabled('new_admin_dashboard', {
  bucketingKey: 'pseudonymous-user-key',
  defaultValue: false,
});
```

Contrato: `isEnabled(key, options): Promise<boolean>`, `evaluate(key, options): Promise<EvaluationResult>`, `close(): void`; cache por `environment/token fingerprint + flag + bucketingKey`, nunca pelo nome da flag isoladamente. Timeout via `AbortController`, retries limitados somente para falhas transitórias, jitter, sem retry de `401/403/422`; fallback **fail-closed** por padrão. Não persistir tokens em localStorage; não logar `bucketingKey`; não armazenar contexto sensível. Browser público usa BFF no MVP; chave publicável/endpoint público fica para decisão posterior. Erros locais do SDK usam razões próprias e revision null; não inventar revisão remota. Versionamento SemVer, ESM/CJS conforme estratégia de build e declarações TypeScript.

**Testes SDK:** respostas válidas, 404, 401, timeout, offline, cache expirado, abort, concorrência, fallback, separação de ambientes e vetores de bucketing Go/JS.

## 9. Segurança e modelo de ameaças

| Ameaça | Mitigação | Teste obrigatório |
|---|---|---|
| IDOR/multi-tenant | Autorização em toda consulta e escrita | Usuário A não lê B |
| Token vazado | Hash no banco, prefixo, expiração, revogação e escopo | Token revogado falha |
| CSRF | Cookie SameSite + token CSRF em mutações | Rejeitar requisição sem token |
| XSS | React escaping, CSP, validação, sem HTML arbitrário | Payloads de injeção |
| Abuso API | Rate limiting por credencial/IP e limites de payload | 429 sob carga |
| Replay/concorrência | Revisão/If-Match e idempotência onde aplicável | If-Match desatualizado gera 412; conflito de domínio gera 409 |
| Exposição de contexto | Contexto mínimo, sem persistência, TLS | Logs sem identificadores |
| Falha da API | SDK timeout e fallback seguro | Serviço indisponível |
| Escalada por flags | Autorização independente da flag | Flag ligada não concede permissão |
| Supply chain | Dependências fixadas, SBOM, scan de imagens | CI bloqueia vulnerabilidades críticas |

Credenciais em variáveis secretas do Coolify, nunca no repositório. Passwords com Argon2id e política de rate limiting/MFA em fase posterior. Sessões revogáveis e rotação. Auditoria com retenção configurável e redaction. Backups criptografados e acesso mínimo.

## 10. Estratégia de testes

- **Unitários Go:** avaliador, limites percentuais, parser, RBAC, validação, revisão.
- **Integração:** PostgreSQL real via Testcontainers; migrations; isolamento de tenants; transações; revogação.
- **Contrato:** OpenAPI validado, testes positivos/negativos e compatibilidade de SDK.
- **Frontend:** componentes e acessibilidade (Testing Library/axe); fluxos Playwright.
- **E2E:** criar projeto → ambiente → flag → ativar → avaliar → auditar → revogar token.
- **Carga:** k6 com cenários 10/50/100 req/s, p50/p95/p99, erro, saturação; ajustar SLO à capacidade real da VPS.
- **Segurança:** OWASP ASVS nível 2 como referência, SAST, SCA, secret scan, DAST em homologação, fuzzing de entradas.
- **Resiliência:** banco indisponível, perda de rede, reinício, cache expirado, recuperação de backup.
- **Matriz:** Chrome/Firefox; viewport mobile/tablet/desktop; Node LTS; Linux servidor.

**Gate de release:** CI verde; cobertura alvo ≥80% para lógica de avaliação e autorização (cobertura não substitui qualidade); zero falhas críticas/altas conhecidas sem decisão formal; E2E aprovado; rollback testado.

## 11. Roadmap e backlog

| Etapa | Entregas | Critério de saída | Estimativa indicativa |
|---|---|---|---|
| 0 — Discovery | ADRs, threat model, UX wireframes, OpenAPI inicial | Escopo MVP aprovado | 2–3 dias |
| 1 — Fundação | Monorepo, CI, Docker, Go, Postgres, migrations | Health + testes DB | 4–6 dias |
| 2 — Identidade | Login, sessões, organizações, RBAC | Testes de isolamento | 5–7 dias |
| 3 — Flags | Projetos, ambientes, CRUD, auditoria, revisão | Fluxo admin completo | 5–7 dias |
| 4 — Avaliação | API, algoritmo, tokens, rate limiting | Vetores determinísticos | 4–6 dias |
| 5 — SDK JS | Cliente Node, cache, timeout, fallback | Testes de contrato | 3–5 dias |
| 6 — Dashboard | React, formulários, estados, auditoria | E2E e a11y | 5–8 dias |
| 7 — Homologação | Coolify, DNS, TLS, observabilidade | Smoke tests e backup | 3–4 dias |
| 8 — Produção | Hardening, benchmark, docs, rollback | Checklist de release | 3–5 dias |

Estimativa total: **34–51 dias úteis** de esforço, variando conforme dedicação, experiência e escopo. Rollout percentual entra após o núcleo booleano e antes do release 1.0 se não comprometer o MVP. Pós-MVP: segmentos, variantes, SDK Go, SSO/MFA, aprovação de mudanças, webhooks, Redis, Cloud comercial.

### Issues iniciais recomendadas
1. Criar monorepo e CI.
2. Especificar OpenAPI integral e gerar mocks.
3. Criar migrations e testes de integridade.
4. Implementar autenticação e RBAC.
5. Implementar CRUD de projetos/ambientes.
6. Implementar flags e auditoria transacional.
7. Implementar tokens e avaliação determinística.
8. Implementar SDK JS e vetores de teste compartilhados.
9. Criar dashboard e E2E.
10. Configurar homologação, backups, observabilidade e runbooks.

## 12. Deploy na VPS Contabo via Coolify

**Ambientes propostos (não presumir DNS criado):** `flagpilot-dev.franklem.uk` e `flagpilot.franklem.uk`. **Pré-requisitos:** acesso autorizado à VPS, Coolify/Traefik funcionais, DNS apontado, TLS, backups, recursos livres verificados. Não modificar serviços existentes sem inventário e plano de rollback.

### Topologia
- Stack homologação separada de produção, cada qual com banco e secrets próprios.
- API e frontend com serviços independentes; PostgreSQL em rede privada sem porta pública.
- HTTPS pelo Traefik; somente 80/443 públicos (e SSH restrito conforme política atual).
- `/health/live` sem dependência externa; `/health/ready` verifica banco; `/metrics` somente rede interna.
- Imagens com tags imutáveis/digest; usuário não-root; filesystem read-only onde viável; limites de CPU/memória; logs com rotação.

### Sequência de implantação
1. Verificar capacidade livre (`docker stats`, disco, RAM), versões e backup atual.
2. Criar repositório Git privado durante desenvolvimento; `.env.example` sem valores reais.
3. Criar projeto Coolify `flagpilot-dev` e serviços API, web, PostgreSQL isolados.
4. Cadastrar secrets: `DATABASE_URL`, `SESSION_SECRET`, `CSRF_SECRET`, `ALLOWED_ORIGINS`, `PUBLIC_BASE_URL` e parâmetros de conexão; gerar valores seguros fora do Git.
5. Configurar DNS do ambiente de homologação e HTTPS; validar certificados.
6. Rodar migrations como job controlado antes de atualizar API; garantir compatibilidade entre versões durante rollout.
7. Implantar imagens de CI por digest; testar health/readiness.
8. Rodar smoke tests e E2E contra homologação; executar teste de restore em instância isolada.
9. Criar projeto `flagpilot-prod` com banco, volumes, credenciais e domínio próprios.
10. Promover a **mesma imagem testada** para produção; verificar métricas e logs após deploy.

### Backup e recuperação
- `pg_dump -Fc` periódico + cópia criptografada fora da VPS; retenção definida (exemplo inicial: 7 diários e 4 semanais, ajustável).
- Validar restauração em PostgreSQL isolado ao menos mensalmente; registrar duração e integridade.
- Documentar RPO/RTO alvo inicial: RPO ≤24 h, RTO ≤4 h **somente após teste demonstrar viabilidade**.
- Guardar segredos de recuperação em local separado; nunca anexar dumps ao GitHub.

### Rollback
- Reverter imagem para digest anterior no Coolify.
- Migrações destrutivas exigem plano de expansão/contração e backup; rollback de imagem não garante rollback de schema.
- Se banco mudou de forma incompatível, executar procedimento de recuperação previamente ensaiado, avaliando perda de dados.
- Monitorar 5xx, latência, disponibilidade e erro de avaliação durante e após promoção.

### Checklist de produção
- [ ] DNS/TLS válidos; CORS e CSP restritos.
- [ ] PostgreSQL inacessível publicamente.
- [ ] Credenciais distintas por ambiente; tokens mínimos.
- [ ] Autenticação, RBAC, CSRF, rate limit e auditoria testados.
- [ ] Migrations revisadas; backup e restore testados.
- [ ] Health, métricas, logs e alertas configurados.
- [ ] Imagens verificadas e fixadas por digest.
- [ ] SDK com fallback seguro e sem segredo no browser.
- [ ] Runbook de incidentes e rollback disponível.
- [ ] Documentação e licença revisadas para GitHub.

## 13. CI/CD GitHub Actions

**PR:** format/lint Go + TS; unitários; integração PostgreSQL; Spectral OpenAPI; testes de segurança/secret scan; build. **Main:** gerar SBOM, construir imagens multi-stage, escanear dependências/imagens, publicar em registry com digest. **Deploy futuro (não autorizado nesta fase):** homologação automática somente após autorização de configuração externa e aprovação de produção manual; credenciais de deploy por OIDC quando suportado, senão secrets com escopo mínimo. Evitar deploy automático de PR não confiável. Releases com tags SemVer e changelog.

## 14. UX/UI e telas

Telas: Login; seleção de organização; projetos; ambientes; lista de flags com status por ambiente; criação/edição de flag; rollout; tokens (valor mostrado uma única vez); auditoria; configurações; estados vazios/erro. Em produção, alterações exigem confirmação com resumo do impacto. Exibir claramente ambiente atual com badge persistente; nunca depender apenas de cor. Padrões de acessibilidade, feedback de salvamento, skeleton loading e navegação por teclado.

## 15. Riscos, decisões e pendências

| Risco | Resposta |
|---|---|
| Expor token no navegador | BFF ou credencial publicável de escopo limitado; não usar token secreto |
| Rollout instável | Vetores compartilhados e hashing determinístico versionado |
| Alteração acidental em produção | Confirmação, RBAC, auditoria e revisão otimista |
| VPS sem capacidade | Medir antes; limitar workers/serviços e ajustar sizing |
| Migração irreversível | Expand/contract, backup e ensaio de recuperação |
| Feature flag usada como permissão | Documentar e testar autorização independente |
| Sobrecarga de escopo | MVP booleano antes de segmentos/variantes |

**ADRs obrigatórias:** ADR-001 Go/Chi/pgx; ADR-002 auth por cookie; ADR-003 bucketing e compatibilidade SDK; ADR-004 estratégia de cache/fallback; ADR-005 multi-tenancy; ADR-006 deploy Coolify; ADR-007 modelo de token público/BFF.

## 16. Definition of Done

Uma funcionalidade só é concluída com: requisito rastreável; implementação revisada; testes automatizados; validação de permissão e entrada; observabilidade adequada; documentação/API atualizada; acessibilidade quando houver UI; migração e rollback avaliados; CI verde; validação em homologação. A versão 1.0 só será publicada após checklist de produção e recuperação validada.

## 17. Primeiro prompt para execução no VS Code/Codex ou Work

> Implemente a **Etapa 0 e a Etapa 1** do FlagPilot conforme este documento. Crie monorepo, estrutura Go/Chi, React/Vite, PostgreSQL, Docker Compose de desenvolvimento, migrations iniciais, health/readiness, Makefile, `.env.example`, lint, testes básicos e GitHub Actions. Não faça deploy nem crie DNS nesta etapa. Documente comandos para executar e testar, registre ADRs iniciais e reporte decisões que precisem de aprovação. Não adicione segredos reais e não implemente recursos pós-MVP.

---

**Nota:** versões de bibliotecas, nomes de domínio e capacidade da VPS devem ser confirmados na implantação. Este documento é um plano de implementação; não afirma que a infraestrutura já foi modificada.
