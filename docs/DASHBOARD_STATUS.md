# Incremento 6A — filtros e auditoria

Data: 25/10/2026. Continuidade autorizada por Franklin após entrega do SDK. A autorização para continuar não representa aceitação manual do SDK ou das novas telas; os cenários abaixo foram verificados automaticamente.

## Entregas

- Catálogo de flags com busca literal por trecho da chave e estado all/active/archived, aplicados no PostgreSQL antes de paginar. Parâmetros normalizados, limites e validação; cursor vinculado ao ator/projeto/filtros. Cache invalidado por prefixo após mutações, abrangendo todas as combinações de filtros.
- Histórico por organização em GET /organizations/{orgId}/audit-events, somente leitura para Owner/Admin/Editor/Viewer com sessão ativa. Organização bloqueada SHARE e associação revalidada após espera; sem autorização por Bearer de avaliação.
- Paginação temporal createdAt/id decrescente, com desempate estável, âncora limitada à organização e filtros exatos action/resourceType. JSON Before/After projetado por campos escalares conhecidos; segredo/hash/salt/contexto e estruturas desconhecidas não são devolvidos.
- Navegação Ver auditoria na conta/projetos; filtros na URL, estados de carregamento/erro/vazio, atualização, carregar mais, detalhes Antes/Depois em controle nativo de teclado. Foco no título ao entrar, labels e status acessíveis. Percentuais legíveis em %.
- Sem nova dependência, variável ou migration; schema 5 e dados existentes preservados. Não há endpoint de alteração, restauração ou limpeza de auditoria.

## Verificação

`make ci` aprovado: gofmt/vet, ESLint, Spectral sem avisos, tipos, Go com race detector, 19 testes frontend, 16 testes SDK com tipos ESM/CJS/vetores, PostgreSQL real e builds. Integração adicional cobre busca/status antes da paginação, cursores vinculados, ordem com horários iguais, todos os papéis, isolamento organizacional, conta desativada, remoção enquanto leitor espera lock, validação HTTP e projeção de campos sensíveis/desconhecidos. Testes existentes de transações/rollback/If-Match e auditoria continuam aprovados.

Seis fluxos Chromium passaram em stack descartável: filtros vazio/ativas/arquivadas e persistência ao recarregar, histórico de mudanças de configuração, detalhe aberto por Enter, foco no título, viewport 375 px sem overflow, leitura por Viewer e negação de outra organização. Login/projetos/flags/tokens/SDK continuam exercitados sem logins adicionais que contornem o limitador. Unitários verificam busca literal/normalizada e HTML exibido como texto.

## Uso e limites

Ver [runbooks/dashboard.md](runbooks/dashboard.md) e [ADR-012](adr/012-dashboard-audit-query.md). Conta → Ver auditoria; catálogo do projeto → Buscar chave/Estado no catálogo → Aplicar filtros. Ctrl+Shift+R após atualizar a stack local. Cursores anteriores do catálogo devem ser reiniciados.

A auditoria identifica o ator por UUID/rótulo e apresenta somente metadados permitidos; não reconstrói nomes históricos nem garante sanitização de segredos que usuários tenham digitado intencionalmente em nomes/descrições. Paginação não promete snapshot congelado entre requests. Busca por descrição/data, exportação, restauração, retenção automática e índices adicionais aguardam escopo/medição.

Acessibilidade verificada nos cenários descritos não é certificação WCAG integral; Firefox e matriz completa de viewports/estados seguem pendentes. Próximo incremento: 6B, fechamento de UX/acessibilidade e critérios operacionais do MVP. Licença, publicação, benchmark/SLO, homologação externa e produção permanecem pendentes; nenhum servidor externo alterado.

Verificações finais: containers locais reconstruídos e saudáveis, volume preservado, readiness/frontend HTTP 200 e consulta de auditoria sem sessão HTTP 401. Nova tela servida pelo frontend local. sqlc/OpenAPI regenerados sem diferenças; Gitleaks staged sem vazamentos. Nenhuma credencial real criada pelos testes nem servidor externo modificado.
