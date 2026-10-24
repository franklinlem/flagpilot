# Migrations — operação inicial

Ferramenta: golang-migrate 4.20.1; SQL sequencial em `db/migrations`. A API não migra o banco. `make dev-db` e `make migrate` executam o job controlado com credencial de DDL separada. Runtime verifica exatamente a versão compatível (1 nesta entrega), além de dirty=false.

`000001_identity` cria users, organizations, memberships e sessions. Não cria usuário administrador. Auditoria será `000002`, antes das primeiras mutações administrativas; CLI bootstrap precisa aguardar a migration de auditoria antes de ser entregue.

Cada par up/down é versionado e usa transação/lock_timeout. Não editar migration compartilhada. O `.down.sql` inicial remove dados: é apenas para teste em banco descartável. Restore e rollback de imagem não se confundem com down de schema.

Se uma migration falhar:

1. Interromper a atualização; manter a versão anterior da aplicação quando compatível.
2. Inspecionar logs sem compartilhar URLs/credenciais. Consultar `SELECT version, dirty FROM schema_migrations` usando acesso autorizado.
3. Confirmar se houve rollback transacional e quais objetos realmente existem. Não supor que dirty implica mudança aplicada parcialmente nem que um rollback limpou o marcador.
4. Corrigir a causa. Usar roll-forward com migration nova para mudanças já compartilhadas; recuperação/restauração precisa de plano e backup.
5. Não executar `force` automaticamente. Somente um operador, após reconciliar schema e versão e registrar evidência, pode ajustar marcador.
6. Reexecutar validação, testes de integridade e readiness antes de considerar o banco pronto.

Testes automatizados cobrem banco vazio/up, up sem mudanças, down/up descartável, dirty/incompatibilidade, unicidade case-insensitive de email, FK, hash de sessão e proibição de DDL no runtime. Caminho de upgrade entre duas migrations será acrescentado quando existir a segunda versão.

O papel `flagpilot_app` é preexistente, criado pelo provisionamento local/init. Migrations concedem apenas permissões necessárias ao runtime. `schema_migrations.sql` dentro de storage é somente entrada de tipos para sqlc; a tabela operacional é criada exclusivamente por golang-migrate, não por esse arquivo.
