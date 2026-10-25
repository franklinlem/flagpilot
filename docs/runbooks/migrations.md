# Migrations — operação inicial

Ferramenta: golang-migrate 4.20.1; SQL sequencial em `db/migrations`. A API não migra o banco. `make dev-db` e `make migrate` executam o job controlado com credencial de DDL separada. Runtime verifica exatamente a versão compatível (5 no incremento de tokens/avaliação), além de dirty=false.

`000001_identity` cria users, organizations, memberships e sessions. Não cria usuário administrador. `000002_audit` cria eventos append-only para o runtime; bootstrap e recuperação já usam auditoria transacional. `000003_projects_environments` acrescenta projetos/ambientes e classificação de produção. `000004_boolean_flags` acrescenta flags/configurações e privilégios para revisões, preservando projetos/ambientes existentes. `000005_evaluation_tokens` acrescenta credenciais por ambiente, sem alterar flags/configurações existentes. Não editar migrations já aplicadas.

Cada par up/down é versionado e usa transação/lock_timeout. Não editar migration compartilhada. O `.down.sql` inicial remove dados: é apenas para teste em banco descartável. Restore e rollback de imagem não se confundem com down de schema.

Se uma migration falhar:

1. Interromper a atualização; manter a versão anterior da aplicação quando compatível.
2. Inspecionar logs sem compartilhar URLs/credenciais. Consultar `SELECT version, dirty FROM schema_migrations` usando acesso autorizado.
3. Confirmar se houve rollback transacional e quais objetos realmente existem. Não supor que dirty implica mudança aplicada parcialmente nem que um rollback limpou o marcador.
4. Corrigir a causa. Usar roll-forward com migration nova para mudanças já compartilhadas; recuperação/restauração precisa de plano e backup.
5. Não executar `force` automaticamente. Somente um operador, após reconciliar schema e versão e registrar evidência, pode ajustar marcador.
6. Reexecutar validação, testes de integridade e readiness antes de considerar o banco pronto.

Testes automatizados cobrem banco vazio/up, up sem mudanças, down/up descartável, dirty/incompatibilidade, unicidade case-insensitive de email, FK, hash de sessão e proibição de DDL no runtime. Os upgrades 2→3→4→5 são verificados em banco real com identidade, projeto/ambiente e flags/configurações existentes preservados.

O papel `flagpilot_app` é preexistente, criado pelo provisionamento local/init. Migrations concedem apenas permissões necessárias ao runtime. `schema_migrations.sql` dentro de storage é somente entrada de tipos para sqlc; a tabela operacional é criada exclusivamente por golang-migrate, não por esse arquivo.

Prontidão do container PostgreSQL usa `pg_isready -h 127.0.0.1` para esperar o servidor final TCP, não o servidor temporário socket-only usado pelo entrypoint durante init. Em falha de smoke, diagnóstico sanitizado é coletado antes de remover o banco efêmero.
