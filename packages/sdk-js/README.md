# @flagpilot/sdk-js — 0.1.0 privado

Cliente HTTP para backend Node 24.22.1 (matriz atual: 24.x), com TypeScript, ESM e CJS. Pacote privado do monorepo, ainda sem publicação no npm ou licença atribuída. Não fornece avaliação offline nem token publicável. Browser usa um BFF da aplicação consumidora que autorize seus usuários/flags; o construtor recusa um ambiente com window.document.

```ts
import { FlagPilotClient } from '@flagpilot/sdk-js';

const flags = new FlagPilotClient({
  baseUrl: 'http://127.0.0.1:8080/api/v1',
  token: process.env.FLAGPILOT_EVALUATION_TOKEN!,
});
try {
  const enabled = await flags.isEnabled('new_checkout', {
    bucketingKey: 'contexto-pseudonimo-123',
    defaultValue: false,
  });
  // A autorização da aplicação continua independente de enabled.
  console.log(enabled);
} finally { flags.close(); }
```

CJS: `const { FlagPilotClient } = require('@flagpilot/sdk-js');`. Reutilize uma instância no ciclo de vida do backend; feche no shutdown. O exemplo abre/fecha somente para demonstrar liberação de recursos.

## Contrato

- `evaluate(key, { bucketingKey?, defaultValue?, signal? }): Promise<EvaluationResult>` retorna valor, motivo, ambiente e revisão. `isEnabled` retorna somente boolean. Argumentos inválidos rejeitam com TypeError e não enviam requests.
- Respostas remotas válidas preservam reason/environment/revision. Default por chamada substitui valor somente em MISSING_CONTEXT/NOT_FOUND e em falhas locais; DISABLED válido continua false. PERCENTAGE_ROLLOUT mantém o valor remoto.
- Falhas operacionais retornam fallback false por padrão, com `environment: null`, `revision: null` e motivo SDK_TIMEOUT, SDK_ABORTED, SDK_NETWORK_ERROR, SDK_HTTP_ERROR, SDK_UNAUTHORIZED, SDK_INVALID_RESPONSE, SDK_BUSY ou SDK_CLOSED. `SDK_UNAUTHORIZED` cobre HTTP 401/403. Não há erro bruto, corpo HTTP ou segredo no resultado.
- Chave/contexto seguem os limites da API (contexto de 1 a 256 bytes UTF-8, sem NUL ou surrogates inválidos). Não enviar email ou dados sensíveis. UTF-8 é preservado sem normalização Unicode.
- URL absoluta termina em /api/v1 (prefixo de proxy permitido); HTTPS obrigatório fora de localhost/127.0.0.1/[::1]. Sem credenciais na URL, query ou fragmento. Transporte não segue redirects nem envia cookies. Respostas JSON de até 64 KiB são validadas antes de uso/cache.

## Opções e limites

| Opção | Padrão | Limites |
|---|---|---|
| timeoutMs | 1500 | 1–30000; orçamento total, incluindo body, espera e retry |
| cacheTtlMs | 30000 | 0–300000; zero desabilita cache |
| maxCacheEntries | 1000 | 0–10000; zero desabilita cache |
| maxConcurrentRequests | 100 | 1–1000 requests distintos em voo |
| maxRetries | 1 | 0 ou 1 |
| fetch | fetch nativo | Transporte injetável confiável; não usar para logar headers/body |

Cache LRU limitado, somente respostas remotas válidas, por instância com URL/credencial fixas, flag e contexto. Default por chamada não entra no cache. TTL invalida a leitura; entradas vencidas são removidas no acesso/inserção ou close, sem timer de limpeza. A revisão recebida pertence ao snapshot remoto e não assegura atualidade durante o TTL. Descrição do hash em ADR-003; o cliente usa avaliação remota, sem baixar salts/configurações.

Mudanças e revogação podem demorar até o TTL para aparecer em hits do cache. Para verificar imediatamente, use cacheTtlMs: 0 ou uma nova instância. Ao observar 401/403, a instância limpa seu cache e impede que respostas anteriores em voo o repopulem. Falhas não são cacheadas e não usam valor expirado como fallback.

Requests equivalentes compartilham uma chamada, inclusive com defaults distintos. Abort de um consumidor não interrompe os demais; cancelar o último aborta o transporte e libera o slot. Chamadas tardias compartilham o orçamento restante da operação existente. close é idempotente, aborta pendências, limpa cache/contextos retidos e faz chamadas futuras retornarem SDK_CLOSED. O contexto fica em memória durante requests/cache; não há gravação em disco, storage do navegador, logs ou telemetria. Não há promessa de apagar strings da memória do runtime.

Até um retry para falha de rede ou HTTP 408/429/500/502/503/504, com jitter. Retry-After (segundos ou data HTTP) só é aguardado se couber no orçamento total; não há retry de 401/403/404/422, resposta inválida ou abort. Um fallback true é decisão explícita da aplicação e deve ser usado com cuidado.

## Uso local

Na raiz: `npm run build --workspace @flagpilot/sdk-js`. O workspace permite importar o pacote a partir do exemplo `packages/sdk-js/examples/server.mjs`. Para outro projeto, gere um tarball local com `npm pack --workspace @flagpilot/sdk-js --pack-destination /tmp` e instale o arquivo produzido; isso não publica o pacote. Licença/nome público continuam pendentes.

Ver [runbook do SDK](../../docs/runbooks/sdk.md) para configuração segura, exemplo executável e aceitação local. `npm run test --workspace @flagpilot/sdk-js` verifica contrato, falhas, cache/concorrência, tipos ESM/CJS e vetores Go/JS. Não é cliente administrativo e não administra tokens, flags ou sessões.
