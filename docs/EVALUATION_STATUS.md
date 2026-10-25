# Avaliação — incrementos 4A e 4B

Data: 24/10/2026. Continuidade autorizada por Franklin. Aceitação manual anterior: Franklin confirmou criação, ativação em ambientes distintos e arquivamento de flags. Isso confirma os fluxos exercitados em 3B; não representa teste manual de toda a matriz de segurança.

## Entregas 4A

- Owner/Admin emitem e revogam tokens por ambiente; listagem somente de metadados. Segredo aleatório exibido uma vez; SHA-256 persistido, escopo evaluation e FK composta. Prazo obrigatório até 365 dias, sugestão de 90 dias na UI.
- Estado temporário da emissão fora de cache de consultas/mutações e storage/URL. Revogação explícita/idempotente com confirmação. Auditoria atômica sem segredo/hash/contexto.
- POST evaluate e evaluate/batch com Bearer exclusivo de servidor, ambiente inferido da credencial e validação de revogação/expiração a cada consulta. Sessão não avalia e Bearer não administra.
- Decisão booleana determinística, flag ausente/arquivada NOT_FOUND, desativada/0 DISABLED, habilitada/10000 ENABLED. Sem percentuais intermediários nesta etapa.
- Batch no mesmo snapshot, ordem/duplicatas preservadas; body 64 KiB, até 100 itens e contexto validado sem persistência. Rate limit em memória limitada, Retry-After e peer real.
- Migration aditiva 000005, schema 5 no readiness. OpenAPI, tipos gerados, navegação e uso atualizados. Nenhuma nova dependência ou variável secreta do servidor.

## Verificação

`make ci` aprovado localmente: gofmt/vet, ESLint, Spectral sem avisos, tipos TypeScript, Go com race detector, 13 testes frontend, PostgreSQL real e builds. Chromium: 6 fluxos aprovados em stack descartável, incluindo tokens/avaliação. Cobertura: política de papéis, segredo/hash, isolamento de ambiente/projeto, expiração, revogação, rollback de auditoria, lote consistente sob escrita concorrente, ausência de segredo/contexto em metadata/auditoria, constraints/privilégios SQL, limites de taxa/body/contexto e upgrades com identidade/projeto/ambiente/flags preservados. Chromium exercita emissão única, armazenamento do browser vazio, avaliação single/batch, separação entre ambientes e revogação. Viewer sem gestão é verificado no fluxo autenticado de projetos, evitando logins adicionais que excedam a proteção existente.

CI está configurada; sem execução no GitHub, publicação de credenciais/pacotes ou deploy externo. Franklin confirmou manualmente emissão com validades diferentes, cópia, ocultação e revogação no painel. A aceitação cobre esses fluxos; API e segurança foram verificadas automaticamente. Performance/SLO, restauração e implantação em VPS não foram medidos/executados.

## Uso e continuidade

Ver [tokens e avaliação](runbooks/evaluation.md) e [ADR-011](adr/011-evaluation-tokens.md). Acesse Gerenciar tokens na lista de ambientes, emita uma credencial explicitamente e teste pelo terminal/backend. Não enviar tokens pelo chat.

Continuidade original de 4A: rollout 4B, implementado abaixo, e etapa 5 (SDK). Não há SDK funcional, chave publicável, cache de avaliação ou consulta de auditoria nesta etapa.

Containers locais atualizados com migration 000005 e saudáveis, sem remover o volume ou emitir credenciais reais automaticamente. A validação da instalação local mantém os dados anteriores; nenhum servidor externo foi alterado.

Verificações finais: sqlc/OpenAPI regenerados sem diferenças; Gitleaks staged sem vazamentos; readiness HTTP 200, frontend HTTP 200 e avaliação sem Bearer HTTP 401. A aplicação local permanece disponível para aceitação manual; não compartilhar o segredo emitido.

## Incremento 4B — rollout determinístico

Continuidade autorizada por Franklin após aceitação da gestão de tokens. Avaliação percentual single/batch usa SHA-256, primeiros 8 bytes uint64 big-endian e módulo 10000, conforme ADR-003. UUIDs canônicos e salt persistido por configuração; contexto apenas em memória. Parcial sem contexto retorna false/MISSING_CONTEXT; com contexto retorna PERCENTAGE_ROLLOUT e seu valor calculado. Desativada/0%, 100% e ausente/arquivada preservam os motivos anteriores.

Editor+ configura 0 a 100%, com até duas casas decimais. Vírgula/ponto são aceitos e convertidos exatamente em pontos-base. Ativar/desativar preserva o percentual e salvar percentual preserva o estado. Confirmação em produção mostra valores antes/depois; revisão/If-Match e auditoria permanecem atômicos. Salt é preservado e não exposto em resposta administrativa/avaliação. Não houve migration, alteração de dependências ou variáveis de ambiente; schema continua 5.

### Verificação 4B

- `make ci`: lint/tipos/OpenAPI, Go com race detector, 17 testes frontend, testes reais PostgreSQL e builds Go/web/ESM/CJS.
- Onze vetores compartilhados versionados em `testdata/bucketing-v1.json`, lidos por Go e JavaScript compilado de TypeScript em ESM/CJS: UTF-8, dois-pontos, Unicode sem normalização, máximo de 256 bytes, mudanças de projeto/ambiente/flag/salt e buckets extremos 0/9999. Limiares exatos e monotonicidade verificados no avaliador Go.
- Integração: percentual fracionário, salt estável após múltiplas edições, isolamento de ambiente, confirmação de produção, revisão/versão, duplicatas/ordem, ausência de contexto e lote consistente sob alterações percentuais concorrentes. Testes anteriores de RBAC, auditoria/rollback, revogação e integridade continuam aprovados.
- Chromium: seis fluxos aprovados em stack descartável, com edição 37,25%, preservação ao alternar estado, cancelamento/confirmação em produção, conflito de revisão e avaliação parcial single/batch via HTTP. Credenciais e volume de testes isolados dos dados reais.

A referência TypeScript do algoritmo é interna e não exporta um cliente público. Próxima etapa: 5, SDK HTTP com cache, timeout e fallback. Aceitação manual do novo campo de percentual segue com Franklin. Distribuição aproximada por contexto não é quota exata; benchmark/SLO, homologação e produção continuam pendentes.

Containers locais reconstruídos e saudáveis, volume existente preservado. Readiness/frontend HTTP 200, novo campo servido pelo frontend e avaliação sem Bearer HTTP 401. Regeneração sqlc/OpenAPI sem diferenças e Gitleaks staged aprovado. Os dois vetores extremos usam identificadores sintéticos preenchidos com zeros, sem supressão de regras de detecção. Nenhuma credencial real emitida pelos testes e nenhum servidor externo alterado.

## Aceitação manual e continuidade — 25/10/2026

Franklin confirmou visualização e salvamento de percentuais diferentes no painel e autorizou a etapa 5. Isso confirma os cenários exercitados de configuração; não presume aceitação manual da distribuição determinística via HTTP. SDK entregue em [SDK_STATUS.md](SDK_STATUS.md); cache do consumidor pode retardar a observação de mudanças/revogação até o TTL.
