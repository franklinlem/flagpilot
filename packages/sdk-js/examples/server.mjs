import { FlagPilotClient } from '@flagpilot/sdk-js';
// Configurar no ambiente secreto do backend consumidor, nunca no bundle público.
const { FLAGPILOT_BASE_URL, FLAGPILOT_EVALUATION_TOKEN, FLAGPILOT_FLAG_KEY, FLAGPILOT_BUCKETING_KEY } = globalThis.process.env;
let flags;
try {
  flags = new FlagPilotClient({ baseUrl: FLAGPILOT_BASE_URL ?? 'http://127.0.0.1:8080/api/v1', token: FLAGPILOT_EVALUATION_TOKEN });
  const result = await flags.evaluate(FLAGPILOT_FLAG_KEY, { ...(FLAGPILOT_BUCKETING_KEY === undefined ? {} : { bucketingKey: FLAGPILOT_BUCKETING_KEY }), defaultValue: false });
  globalThis.console.log(JSON.stringify(result));
} catch {
  globalThis.console.error('Configuração inválida: confira URL, token e chave da flag no ambiente do consumidor.');
  globalThis.process.exitCode = 1;
} finally { flags?.close(); }
