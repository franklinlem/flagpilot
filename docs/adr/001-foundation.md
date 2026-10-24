# ADR-001 — Fundação modular

Data: 24/10/2026. Estado: Aceita e iniciada.

## Contexto

Precisamos evoluir no Debian com poucos serviços e manter o futuro deploy self-hosted simples.

## Decisão

Monólito Go/Chi com módulo único, pgx/sqlc e SQL versionado; React/TypeScript/Vite, npm workspaces e SDK independente. Versões exatas registradas em manifests/lockfiles; TypeScript 5.9.3 atende o peer dependency do openapi-typescript 7.13.0. Sem Redis, ORM ou microserviços.

## Alternativas

Microserviços, ORM e gerenciador de monorepo adicional aumentariam a operação sem necessidade demonstrada.

## Consequências

Módulos por domínio serão criados quando tiverem código; não criar serviços/pacotes vazios. Sqlc e tipos OpenAPI gerados são versionados e verificados na CI.
