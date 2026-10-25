# SDK TypeScript/JavaScript — uso em backend

O pacote privado `@flagpilot/sdk-js` 0.1.0 está no monorepo, com builds ESM/CJS e tipos. Ambiente validado: Node 24.22.1, linha 24.x. Não há publicação no npm; o cliente não é para inclusão no navegador. Para React público, mantenha a credencial no BFF da aplicação consumidora e autorize ali cada acesso, sem expor um proxy genérico de flags.

## Testar localmente

1. Em **Ver ambientes → Configurar flags**, escolha uma flag, ative e salve seu percentual.
2. Em **Gerenciar tokens** do mesmo ambiente, emita/copie e guarde uma credencial. O segredo aparece uma vez. Use [runbook de avaliação](evaluation.md) para detalhes.
3. Na raiz do monorepo, compile o SDK e execute o exemplo abaixo. O segredo não vai ao histórico nem ao argv do processo:

```bash
npm run build --workspace @flagpilot/sdk-js
read -rsp 'Token de avaliação: ' FLAGPILOT_EVALUATION_TOKEN
printf '\n'
export FLAGPILOT_EVALUATION_TOKEN
FLAGPILOT_FLAG_KEY=flag-dev01 \
FLAGPILOT_BUCKETING_KEY=contexto-pseudonimo-123 \
node packages/sdk-js/examples/server.mjs
unset FLAGPILOT_EVALUATION_TOKEN
```

Troque `flag-dev01` pela chave real. A URL padrão é `http://127.0.0.1:8080/api/v1`; sobrescreva `FLAGPILOT_BASE_URL` para outro backend HTTPS. Essas variáveis pertencem ao **consumidor**, sem novas variáveis no `.env` do servidor FlagPilot. Não enviar segredo por chat ou Git. O exemplo imprime somente o resultado e fecha o cliente; um resultado false pode ser válido, confira também reason.

## Interpretar o resultado

| reason | Interpretação |
|---|---|
| ENABLED | Flag habilitada em 100% |
| DISABLED | Desativada ou em 0%; defaultValue não sobrepõe |
| PERCENTAGE_ROLLOUT | Valor remoto calculado para aquele contexto |
| MISSING_CONTEXT / NOT_FOUND | Default por chamada, false por padrão; revisão remota preservada |
| SDK_UNAUTHORIZED | Token inválido/expirado/revogado ou acesso recusado; cache da instância limpo |
| SDK_TIMEOUT / SDK_NETWORK_ERROR / SDK_HTTP_ERROR | Falha operacional; fallback por chamada, revisão e ambiente null |
| SDK_INVALID_RESPONSE | Resposta incompatível, inválida ou acima do limite; sem cache/retry |
| SDK_ABORTED / SDK_CLOSED / SDK_BUSY | Abort solicitado, cliente fechado ou concorrência máxima atingida |

Configuração/argumentos inválidos rejeitam com TypeError antes de enviar. Falhas operacionais retornam o fallback e não lançam erro bruto. `evaluate` preserva os motivos; `isEnabled` entrega só o boolean. Em uma aplicação, crie a instância na inicialização, reutilize e chame close no shutdown.

## Cache e revogação

O cache é exclusivo de cada instância/URL/credencial/flag/contexto, limitado a 1000 entradas LRU com TTL 30 segundos por padrão. Não há compartilhamento global, persistência ou uso de entradas expiradas como fallback. Mudança ou revogação pode levar até o TTL para aparecer em hits. Para validar imediatamente, configure cacheTtlMs: 0; não confunda cache do cliente com a revogação imediata na API. Entradas expiradas são removidas durante acesso/inserção ou close, sem timer de limpeza.

Timeout total 1500 ms inclui download e até um retry transitório. Retry-After só é aguardado se couber no orçamento; 401/403/404/422 não têm retry. Chamadas equivalentes compartilham transporte; AbortSignal de uma chamada não interrompe outros consumidores. close é idempotente e libera os recursos da instância. Limites, tipos e exemplos de import/require estão no [README do pacote](../../packages/sdk-js/README.md).

## Usar em outro projeto

```bash
npm run build --workspace @flagpilot/sdk-js
npm pack --workspace @flagpilot/sdk-js --pack-destination /tmp
```

Instale o arquivo `/tmp/flagpilot-sdk-js-0.1.0.tgz` no backend consumidor com npm install. Isso gera/usa um artefato local, sem publicar. Nome público/licença/release continuam pendentes. O cliente fornece avaliação HTTP individual; não inclui APIs administrativas, snapshot offline ou adapter público de browser.
