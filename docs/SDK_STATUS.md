# Incremento 5 — SDK HTTP TypeScript/JavaScript

Data: 25/10/2026. Continuidade autorizada por Franklin após visualizar/salvar porcentagens diferentes no painel. Essa aceitação cobre os fluxos de configuração exercitados, sem presumir teste manual da distribuição via HTTP.

## Entregas

Pacote privado `@flagpilot/sdk-js` 0.1.0 para backend Node 24.x, validado em 24.22.1. Exports ESM/CJS e declarações TypeScript, evaluate/isEnabled/close, validação de configuração/argumentos/respostas, timeout total e até um retry transitório. Nenhuma dependência nova, migration, mudança no servidor ou credencial real criada automaticamente. Schema permanece 5.

Cache LRU em memória limitado por instância com URL/credencial fixas, flag e contexto, TTL padrão 30 s, sem cache de falhas nem valor expirado como fallback. Default por chamada não altera o resultado bruto cacheado; aplica em falhas locais e MISSING_CONTEXT/NOT_FOUND, sem sobrepor DISABLED/PERCENTAGE_ROLLOUT válido. Falha local tem ambiente/revisão null e motivo SDK_*.

Requests equivalentes deduplicados, cancelamento independente por consumidor, concorrência limitada e close idempotente. Revogação na API é imediata para novas chamadas; hits do cache podem durar até o TTL. HTTP 401/403 limpa cache e bloqueia repopulação por respostas anteriores em voo. Credencial em campo privado, sem logging/persistência/storage público; contexto temporariamente retido nos mapas limitados. Não há promessa de apagar strings do runtime.

HTTPS fora de loopback, redirects recusados, cookies omitidos e JSON de resposta até 64 KiB. Cliente recusa window.document; browser público continua via BFF autorizado da aplicação consumidora. Sem adapter genérico público, avaliação offline, APIs administrativas ou cliente batch nesta entrega.

## Verificação

- `make ci` passou: lint/tipos/Spectral, Go com race detector, 17 testes frontend, PostgreSQL real e builds/API/web/SDK/import/require. Após refinamentos de cancelamento/validação, suíte final do SDK aprovada com 16 testes e compilação de consumidores TypeScript ESM/CJS; lint novamente aprovado.
- SDK: TTL/LRU/limites, defaults por chamada, falha fechada, respostas inválidas/oversize, rede/404/401/403/422/429/503, retry limitado e orçamento, body/transporte que ignore signal, abort independente, cancelamento do último consumidor, close, concorrência, isolamento de contexto/instância/credencial/URL e invalidação após 401 com resposta anterior em voo. Vetores compartilhados Go/JS continuam aprovados; UTF-8/Unicode/BOM e bytes exatos preservados.
- Seis fluxos Chromium aprovados em stack descartável. O fluxo de tokens usa o SDK Node contra a API real para avaliar booleano, rollout, MISSING_CONTEXT/default, cache eventualmente consistente e revogação sem cache. Sessões/credenciais/volume efêmeros separados dos dados locais de Franklin.
- Exemplo executável e instruções em [runbooks/sdk.md](runbooks/sdk.md), API/limites em [README do pacote](../packages/sdk-js/README.md) e decisão em [ADR-004](adr/004-cache-fallback.md).

## Continuidade e limites

Próxima etapa: 6, consolidação do painel com filtros, consulta de auditoria e acessibilidade. Nome público, licença, publicação npm/GitHub, homologação externa, benchmark/SLO e produção permanecem pendentes. A interface existente e os dados locais são preservados; nenhuma implantação externa foi realizada. O SDK pode ser integrado via workspace ou tarball local, sem publicação no registry.

Verificações finais: tarball privado 0.1.0 contém somente dist/README/package.json; instalação offline e import/require aprovados em consumidor descartável separado. Gitleaks staged sem vazamentos. Readiness e frontend locais HTTP 200. Serviços/dados de desenvolvimento preservados, sem necessidade de rebuild da API ou migration neste incremento.
