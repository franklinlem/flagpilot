# ADR-004 — Cache e fallback

Data: 24/10/2026. Estado: Aceita como direção; implementação no incremento 5.

## Contexto

Clientes precisam de disponibilidade sem prometer consistência imediata de valores locais.

## Decisão

Avaliação remota, cache em memória limitado por instância/baseURL/credencial/flag/contexto; TTL 30 s e timeout total 1500 ms; até um retry transitório com jitter dentro do orçamento. Fallback false por padrão; erro local tem reason próprio e revision null. Default por chamada aplica em MISSING_CONTEXT e NOT_FOUND, sem sobrepor DISABLED válido.

## Alternativas

Avaliação offline completa e cache distribuído não entram no MVP.

## Consequências

Revogação e mudança podem levar até o TTL para aparecer no cliente. Sem cache de autenticação na API inicialmente. SDK ainda está vazio; contrato local será testado antes do cliente real.
