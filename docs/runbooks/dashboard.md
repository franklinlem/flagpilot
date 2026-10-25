# Dashboard — filtros e histórico

Atualize http://127.0.0.1:5173 com Ctrl+Shift+R após atualizar os containers.

## Filtrar flags

Abra **Ver projetos → Ver ambientes → Ver flags do projeto**. Em **Buscar chave**, informe um trecho da chave, como `checkout`, e selecione **Estado no catálogo**: Todas, Ativas no catálogo ou Arquivadas. Clique **Aplicar filtros**. Busca aceita letras/números/ponto/hífen/sublinhado, normaliza espaços externos e maiúsculas, e não usa curingas `%`/`*`. **Limpar filtros** volta ao catálogo completo.

O filtro é aplicado no banco antes de paginar. **Carregar mais flags** continua o mesmo filtro; alterar os filtros reinicia a lista. URL preserva filtros ao recarregar ou voltar/avançar no navegador. O número mostrado conta os itens já carregados, sem simular um total global.

Ativa no catálogo significa que a flag não foi arquivada. Ela ainda pode estar desativada ou com 0% em um ambiente. Ativação/rollout continuam em **Configurar flags**, com confirmação em produção e controle de revisão.

## Consultar auditoria

Na organização, abra **Ver auditoria**, disponível na conta e na lista de projetos. Todos os papéis da organização podem consultar. O histórico mostra ação, horário local do navegador, tipo/ID do recurso e ator por UUID ou Operador local. O registro temporal é armazenado em UTC.

Use **Ação** e **Tipo de recurso**, depois **Aplicar filtros**; os dois filtros são combinados. Por exemplo, `flag.config_updated` com `flag_config` mostra mudanças de ativação/rollout. **Ver alterações** abre **Antes/Depois**; Tab e Enter funcionam no controle nativo. Percentual é mostrado em %, e nenhum JSON arbitrário do banco é entregue à tela. Metadados conhecidos de tokens podem aparecer, sem o segredo/hash.

**Carregar mais eventos** busca registros mais antigos. **Atualizar histórico** consulta os eventos recentes e atualiza as páginas carregadas. Ler histórico não cria outro evento. Associação removida/conta desativada perde acesso; não há acesso por token de avaliação. Campos desconhecidos ou detalhes não registrados aparecem como indisponíveis, não como um snapshot restaurável.

## Diagnóstico e limites

- Lista vazia pode ser resultado legítimo dos filtros; limpe para conferir o conjunto completo.
- 422 indica parâmetros/cursor inválidos. Recarregue/reinicie a listagem após mudar filtros; não reutilize cursor de outro usuário/organização/projeto/filtro. Cursores anteriores à etapa 6A do catálogo devem ser reiniciados.
- 401 pede novo login; 404 indica organização/recurso ausente ou fora da associação autorizada. Erro de serviço permite tentar novamente.
- Não há restauração de versões, exportação, exclusão/retenção automática ou filtros por descrição/data neste incremento. Auditoria identifica contas por ID, sem prometer email/nome histórico.

`make dev-containers` reconstrói a aplicação sem apagar volumes; não há migration nova (schema 5). `make ci` e `python3 scripts/auth-e2e.py` usam banco/stack de testes descartáveis. Nenhum deploy externo é necessário para usar essas telas localmente.
