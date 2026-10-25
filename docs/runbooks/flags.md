# Flags booleanas — uso local

Na conta, abra **Ver projetos** → **Ver ambientes** → **Ver flags do projeto**. Proprietários, Administradores e Editores podem criar e alterar flags; Visualizadores apenas consultam.

## Criar e editar

Em **Nova flag**, informe chave e descrição. A chave tem 2–80 caracteres, começa com letra minúscula e aceita letras minúsculas, números, ponto, hífen e sublinhado. Exemplo: `checkout.new`. A chave não poderá ser renomeada nem reutilizada depois de arquivada.

A flag começa **desativada em todos os ambientes**. Em **Ver flag**, altere a descrição e salve. A disponibilidade no catálogo não significa ativação nos ambientes.

## Ativar por ambiente

Volte aos ambientes e use **Configurar flags** no ambiente desejado. Os botões **Ativar flag** e **Desativar flag** alteram somente aquele ambiente. Em produção, confira o resumo com projeto, ambiente e chave e clique em **Confirmar alteração em produção**; **Cancelar** não envia a alteração. A mesma regra vale para ambientes adicionais marcados como produção com slug personalizado.

Criar um ambiente adicional inicializa nele todas as flags ainda não arquivadas, sempre desativadas, mesmo se estiverem ativadas em outro ambiente.

Se aparecer “os dados foram alterados; recarregue antes de salvar”, outra operação alterou a revisão. Use **Recarregar configurações** ou **Recarregar flag**, confira o estado atualizado e decida novamente. A aplicação não sobrescreve nem repete automaticamente uma alteração conflitante. Se uma lista paginada mudar durante a leitura, recarregue a lista para iniciar um snapshot atual.

## Arquivar

Em **Ver flag**, use **Arquivar flag** e leia o impacto antes de confirmar. O arquivamento remove a flag das configurações ativas de **todos os ambientes do projeto**, inclusive produção, e preserva a chave. A flag continua identificada como arquivada no catálogo. Não existe restauração ou exclusão física nesta etapa.

## Atualização e limites

`make dev-containers` aplica a migration aditiva 000004 e atualiza somente a stack local; a API exige schema 4. Contas, senhas, projetos e ambientes anteriores são preservados. Não executar down/reset de migrations nem apagar volumes para atualizar. Use http://127.0.0.1:5173 e Ctrl+Shift+R.

Esta etapa entrega o controle administrativo. Tokens de avaliação, endpoint de avaliação, integração de aplicações, SDK funcional e rollout percentual ainda serão implementados. Ativar uma flag aqui ainda não oferece uma API de avaliação para aplicativos consumidores.

Para clientes administrativos: obter ETag no GET e enviar seu valor exato em If-Match no PATCH/PUT; sem header retorna 428, revisão antiga retorna 412. PUT exige `enabled` e `rolloutBps` (0 ou 10000); produção e arquivamento com impacto em produção exigem `confirmProduction=true`. POST/PATCH/PUT exigem cookie de sessão, Origin permitido e CSRF. Ver contrato completo em `api/openapi.yaml`.
