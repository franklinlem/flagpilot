# ADR-005 — Tenancy, papéis e revisão

Data: 24/10/2026. Estado: Aceita; autorização de organizações/membros implementada em 2B.

## Contexto

Há apenas memberships organizacionais na fonte, embora o texto mencione autorização por projeto.

## Decisão

MVP com papéis organizacionais herdados: Owner gerencia membros/tudo; Admin projetos/ambientes/tokens/flags; Editor flags; Viewer leitura sem segredo. Último Owner protegido transacionalmente. Configurações/tokens terão FKs compostas por projeto. Atualizações exigem ETag/If-Match: 428 ausente, 412 falso, 409 conflito de domínio.

## Alternativas

Grants por projeto/flag e RLS podem ser avaliados depois; não simular permissões inexistentes no schema.

## Consequências

Cada consulta valida escopo; metadata da flag terá versão própria. Revisão ambiental e auditoria mudam na mesma transação. Versões históricas restauráveis ficam fora do núcleo. Identidade e gestão de membros/organizações estão implementadas com releitura de papel no banco, isolamento e último Owner ativo. Ver ADR-008 para locks e cadastro local; permissões dos demais recursos serão aplicadas junto de cada módulo.
