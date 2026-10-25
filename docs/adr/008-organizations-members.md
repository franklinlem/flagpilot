# ADR-008 — Organizações, membros e concorrência

Data: 24/10/2026. Estado: aceita e implementada no incremento 2B, conforme continuação do plano autorizada por Franklin.

## Contexto

A identidade já existe; gestão de membros precisa preservar escopo e último Owner sem credenciais padrão ou signup público. Em READ COMMITTED, o papel observado antes de aguardar um lock pode ficar desatualizado. O CLI também precisa permitir provisionar usuários além do bootstrap único.

## Decisão

- Qualquer usuário autenticado ativo pode criar organização própria e passa a Owner **somente dessa organização**. Listagem e detalhe são filtrados pela associação autenticada; não existe papel global de aplicação.
- Owner gerencia membros; Owner/Admin/Editor/Viewer leem membros do próprio escopo. Papel/permissão desconhecidos negados. 401 sem sessão, 404 fora do escopo e 403 para privilégio insuficiente no escopo autorizado.
- Mutações adquirem `FOR NO KEY UPDATE OF organizations`, depois **releem** a associação/papel em outro statement, validam e gravam a alteração + auditoria na mesma transação. Leituras de detalhe/membros usam `FOR SHARE` e releitura equivalente. Não confiar na lista de papéis da sessão recebida previamente.
- NO KEY UPDATE serializa mutações de membros e é compatível com KEY SHARE de FKs de auditoria; evita ciclo com login/recuperação que bloqueiam usuário e inserem auditoria organizacional. Não há alterações de chave nem exclusão de organizações nessa operação.
- Impedir remover/despromover o último Owner **ativo**, contando usuários não desativados no mesmo lock. Duas autodespromoções concorrentes deixam ao menos um Owner. Operador SQL privilegiado não é restringido por essa regra de aplicação; eventual desativação global pelo produto precisará respeitar as organizações afetadas.
- Remover associação não apaga usuário nem revoga sessões globais. O acesso ao escopo removido é negado na próxima operação; outras organizações permanecem acessíveis.
- Paginação por UUID estável, default 25/máximo 100, cursor opaco vinculado à coleção/ator/organização. Cursor não é credencial nem prova de autorização; toda consulta continua filtrada por escopo.
- CLI `create-user --org-id` cria usuário e associação Viewer com auditoria atômica, sem endpoint HTTP de cadastro. Nome/email global não é pesquisável pela API. Owner associa usuários existentes; pode promover papel pelo painel. Usuário existente nunca tem senha alterada por esse comando.
- React Router para URLs de membros; TanStack Query com chaves incluindo usuário e organização, abort de queries e limpeza no logout/troca de identidade. Formulários são recriados ao trocar a rota organizacional. Mutações sem retry automático; UI não concede autorização.

## Alternativas

Um COUNT de Owners sem serialização permite duas remoções concorrentes. Consultar papel apenas no middleware permite aplicar permissão revogada enquanto a operação aguarda. FOR UPDATE no pai conflita com KEY SHARE usado por auditoria/FKs. Um diretório público/global de emails ampliaria enumeração e exposição entre organizações. Convites por email e cadastro web seguem fora deste MVP.

## Consequências

Não há migration nova: schema 2 já suporta 2B. Locks curtos limitados pelo timeout DB da aplicação; uma organização de cada vez. Atualizações de membro usam serialização com último escritor autorizado, sem ETag; revisão de flags continua reservada ao módulo de flags. Não entregar edição/exclusão de organização ou desativação global de usuários implicitamente.

Referências: [PostgreSQL 16 — locks](https://www.postgresql.org/docs/16/explicit-locking.html), [OWASP Authorization](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html), [TanStack query keys](https://tanstack.com/query/latest/docs/framework/react/guides/query-keys).

## Ajuste após uso manual — seleção de usuários

Franklin solicitou substituir o UUID digitado por um seletor de contas existentes. `GET /organizations/{orgId}/member-candidates` retorna somente userId/email de usuários ativos ainda não associados ao destino. Exige Owner no destino e aplica a mesma releitura do papel depois do lock de leitura.

Candidatos precisam compartilhar uma organização com o ator **ou** constar em eventos de associação anteriores do destino (`member.added`, `member.removed`, `member.role_changed`). Assim é possível selecionar colegas conhecidos entre organizações e reincluir um membro removido, mesmo que tenha perdido sua única associação. Somente o histórico do destino participa; a consulta não fornece um diretório de todos os usuários da instalação. Não revela nomes/papéis das organizações de origem. Usuários totalmente externos continuam fora do seletor. O endpoint de associação por UUID permanece compatível, com autorização Owner existente.

Paginação de 25/máximo 100, cursor ligado a coleção/ator/destino; botão para carregar mais usuários. A elegibilidade e os dados são consultados novamente no servidor; valores em cache não concedem autorização. Histórico não concede acesso por si só: somente Owner atual do destino pode listar candidatos. A exclusão futura de auditoria precisará considerar a disponibilidade de antigos membros no seletor; não existe limpeza automática nesta etapa.

Botões desabilitados por ausência de alterações usam cursor padrão; `aria-busy` e cursor de progresso ficam restritos às operações em andamento. A causa do carregamento aparente era `button:disabled { cursor: wait }`, inclusive após persistir com sucesso. Regressão medida no Chromium com o CSS real, além do fluxo de alteração/reinclusão.
