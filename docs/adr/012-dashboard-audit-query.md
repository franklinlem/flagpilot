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

## Ampliação 6B — navegação e recuperação

Atalhos de conteúdo/topo, foco em títulos e títulos da aba identificam a tela sem colocar conta/segredo no document.title. Mudar filtros reposiciona o foco em elemento persistente, evitando deixá-lo no body quando o formulário é remontado. Consultas do painel combinam cancelamento do chamador/RequestInit e timeout de 8 s; um sinal externo não substitui o prazo. Token issuance usa o mesmo mecanismo comum. Não há retry automático de mutações: falha/cancelamento do transporte não garante que a gravação não aconteceu.

O harness aceita --browser chromium|firefox, padrão Chromium. Cada execução usa sua própria stack e credenciais efêmeras; a CI executa ambos separadamente para não consumir duas vezes a janela de login no mesmo banco. Não há relaxamento de rate limit nem novo segredo de servidor. Testes de navegador verificam casos de recuperação/teclado/foco; não certificam WCAG integral.
