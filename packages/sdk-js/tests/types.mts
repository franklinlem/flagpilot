import { FlagPilotClient, type EvaluationResult } from '@flagpilot/sdk-js';
const flags = new FlagPilotClient({ baseUrl: 'http://127.0.0.1:8080/api/v1', token: 'fp_eval_' + 'a'.repeat(64) });
const value: Promise<boolean> = flags.isEnabled('checkout', { defaultValue: false });
const result: Promise<EvaluationResult> = flags.evaluate('checkout');
void value; void result;
// @ts-expect-error defaultValue exige boolean
flags.evaluate('checkout', { defaultValue: 'yes' });
// @ts-expect-error token obrigatório no cliente de servidor
new FlagPilotClient({ baseUrl: 'http://127.0.0.1:8080/api/v1' });
flags.close();
