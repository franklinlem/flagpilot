# ADR-012 — Filtros de catálogo e consulta de auditoria

Data: 25/10/2026. Estado: implementada localmente em 6A.

## Decisão

O catálogo aplica filtros no PostgreSQL antes de LIMIT: q é uma substring literal da chave (trim/minúsculas, até 80 caracteres de chave); status é all/active/archived, referindo-se ao arquivamento no projeto, independente do enabled de cada ambiente. Sem busca por descrição nem curingas SQL neste incremento. A paginação UUID existente inclui agora uma impressão dos filtros no escopo, além de ator/projeto. Clientes com cursor anterior devem reiniciar a listagem.

Auditoria usa módulo próprio de consulta, GET /organizations/{orgId}/audit-events, sessão administrativa e permissão audit.read para Owner/Admin/Editor/Viewer. Token de avaliação não autoriza histórico. A leitura bloqueia organização SHARE e relê associação/usuário ativo após aguardar, protegendo contra remoção concorrente. Não há endpoint de escrita, limpeza ou restauração do histórico.

Ordenação created_at DESC,id DESC aproveita o índice existente. Cursor contém a âncora UUID; sua data é recuperada somente dentro da organização autorizada e usada no predicado de tupla. Empates de horário não repetem eventos. Escopo vinculado a ator/organização e hash dos filtros action/resourceType exatos; limites 25 padrão/100 máximo/512 caracteres de cursor. Cursores opacos não são credenciais e não substituem a autorização SQL. Novos eventos mais recentes são vistos ao atualizar a lista; não prometemos um snapshot histórico congelado entre requests.

A API projeta antes/depois por lista explícita de campos conhecidos para cada tipo, aceitando somente escalares/null. Objetos aninhados, documentos inválidos, tipos desconhecidos e campos como password/hash/secret/salt/context são omitidos; não se devolve o JSON bruto do escritor. Isso é defesa adicional à redação existente nas gravações. Descrições e nomes continuam conteúdo fornecido por usuários, exibido pelo React como texto; não se promete identificar/remover segredos digitados intencionalmente nesses campos. Auditoria tem ator por UUID e rótulo user/local-operator/unknown; não reconstruímos nome/email históricos.

UI aplica filtros explicitamente, preserva filtros na URL e isola cache por usuário/escopo/filtros. Detalhes usam details/summary com Antes/Depois e porcentagens legíveis. Foco no título ao entrar, labels, status de carregamento/resultado, erros recuperáveis e mesma leitura para Viewer. Formulário de filtros permanece acessível quando a consulta falha, permitindo limpar parâmetros inválidos. Busca filtra todo o catálogo, não apenas páginas baixadas.

## Consequências e limites

Sem nova migration/dependência/variável: schema 5 e privilégios existentes SELECT/INSERT da auditoria permanecem. Consulta de histórico é somente leitura e não gera evento por si. Filtrar um catálogo grande por substring ou histórico por ação/tipo pode exigir índice adicional após medição; não alegamos benchmark/SLO.

A cobertura atual de teclado/mobile/labels no Chromium não certifica WCAG integral nem compatibilidade Firefox. Retenção automática, exportação, snapshots restauráveis, busca por descrição, identidade nominal histórica e métricas operacionais ficam para incrementos específicos. Alterações externas continuam sem autorização.
