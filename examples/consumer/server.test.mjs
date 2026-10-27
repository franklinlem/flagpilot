import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createConsumer } from './server.mjs';
import { FlagPilotClient } from '@flagpilot/sdk-js';

async function fixture(t, client) {
  const server = createConsumer({ client });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  t.after(() => { server.closeAllConnections(); server.close(); client.close?.(); });
  const origin = `http://127.0.0.1:${server.address().port}`;
  return { origin, post: (input, headers = {}) => fetch(origin + '/api/quote', { method: 'POST',
    headers: { Origin: origin, 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(input) }) };
}

test('cálculo server-side, contexto fixo e entradas restritas', async t => {
  const calls = [];
  let enabled = false;
  const { post, origin } = await fixture(t, { evaluate: async (...args) => { calls.push(args); return { value: enabled, reason: enabled ? 'ENABLED' : 'DISABLED' }; } });
  assert.equal((await (await post({ quantity: 3 })).json()).totalCents, 7200);
  enabled = true;
  assert.equal((await (await post({ quantity: 3 })).json()).totalCents, 6000);
  assert.equal((await (await post({ quantity: 1 })).json()).shippingCents, 800);
  assert.deepEqual(calls[0], ['shipping_v2', { bucketingKey: 'demo-customer-1', defaultValue: false }]);
  for (const input of [{ quantity: 0 }, { quantity: 1.5 }, { quantity: '3' }, { quantity: 3, bucketingKey: 'attacker' }, null]) {
    assert.equal((await post(input)).status, 400);
  }
  assert.equal((await post({ quantity: 3 }, { Origin: 'https://example.invalid' })).status, 403);
  assert.equal((await post({ quantity: 3 }, { 'Content-Type': 'text/plain' })).status, 415);
  assert.equal((await fetch(origin + '/.env')).status, 404);
  assert.equal((await post({ quantity: 'x'.repeat(2048) })).status, 413);
});

test('SDK real usa fallback padrão em erro de rede, timeout e token revogado', async t => {
  for (const transport of [async () => { throw new Error('network'); }, async () => new Response('', { status: 401 }), () => new Promise(() => {})]) {
    const client = new FlagPilotClient({ baseUrl: 'http://127.0.0.1:8080/api/v1', token: 'fp_eval_' + '1'.repeat(64),
      fetch: transport, cacheTtlMs: 0, timeoutMs: 40, maxRetries: 0 });
    const { post, origin } = await fixture(t, client);
    const response = await post({ quantity: 3 });
    assert.deepEqual(await response.json(), { quantity: 3, subtotalCents: 6000, shippingCents: 1200, totalCents: 7200, experience: 'standard', fallback: true });
    for (const path of ['/', '/app.js', '/style.css']) {
      const asset = await fetch(origin + path);
      assert.equal(asset.status, 200);
      assert.ok(!(await asset.text()).includes('fp_eval_'));
    }
  }
});
