# Organizações e membros — uso local

Depois do login, a conta mostra as organizações autorizadas. **Nova organização** cria um escopo separado e atribui ao criador o papel Proprietário. A operação não concede permissão sobre organizações existentes.

**Ver membros** abre uma URL específica da organização, mostra nome/ID e papel atual, e permite consultar membros. Apenas Proprietário vê adição, seleção/salvamento de papel e remoção. A API valida as mesmas regras independentemente da UI. Remoção exige confirmação com o usuário/escopo afetado; despromover/remover o último Proprietário ativo retorna conflito e preserva os dados.

## Cadastrar outro usuário

Sem signup público, convites por email ou senha padrão. O operador local cadastra uma conta inicialmente como Visualizador, em uma organização existente. Copie o ID da organização na tela de membros e execute na raiz:

```sh
make create-user ARGS='--email novo-usuario@example.com --org-id UUID_DA_ORGANIZACAO'
```

A senha é solicitada e confirmada sem echo. O comando imprime o UUID do usuário (não é segredo), cria a associação Viewer e a auditoria na mesma transação. Não modifica credenciais de usuário existente. Para associar uma conta já cadastrada, use **Usuário existente** no painel e selecione pelo email. Senhas seguem a política de identidade (12+ caracteres, até 256 bytes). Não compartilhar senhas no chat ou passar por argumentos.

Alternativa usando o binário da imagem, sem Go no host:

```sh
docker compose --env-file .env -f deploy/compose.dev.yml exec api /admin create-user --email novo-usuario@example.com --org-id UUID_DA_ORGANIZACAO
```

Para automação, `--password-stdin` aceita uma linha, com email/org-id explícitos. Use somente canal de entrada controlado, sem histórico de shell ou credenciais no Git. E2E usa senhas aleatórias em memória e volumes descartáveis separados; não cria usuários reais na organização de Franklin.

## Papéis e limites

| Papel na organização | Consultar membros | Adicionar/alterar/remover membros |
|---|---|---|
| Proprietário (Owner) | Sim | Sim, preservando último Owner ativo |
| Administrador (Admin) | Sim | Não |
| Editor | Sim | Não |
| Visualizador (Viewer) | Sim | Não |

Permissões de projetos, flags, tokens e auditoria consultável ainda serão implementadas nos respectivos módulos. Papéis são herdados no escopo organizacional conforme ADR-005, sem grants por projeto ou papel global. Não existe busca global de usuários/emails. Saber UUID não autoriza acesso: fora da associação retorna 404.

Revogação de membro/papel passa a valer na próxima consulta/escrita do servidor, inclusive com cookie existente. Não revogar todas as sessões de alguém por sair de uma única organização. A interface pode precisar refazer consulta após mudança feita por outro operador; uma UI desatualizada nunca supera a validação do backend. Sessão expirada retorna ao login e limpa o cache. Cache é separado por identidade/organização, sem persistência no storage do browser.

Collections usam default 25, máximo 100 e botões para carregar mais. Não reutilizar cursor em outra conta/organização; cursor inválido retorna 422. Conflito de slug/associação e último Owner retornam 409. Falha de auditoria reverte a mutação inteira; ausência de rede não dispara retry automático de escrita.

## Dados preservados

O incremento 2B usa schema 2 existente; não há migration, reset de banco ou alteração da senha/conta local de Franklin. `make dev-containers` atualiza somente os containers deste projeto e preserva o volume. Exclusão/edição de organizações, desativação global de usuários, convites e transferência formal de propriedade continuam fora deste incremento.

## Selecionar um usuário existente

O formulário de adição agora usa um seletor por email, sem digitar UUID. Aparecem usuários ativos de organizações às quais o operador tem acesso e antigos membros desta organização. Contas já associadas e desativadas são excluídas. Use **Carregar mais usuários** quando disponível. Somente Proprietários podem consultar essa lista e adicionar membros. Uma conta sem relação com suas organizações não aparece; o cadastro inicial pelo CLI continua indicando a organização correta.

Após salvar um papel, o botão fica desabilitado até haver outra alteração e o cursor volta à seta. Durante a operação aparece **Salvando…** com indicador de progresso; em caso de erro os controles são liberados e há aviso.
