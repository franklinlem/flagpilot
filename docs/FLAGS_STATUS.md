# Incremento 3B — flags booleanas

Data: 24/10/2026. Implementação local autorizada por Franklin. Aceitação manual anterior: Franklin criou projetos e confirmou os ambientes automáticos development/staging/production, com produção identificada. Isso confirma os fluxos exercitados em 3A, sem representar teste manual de toda a matriz de segurança.

## Entregas

- Catálogo de flags por projeto: criar, listar, consultar, editar descrição e arquivar. Chave imutável/reservada; sem exclusão ou restauração.
- Configuração por ambiente: consultar snapshot/configuração e ativar/desativar, sem afetar outros ambientes. Owner/Admin/Editor escrevem; Viewer lê.
- If-Match/ETag: 428 sem revisão, 412 se alterada; UI exige recarregamento antes de tentar de novo. Versões individuais e revisão ambiental avançam transacionalmente.
- Confirmação de produção consultando classificação persistida, inclusive slug personalizado. Arquivamento avisa sobre todos os ambientes e requer confirmação quando há produção.
- Inicialização atômica flag/ambientes com estado desativado; criação concorrente de flag/ambiente serializada no projeto. Configurações têm FKs compostas e salt persistente não exposto.
- Auditoria atômica de todas as mutações. Nenhum endpoint de consulta de auditoria antecipado.
- Migration aditiva 000004; readiness schema 4. JSON validado/limitado, OpenAPI e tipos gerados atualizados. Nenhuma dependência/variável secreta nova.
- UI com navegação catálogo/detalhe/configuração, confirmação, estados de erro/vazio, paginação e cache por identidade/escopo.

## Verificação local

`make ci` aprovado: gofmt/vet, ESLint, Spectral sem avisos, tipos TypeScript, Go/race, 12 testes frontend, integração PostgreSQL real e builds. Chromium: 5 fluxos aprovados em stack descartável (autenticação, cursor, flags, organizações/membros e projetos/ambientes). Testes cobrem papéis, isolamento, revisão e confirmação, staging separado de produção, dois escritores da mesma revisão, criação concorrente flag/ambiente, autorização revogada durante espera, paginação vinculada ao snapshot, rollback de auditoria, arquivamento/chave reservada e restrições SQL. Upgrade 3→4 exercita projeto/ambiente existente com identidade preservada. Chromium exercita criação/descrição, ativação, cancelamento/confirmação em produção, conflito entre abas, Viewer e arquivamento.

CI está configurada; nenhuma execução no GitHub ou deploy externo foi realizada. Aceitação visual/manual das novas telas permanece com Franklin.

## Uso e continuidade

Consulte [uso de flags](runbooks/flags.md) e [ADR-010](adr/010-boolean-flags.md). Atualize http://127.0.0.1:5173 com Ctrl+Shift+R e abra Ver projetos → Ver ambientes → Ver flags do projeto. Use Configurar flags para escolher o ambiente.

Próximo incremento: **4A — tokens e avaliação booleana single/batch**. Rollout parcial, SDK funcional, filtros do catálogo e consulta de auditoria ficam para suas etapas. O backend aceita somente rolloutBps 0/10000; nenhum avaliador/integração de aplicação consumidora está disponível nesta etapa.

Ambiente local atualizado: DB/API/web saudáveis, migration 000004 aplicada sem remover o volume nem criar flags demonstrativas nas contas reais. A API exige schema 4.

Verificações finais: geração sqlc/OpenAPI sem diferenças após regenerar; scan Gitleaks staged sem vazamentos; API pronta e frontend HTTP 200; flags sem sessão retornam 401. ETags canônicos também foram verificados com UUID em maiúsculas, sem conflito falso de revisão.

Aceitação manual posterior: Franklin confirmou criação, ativação em ambientes distintos e arquivamento de flags e autorizou a etapa 4A. Estado seguinte em [EVALUATION_STATUS.md](EVALUATION_STATUS.md).
