# ADR-010 — Flags booleanas e revisões

Data: 24/10/2026. Estado: implementada no incremento 3B, conforme continuidade autorizada por Franklin.

## Decisão e escopo

- Módulo `internal/flags` oferece criação/listagem/detalhe, alteração de descrição, arquivamento e configuração booleana por ambiente. Chave imutável (`[a-z][a-z0-9_.-]{1,79}`), descrição até 2000 caracteres; chave arquivada permanece reservada. Não há exclusão nem restauração neste incremento. O catálogo inclui arquivadas, identificadas textualmente; configurações excluem arquivadas.
- Owner/Admin/Editor escrevem; Viewer lê. Toda operação valida associação ativa no banco e segura SHARE na organização após releitura do papel. Fora do escopo retorna 404; privilégio insuficiente, 403. `projects.Authorize` compartilha essa fronteira sem duplicar política de autorização.
- Migration 000004 acrescenta flags e configurações, com FKs compostas por projeto. Runtime não pode alterar chave, projeto, salt ou excluir essas tabelas. São concedidos apenas SELECT/INSERT e UPDATE nas colunas mutáveis; revisão ambiental passa a ser atualizável. Readiness exige schema 4.
- Configuração inicial: enabled=false, rollout_bps=10000, version=1, salt hexadecimal aleatório de 32 bytes, persistente. O salt não é retornado pela API administrativa nem incluído em auditoria. O booleano habilitado admite 0/10000 na API; percentuais intermediários ficam para 4B, embora o banco já prepare o intervalo.
- Criar flag bloqueia organização → projeto → ambientes em ordem UUID, inicializa todos os ambientes e incrementa cada revisão uma vez. Criar ambiente usa organização → projeto e inicializa todas as flags ativas desativadas na mesma transação; o snapshot inicial desse novo ambiente tem revisão zero. `internal/flagstate` compartilha a inicialização sem dependência circular entre os domínios. Corrida entre criação de flag/ambiente não deixa lacunas.
- Configuração bloqueia organização → apenas seu ambiente, sem bloquear projeto. Consulta a flag ativa **depois** de adquirir esse lock, evitando gravar configuração com estado anterior a um arquivamento. Incrementa versão individual e revisão ambiental uma vez por gravação aceita, inclusive se o valor enviado for igual. Alteração de staging não altera produção.
- Arquivamento bloqueia projeto e todos os ambientes em ordem UUID; atualiza archived_at/versão da flag e todas as revisões. Configurações são preservadas fisicamente, mas ficam inacessíveis para configuração ativa (404). O avaliador futuro também precisará excluir flags arquivadas. Não é uma exclusão física nem promete restauração histórica.
- Alteração de descrição usa versão própria da flag. Não muda revisão ambiental, pois snapshots de configuração incluem chave/estado/revisões, sem descrição mutável. Metadados e configurações são recursos distintos.

## Precondições e produção

- GET/PATCH/POST de flag retornam ETag `"flag:<UUID>:<version>"`. PATCH exige o valor exato lido anteriormente.
- GET/PUT de configuração e GET de snapshot retornam ETag `"env:<UUID>:<revision>:<role>"`. Papel herdado faz parte da representação contextual do snapshot; incluí-lo evita reutilizar o mesmo ETag se o papel muda. A UI guarda o ETag recebido; não o calcula a partir de um contador JavaScript.
- If-Match ausente retorna 428; valor diferente/antigo/fraco, lista ou wildcard retorna 412. Não admitir `*` como forma de sobrescrever às cegas. Contrato aceita um único ETag forte exato; a UI pede recarregamento e revisão humana, sem repetir escrita automaticamente.
- PUT em ambiente classificado como produção exige `confirmProduction=true`, inclusive para slug personalizado. Arquivamento exige confirmação se o projeto contém ambiente de produção. Backend consulta o booleano persistido; nome/slug não decidem permissão. Confirmação é proteção contra acidente, não substitui RBAC, CSRF ou If-Match.
- A UI apresenta resumo da flag/projeto/ambiente e botões Confirmar/Cancelar antes da escrita em produção. Arquivamento avisa sobre todos os ambientes e a reserva permanente da chave. Criar flag desativada não exige confirmação de ativação em produção.

## Paginação, auditoria e limites

Paginação por UUID: default 25/máximo 100. Cursor de configuração liga coleção, ator, ambiente e revisão; mudança durante paginação retorna 422 e exige recarregar a coleção, sem misturar silenciosamente snapshots. Leituras mantêm SHARE no ambiente até concluir a representação. Sessão/Origin/CSRF e Cache-Control no-store são reutilizados.

Criação, edição, arquivamento e escrita de configuração registram eventos antes/depois na mesma transação dos dados e revisões. Falha de auditoria reverte tudo, inclusive inicializações. Não há endpoint de consulta de auditoria nesta etapa. JSON estrito, sem campos desconhecidos, até 32 KiB para flags (comporta descrições Unicode); os limites anteriores de outros módulos permanecem.

Ainda fora do incremento: filtros de catálogo, rollout parcial, tokens, avaliação single/batch, SDK funcional, restauração de flags e histórico restaurável. Próxima etapa 4A: tokens e avaliação booleana. Testes automatizados não substituem a aceitação visual/manual de Franklin.
