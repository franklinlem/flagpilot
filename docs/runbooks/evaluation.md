# Tokens e avaliação — uso local

Na conta, abra **Ver projetos → Ver ambientes → Gerenciar tokens** no ambiente desejado. Apenas Proprietários e Administradores podem administrar tokens. **Novo token** solicita nome e validade; o padrão sugerido é 90 dias, com opções entre 1 e 365 dias.

Após **Emitir token**, copie e guarde o segredo no backend consumidor. Ele é exibido somente nessa resposta. **Já guardei, ocultar segredo**, fechar a emissão ou navegar para outra página encerra a exibição. Não é possível consultar o segredo novamente; se perder, revogue a credencial pela lista e emita outra. Não colocar em Git, bundle público, localStorage, URL ou mensagens.

A listagem mostra nome, prefixo, prazo e estado. **Revogar token** pede confirmação e interrompe novas avaliações com aquela credencial. Aplicações já autenticadas em uma requisição podem terminar a chamada em voo. O token pertence ao ambiente: remover a conta de quem o emitiu não o revoga automaticamente.

## Avaliar uma flag com curl

Use uma flag do projeto associado ao token e configure seu estado em **Configurar flags**. O ambiente será sempre o do token; não envie identificador de ambiente/projeto no corpo. Para testar no terminal sem colocar o segredo no histórico nem nos argumentos do curl:

```bash
read -rsp 'Token de avaliação: ' FLAGPILOT_EVALUATION_TOKEN
printf '\n'
printf 'Authorization: Bearer %s\n' "$FLAGPILOT_EVALUATION_TOKEN" |
  curl --silent --show-error --fail-with-body \
    --header @- --header 'Content-Type: application/json' \
    --data '{"flagKey":"chave01"}' \
    http://127.0.0.1:8080/api/v1/evaluate
unset FLAGPILOT_EVALUATION_TOKEN
```

Substitua `chave01` pela chave desejada. A resposta informa `flagKey`, `value`, `reason`, `environment` (slug) e `revision`. Flag habilitada com 10000 retorna true/ENABLED; desativada ou 0 retorna false/DISABLED; ausente/arquivada retorna false/NOT_FOUND. Uma flag em outro ambiente não é consultada por trocar dados do corpo.

## Configurar rollout percentual

Abra **Ver ambientes → Configurar flags**. O campo **Porcentagem de <chave>** aceita 0 a 100, com até duas casas decimais (por exemplo, `37,25` ou `37.25`). **Salvar porcentagem** preserva o estado de ativação; **Ativar/Desativar flag** preserva o percentual. Flag desativada ou percentual zero não libera nenhum contexto. 100% habilitado dispensa contexto. Em produção, revise o resumo de estado/porcentagem antes de confirmar. Em conflito de revisão, recarregue e reaplique sua intenção.

Para parcial habilitado, troque o corpo do curl acima por:

```json
{"flagKey":"chave01","context":{"bucketingKey":"contexto-pseudonimo-123"}}
```

A resposta usa `PERCENTAGE_ROLLOUT`, com `value` true ou false conforme o bucket. Sem bucketingKey, retorna false/`MISSING_CONTEXT`; chave vazia é inválida. Use uma chave estável e pseudônima definida pelo backend consumidor, até 256 bytes UTF-8. Não use email nem dados sensíveis. A mesma chave continua no mesmo bucket naquela configuração, mesmo após reinício ou mudança de percentual. Projetos, ambientes, flags e salts diferentes produzem distribuições independentes. Não há normalização Unicode: preserve exatamente a codificação da chave.

A distribuição é aproximada, não uma contagem exata de usuários. Salt é gerado para cada configuração e preservado nas edições; aumentar a porcentagem mantém os contextos já incluídos. Alterar a bucketingKey também altera o grupo. Flags continuam sem substituir a autorização da aplicação consumidora.

## Avaliar um lote

Leia o segredo novamente com `read -rsp` e envie o header por stdin como acima, trocando o corpo/endpoint por:

```json
{"evaluations":[{"flagKey":"chave01"},{"flagKey":"outra_flag"}]}
```

Endpoint: POST `/api/v1/evaluate/batch`. O retorno contém `results` na mesma ordem, incluindo duplicatas, e `revision`; todos os itens usam o mesmo snapshot e revisão. Máximo 100 solicitações, JSON até 64 KiB. Contexto opcional aceita somente bucketingKey de até 256 bytes UTF-8, sem persistência. Em parcial habilitado, informe essa chave em cada item que precisar do bucketing.

## Erros e integração

| Código | Significado |
|---|---|
| 400/422 | Corpo/campos/tamanho de lote/contexto inválidos; corrigir antes de repetir |
| 401 | Bearer ausente, inválido, expirado ou revogado; corrigir a credencial |
| 403 | Acesso direto de browser recusado ou papel sem gestão de tokens |
| 413/415 | Corpo acima de 64 KiB ou Content-Type diferente de application/json |
| 429 | Aguardar Retry-After; limite por segundo global/peer/credencial |
| 503 | Banco indisponível ou configuração não suportada; consumidor deve tratar falha e aplicar fallback seguro |

Sessão do painel não autoriza avaliação, e token de avaliação não administra flags nem emite outros tokens. A API rejeita Origin/Sec-Fetch-Site; aplicações frontend devem usar um backend/BFF que guarde o segredo e autorize seus próprios usuários e flags. Não construir um proxy público irrestrito. Fora do loopback, usar HTTPS.

Limites por segundo nesta etapa: 600 globais, 120 por peer real e 100 por credencial, em uma réplica. Peers compartilhados pelo proxy compartilham o limite; X-Forwarded-For não é aceito como identidade confiável automaticamente. Ainda não há benchmark ou SLO de capacidade.

## Atualização

`make dev-containers` aplica migration 000005 e atualiza a stack local, sem apagar volumes ou dados existentes. Readiness exige schema 5. Atualize http://127.0.0.1:5173 com Ctrl+Shift+R. Nenhum token real é criado automaticamente pelo desenvolvimento; cada emissão pelo painel é explícita e auditada.

O SDK TypeScript/JavaScript privado para backend está disponível com cache, timeout e fallback. Veja [uso do SDK](sdk.md). Integração HTTP direta e rollout percentual também estão disponíveis. Não armazenar token consumidor no `.env` do servidor FlagPilot; configure-o no ambiente secreto do backend que fará as consultas.
