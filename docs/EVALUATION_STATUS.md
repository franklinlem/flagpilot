# Incremento 4A — tokens e avaliação booleana

Data: 24/10/2026. Continuidade autorizada por Franklin. Aceitação manual anterior: Franklin confirmou criação, ativação em ambientes distintos e arquivamento de flags. Isso confirma os fluxos exercitados em 3B; não representa teste manual de toda a matriz de segurança.

## Entregas

- Owner/Admin emitem e revogam tokens por ambiente; listagem somente de metadados. Segredo aleatório exibido uma vez; SHA-256 persistido, escopo evaluation e FK composta. Prazo obrigatório até 365 dias, sugestão de 90 dias na UI.
- Estado temporário da emissão fora de cache de consultas/mutações e storage/URL. Revogação explícita/idempotente com confirmação. Auditoria atômica sem segredo/hash/contexto.
- POST evaluate e evaluate/batch com Bearer exclusivo de servidor, ambiente inferido da credencial e validação de revogação/expiração a cada consulta. Sessão não avalia e Bearer não administra.
- Decisão booleana determinística, flag ausente/arquivada NOT_FOUND, desativada/0 DISABLED, habilitada/10000 ENABLED. Sem percentuais intermediários nesta etapa.
- Batch no mesmo snapshot, ordem/duplicatas preservadas; body 64 KiB, até 100 itens e contexto validado sem persistência. Rate limit em memória limitada, Retry-After e peer real.
- Migration aditiva 000005, schema 5 no readiness. OpenAPI, tipos gerados, navegação e uso atualizados. Nenhuma nova dependência ou variável secreta do servidor.

## Verificação

`make ci` aprovado localmente: gofmt/vet, ESLint, Spectral sem avisos, tipos TypeScript, Go com race detector, 13 testes frontend, PostgreSQL real e builds. Chromium: 6 fluxos aprovados em stack descartável, incluindo tokens/avaliação. Cobertura: política de papéis, segredo/hash, isolamento de ambiente/projeto, expiração, revogação, rollback de auditoria, lote consistente sob escrita concorrente, ausência de segredo/contexto em metadata/auditoria, constraints/privilégios SQL, limites de taxa/body/contexto e upgrades com identidade/projeto/ambiente/flags preservados. Chromium exercita emissão única, armazenamento do browser vazio, avaliação single/batch, separação entre ambientes e revogação. Viewer sem gestão é verificado no fluxo autenticado de projetos, evitando logins adicionais que excedam a proteção existente.

CI está configurada; sem execução no GitHub, publicação de credenciais/pacotes ou deploy externo. Aceitação visual/manual das novas telas segue com Franklin. Performance/SLO, restauração e implantação em VPS não foram medidos/executados.

## Uso e continuidade

Ver [tokens e avaliação](runbooks/evaluation.md) e [ADR-011](adr/011-evaluation-tokens.md). Acesse Gerenciar tokens na lista de ambientes, emita uma credencial explicitamente e teste pelo terminal/backend. Não enviar tokens pelo chat.

Próximo incremento: 4B, rollout percentual determinístico e vetores Go/JS; depois etapa 5, SDK. Não há SDK funcional, chave publicável, cache de avaliação ou consulta de auditoria nesta etapa.

Containers locais atualizados com migration 000005 e saudáveis, sem remover o volume ou emitir credenciais reais automaticamente. A validação da instalação local mantém os dados anteriores; nenhum servidor externo foi alterado.

Verificações finais: sqlc/OpenAPI regenerados sem diferenças; Gitleaks staged sem vazamentos; readiness HTTP 200, frontend HTTP 200 e avaliação sem Bearer HTTP 401. A aplicação local permanece disponível para aceitação manual; não compartilhar o segredo emitido.
