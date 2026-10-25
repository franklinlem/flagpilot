# Projetos e ambientes — uso local

Na conta inicial, escolha **Ver projetos** na organização desejada. Proprietário e Administrador podem usar **Novo projeto**; Editor e Visualizador consultam os projetos existentes.

Informe nome e slug (2–63 caracteres, letras minúsculas, números e hífen, começando com letra). O slug precisa ser único apenas nessa organização. Cada novo projeto recebe três ambientes separados: `development`, `staging` e `production`.

Em **Ver ambientes**, escolha **Novo ambiente** para cadastrar ambientes adicionais. O slug precisa ser único no projeto. Marque **Este ambiente é de produção** para ambientes que atendam usuários reais, inclusive quando o slug for personalizado. A marcação fica destacada na lista; o slug `production` exige essa classificação.

Nome, slug e classificação não podem ser editados neste incremento. Também não há remoção/arquivamento de projetos ou ambientes. Isso evita exclusões ou reclassificações sem uma política definida. Flags e configurações por ambiente serão o próximo módulo; a criação de ambiente ainda não publica funcionalidades.

Falhas de validação não enviam o formulário; duplicidade retorna conflito, preservando os valores para correção. Sessão expirada retorna ao login. Mutações não são repetidas automaticamente em falha de rede. Listas paginadas oferecem **Carregar mais projetos/ambientes** quando necessário.

## Atualização local

Execute `make dev-containers` na raiz para aplicar migrations e atualizar os containers próprios. A migration `000003_projects_environments` acrescenta tabelas, sem modificar usuários, senhas ou associações existentes. A API passa a exigir schema 3. O volume permanece preservado. Atualize http://127.0.0.1:5173 com Ctrl+Shift+R.

Não executar migration down/reset nem apagar o volume para atualizar. Testes usam bancos separados e descartáveis. Não é necessário recadastrar a conta existente.

## API disponível

- GET/POST `/api/v1/organizations/{orgId}/projects`
- GET `/api/v1/projects/{projectId}`
- GET/POST `/api/v1/projects/{projectId}/environments`

POST exige sessão, Origin permitido e CSRF. Coleções aceitam limit 1–100 (padrão 25) e cursor vinculado à identidade e ao escopo. Ambiente exige booleano `isProduction`; omissão não assume silenciosamente que o ambiente seja de teste. Consulte `api/openapi.yaml` para o contrato completo.
