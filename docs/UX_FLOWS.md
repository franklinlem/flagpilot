# Fluxos iniciais de UX

Estado em 25/10/2026: fluxos de autenticação, organização/projetos/ambientes/flags, rollout e tokens disponíveis. Em 6A, catálogo com busca por chave/estado e histórico organizacional implementados. Wireframe abaixo continua uma direção, sem afirmar navegação lateral/seletor global existentes.

1. Login → selecionar organização → projetos → ambientes → flags. Cabeçalho mantém organização/projeto/ambiente identificados por texto, sem depender só de cor.
2. Lista de flags: filtro por chave e estado no catálogo (descrição permanece pendente); detalhe exibe configuração do ambiente e revisão. Chave é imutável após criação.
3. Alterar flag: formulário → resumo de impacto/confirmar em produção → salvar com If-Match → sucesso e auditoria; em 412, mostrar estado atual e oferecer reedição sem retry automático de escrita.
4. Tokens: criar → valor exibido uma vez com botão copiar → ocultar/fechar sem persistência; lista posterior exibe só nome/prefixo/expiração/revogação. Segredo nunca reaparece.
5. Organização: membros e papéis conforme Owner; impedir último Owner removido. Viewer lê histórico sem receber controles de edição, mas a API ainda deve negar mutações.

Wireframe textual: topo com marca e seletor de organização; lateral com projetos/flags/tokens/auditoria; conteúdo com título do projeto, badge persistente de ambiente, filtro e lista; formulário em rota própria com foco no título. Em mobile, navegação recolhível com labels acessíveis e mesmas confirmações. Estados vazio/erro/expiração/conflito fazem parte de cada tela; teclado e contraste verificados junto da implementação.

Histórico atual: conta/projetos → Ver auditoria → filtrar ação/tipo → Ver alterações → Antes/Depois; atualizar ou paginar sem escrever. Filtros ficam na URL. Labels, foco no título, abertura por teclado e largura mobile foram exercitados no Chromium; WCAG integral e Firefox permanecem pendentes.
