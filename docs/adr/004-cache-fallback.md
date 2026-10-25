# ADR-004 — Cache e fallback

Data: 24/10/2026; implementação: 25/10/2026. Estado: Aceita e implementada localmente no incremento 5.

## Contexto

Clientes precisam de disponibilidade sem prometer consistência imediata de valores locais.

## Decisão

Avaliação remota, cache em memória limitado por instância/baseURL/credencial/flag/contexto; TTL 30 s e timeout total 1500 ms; até um retry transitório com jitter dentro do orçamento. Fallback false por padrão; erro local tem reason próprio e revision null. Default por chamada aplica em MISSING_CONTEXT e NOT_FOUND, sem sobrepor DISABLED válido.

## Alternativas

Avaliação offline completa e cache distribuído não entram no MVP.

## Consequências

Revogação e mudança podem levar até o TTL para aparecer no cliente. Sem cache de autenticação na API inicialmente. SDK privado 0.1.0 implementado; contrato local e integração HTTP são testados.

## Detalhamento do incremento 5

Cliente exclusivamente backend Node 24.x, com ESM/CJS e declarações; browser público continua via BFF, sem adapter genérico ou segredo no bundle. Avaliação individual remota por evaluate/isEnabled; nenhuma mudança na API ou migration/schema 5. Licença, nome público e publicação seguem pendentes.

Cache LRU padrão 1000 entradas, teto 10000; TTL configurável 0–300000 ms e padrão 30000. Mapas privados por instância fixam URL/credencial e isolam flag/contexto; cache guarda resultado remoto antes de aplicar default por chamada. Sem cache de falhas nem fallback de valor expirado. TTL invalida a leitura e remoção acontece em acesso/inserção/close, sem timers de limpeza. Ao observar 401/403, limpa cache e usa uma geração para impedir repopulação por respostas anteriores em voo.

Até 100 operações distintas simultâneas por padrão (configurável 1–1000); demais retornam SDK_BUSY. Equivalentes compartilham operação, com default e abort independentes por consumidor. O último cancelamento aborta transporte/libera slot; close é idempotente, limpa mapas e impede novas consultas. Timeout total de 1–30000 ms inclui download e retry; mesmo transporte injetado que ignore abort não bloqueia indefinidamente o consumidor.

Retry máximo 1 (ou 0), somente rede ou 408/429/500/502/503/504, com jitter e Retry-After dentro do orçamento. Redirects recusados, cookies omitidos, HTTPS fora de loopback e resposta JSON limitada a 64 KiB. Configuração/chamada inválida rejeita com TypeError; falhas operacionais retornam motivos SDK_* e revision/environment null. Não expor mensagem bruta de transporte ou corpo HTTP.

A API de avaliação confirma revogação a cada chamada; cache do SDK é eventualmente consistente até o TTL. Aplicações que precisem observar imediatamente usam cacheTtlMs: 0. Flags/default true não substituem autorização. Não há telemetria ou persistência de contexto; memória pode reter contexto durante cache/request e não há promessa de apagar strings do runtime.
