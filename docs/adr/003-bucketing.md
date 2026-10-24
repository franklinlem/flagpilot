# ADR-003 — Bucketing determinístico v1

Data: 24/10/2026. Estado: Aceita como direção; implementação no incremento 4B.

## Contexto

A especificação define o hash, mas não explicita a redução para 10.000 faixas.

## Decisão

SHA-256 dos componentes separados por dois-pontos na ordem da fonte, UTF-8 e UUIDs canônicos; primeiros 8 bytes uint64 big-endian; bucket=n%10000; enabled e bucket<rollout_bps. Salt hex fixo; JS BigInt; sem normalização Unicode. Somente bucketingKey pode conter delimitador.

## Alternativas

Comparar floats ou converter uint64 para Number perderia precisão; trocar fórmula depois redistribuiria usuários.

## Consequências

Vetores versionados Go/JS serão adicionados antes do rollout. Desativada sempre false, 0% false, 100% true; contexto ausente no parcial retorna false/MISSING_CONTEXT. Booleano vem primeiro.
