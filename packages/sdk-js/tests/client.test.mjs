import { test } from 'node:test';
import assert from 'node:assert/strict';
import { setTimeout as sleep } from 'node:timers/promises';
import { createRequire } from 'node:module';
import { FlagPilotClient } from '../dist/index.js';
const token = 'fp_eval_' + 'a'.repeat(64);
const baseUrl = 'http://127.0.0.1:8080/api/v1';
const remote = (flagKey = 'checkout', extra = {}) => ({ flagKey, value: true, reason: 'ENABLED', environment: 'development', revision: 7, ...extra });
const json = data => new globalThis.Response(JSON.stringify(data), { headers: { 'Content-Type': 'application/json' } });
const client = (fetch, extra = {}) => new FlagPilotClient({ baseUrl, token, fetch, maxRetries: 0, ...extra });
const deferred = () => { let resolve; const promise = new Promise(done => { resolve = done; }); return { promise, resolve }; };

test('ESM/CJS exportam cliente, headers seguros e contrato HTTP validado', async () => {
  assert.equal(typeof createRequire(import.meta.url)('../dist/cjs/index.js').FlagPilotClient, 'function');
  const sdk = client(async (url, init) => {
    assert.equal(url, baseUrl + '/evaluate'); assert.equal(init.headers.Authorization === `Bearer ${token}`, true);
    assert.equal(init.credentials, 'omit'); assert.equal(init.redirect, 'error'); assert.equal(init.cache, 'no-store');
    assert.deepEqual(JSON.parse(init.body), { flagKey: 'checkout', context: { bucketingKey: 'usuário:🚀' } });
    return json(remote());
  });
  assert.deepEqual(await sdk.evaluate('checkout', { bucketingKey: 'usuário:🚀' }), remote());
  assert.equal(JSON.stringify(sdk).includes(token), false);
  sdk.close();
});
test('cache guarda resultado remoto, aplica default por chamada e não mascara DISABLED', async () => {
  let count = 0;
  const sdk = client(async () => { count++; return json(remote('checkout', { value: false, reason: 'MISSING_CONTEXT' })); });
  const first = await sdk.evaluate('checkout', { defaultValue: true }); first.value = false;
  assert.equal(await sdk.isEnabled('checkout'), false);
  assert.equal(await sdk.isEnabled('checkout', { defaultValue: true }), true); assert.equal(count, 1); sdk.close();
  for (const reason of ['DISABLED', 'NOT_FOUND']) {
    const sdk = client(async () => json(remote('checkout', { value: false, reason })));
    assert.equal(await sdk.isEnabled('checkout', { defaultValue: true }), reason === 'NOT_FOUND'); sdk.close();
  }
});
test('TTL, LRU, chave de contexto e isolamento por instância/credencial/baseURL', async () => {
  let count = 0; const fetch = async () => { count++; return json(remote()); };
  const sdk = client(fetch, { cacheTtlMs: 20, maxCacheEntries: 2 });
  await sdk.evaluate('checkout', { bucketingKey: 'one' }); await sdk.evaluate('checkout', { bucketingKey: 'two' });
  await sdk.evaluate('checkout', { bucketingKey: 'one' }); await sdk.evaluate('checkout', { bucketingKey: 'three' });
  assert.equal(count, 3); await sdk.evaluate('checkout', { bucketingKey: 'two' }); assert.equal(count, 4);
  await sleep(30); await sdk.evaluate('checkout', { bucketingKey: 'two' }); assert.equal(count, 5);
  const other = client(fetch, { token: 'fp_eval_' + 'b'.repeat(64) }); await other.evaluate('checkout', { bucketingKey: 'two' });
  const host = client(fetch, { baseUrl: 'https://example.invalid/api/v1' }); await host.evaluate('checkout'); assert.equal(count, 7);
  sdk.close(); other.close(); host.close();
});
test('deduplica requests, mas abort de um consumidor não cancela os demais', async () => {
  const gate = deferred(); let count = 0;
  const sdk = client(async () => { count++; return gate.promise; }); const signal = new globalThis.AbortController();
  const one = sdk.evaluate('checkout', { signal: signal.signal, defaultValue: true }); const two = sdk.evaluate('checkout');
  signal.abort(); assert.equal((await one).reason, 'SDK_ABORTED'); gate.resolve(json(remote()));
  assert.equal((await two).value, true); assert.equal(count, 1); sdk.close();
});
test('cancelar último consumidor libera slot e impede cache tardio', async () => {
  const gate = deferred(); let count = 0;
  const sdk = client(async () => { count++; return count === 1 ? gate.promise : json(remote()); }, { maxConcurrentRequests: 1 });
  const signal = new globalThis.AbortController(); const first = sdk.evaluate('checkout', { signal: signal.signal }); signal.abort();
  assert.equal((await first).reason, 'SDK_ABORTED'); assert.equal((await sdk.evaluate('checkout')).value, true);
  gate.resolve(json(remote('checkout', { value: false, reason: 'DISABLED' }))); await sleep(0);
  assert.equal((await sdk.evaluate('checkout')).value, true); assert.equal(count, 2); sdk.close();
});
test('timeout inclui transporte/body que ignoram signal e não é cacheado', async () => {
  for (const fetch of [async () => new Promise(() => {}), async () => new globalThis.Response(new globalThis.ReadableStream({ start() {} }), { headers: { 'Content-Type': 'application/json' } })]) {
    const sdk = client(fetch, { timeoutMs: 20 });
    const result = await sdk.evaluate('checkout', { defaultValue: true }); assert.equal(result.reason, 'SDK_TIMEOUT'); assert.equal(result.value, true); assert.equal(result.revision, null); sdk.close();
  }
});
test('close aborta pendências, esvazia cache e bloqueia novas chamadas, idempotente', async () => {
  let count = 0; const sdk = client(async () => { count++; return new Promise(() => {}); });
  const promise = sdk.evaluate('checkout'); sdk.close(); sdk.close();
  assert.equal((await promise).reason, 'SDK_CLOSED'); assert.equal((await sdk.evaluate('checkout')).reason, 'SDK_CLOSED'); assert.equal(count, 1);
});
test('limite de concorrência é explícito e chamadas equivalentes compartilham slot', async () => {
  const gate = deferred(); const sdk = client(async () => gate.promise, { maxConcurrentRequests: 1 });
  const first = sdk.evaluate('checkout'); const second = sdk.evaluate('checkout');
  assert.equal((await sdk.evaluate('other_flag')).reason, 'SDK_BUSY'); gate.resolve(json(remote()));
  assert.equal((await first).value, true); assert.equal((await second).value, true); sdk.close();
});
test('retry transitório limitado, Retry-After respeitado e erros permanentes não repetem', async () => {
  for (const status of [401,403,404,422,429,503]) {
    let calls = 0; const sdk = client(async () => { calls++; return new globalThis.Response(null, { status, headers: { 'Retry-After': '3600' } }); }, { maxRetries: 1, timeoutMs: 200 });
    const result = await sdk.evaluate('checkout'); assert.equal(calls, 1); assert.equal(result.revision, null);
    assert.equal(result.reason, [401,403].includes(status) ? 'SDK_UNAUTHORIZED' : 'SDK_HTTP_ERROR'); sdk.close();
  }
  for (const fail of [() => { throw new Error('offline'); }, () => new globalThis.Response(null, { status: 503 }), () => new globalThis.Response(null, { status: 429, headers: { 'Retry-After': '0' } })]) {
    let calls = 0; const sdk = client(async () => { calls++; if (calls === 1) return fail(); return json(remote()); }, { maxRetries: 1, timeoutMs: 500 });
    assert.equal((await sdk.evaluate('checkout')).value, true); assert.equal(calls, 2); sdk.close();
  }
  let calls = 0; const sdk = client(async () => { calls++; throw new Error('do not expose credentials'); }, { maxRetries: 1, timeoutMs: 500 });
  assert.equal((await sdk.evaluate('checkout')).reason, 'SDK_NETWORK_ERROR'); assert.equal(calls, 2); sdk.close();
});
test('401 limpa cache e resposta anterior em voo não o repopula', async () => {
  const gate = deferred(); let count = 0;
  const sdk = client(async (_url, init) => { count++; const { flagKey } = JSON.parse(init.body); return flagKey === 'pending_flag' && count === 2 ? gate.promise : flagKey === 'unauthorized_flag' ? new globalThis.Response(null, { status: 401 }) : json(remote(flagKey)); });
  await sdk.evaluate('cached_flag'); const pending = sdk.evaluate('pending_flag');
  await sdk.evaluate('unauthorized_flag'); gate.resolve(json(remote('pending_flag'))); await pending;
  await sdk.evaluate('cached_flag'); await sdk.evaluate('pending_flag'); assert.equal(count, 5); sdk.close();
});
test('validação de resposta, limite de bytes e falhas não entram em cache', async () => {
  const invalid = [remote('wrong_key'),remote('checkout',{ revision: -1 }),remote('checkout',{ revision: 2**53 }),remote('checkout',{ reason:'future_reason' }),remote('checkout',{ reason:'DISABLED',value:true }),remote('checkout',{ environment: 'bad/env' }), { value: true }];
  for (const payload of invalid) {
    let count = 0; const sdk = client(async () => { count++; return json(payload); });
    assert.equal((await sdk.evaluate('checkout')).reason,'SDK_INVALID_RESPONSE'); await sdk.evaluate('checkout'); assert.equal(count,2); sdk.close();
  }
  for (const body of ['{bad json', ' '.repeat(65537)]) {
    const sdk = client(async () => new globalThis.Response(body,{headers:{'Content-Type':'application/json'}}));
    assert.equal((await sdk.evaluate('checkout')).reason,'SDK_INVALID_RESPONSE'); sdk.close();
  }
});
test('descarta corpo de resposta com Content-Type incompatível', async () => {
  let cancelled = false;
  const sdk = client(async () => new globalThis.Response(new globalThis.ReadableStream({ cancel() { cancelled = true; } }), { headers: { 'Content-Type': 'text/html' } }));
  assert.equal((await sdk.evaluate('checkout')).reason,'SDK_INVALID_RESPONSE'); assert.equal(cancelled,true); sdk.close();
});
test('validação local não envia requests e default não modifica contexto remoto', async () => {
  let count = 0; const sdk = client(async () => { count++; return json(remote()); });
  for (const key of ['', 'Bad Key', 'x']) await assert.rejects(sdk.evaluate(key), TypeError);
  for (const bucketingKey of ['', 'á'.repeat(129), '\ud800', 'has\0nul']) await assert.rejects(sdk.evaluate('checkout',{bucketingKey}), TypeError);
  const signal = new globalThis.AbortController(); signal.abort(); assert.equal((await sdk.evaluate('checkout',{signal:signal.signal})).reason, 'SDK_ABORTED'); assert.equal(count,0); sdk.close();
  for (const options of [{baseUrl:'http://example.com/api/v1'},{baseUrl:'https://user:secret@example.com/api/v1'},{baseUrl:baseUrl+'?token=bad'},{baseUrl:baseUrl+'#secret'},{baseUrl:'https://example.com/other'},{timeoutMs:0},{cacheTtlMs:-1},{maxRetries:2},{maxCacheEntries:10001},{maxConcurrentRequests:0}]) assert.throws(() => client(async () => json(remote()),options),TypeError);
});

test('contexto preserva Unicode e BOM, com limite exato de bytes, sem normalização', async () => {
  const seen = [];
  const sdk = client(async (_url, init) => { seen.push(JSON.parse(init.body).context.bucketingKey); return json(remote()); });
  for (const bucketingKey of ['á'.repeat(128), '\ufeffcontext', 'é', 'e\u0301']) assert.equal(await sdk.isEnabled('checkout', { bucketingKey }), true);
  assert.deepEqual(seen,['á'.repeat(128),'\ufeffcontext','é','e\u0301']); sdk.close();
});
test('recusa construtor no navegador e abort interrompe espera de retry', async () => {
  globalThis.window = { document: {} };
  try { assert.throws(() => client(async () => json(remote())),TypeError); }
  finally { delete globalThis.window; }
  let count = 0; const signal = new globalThis.AbortController();
  const sdk = client(async () => { count++; return new globalThis.Response(null,{status:503}); },{maxRetries:1});
  const pending = sdk.evaluate('checkout',{signal:signal.signal}); await sleep(0); signal.abort();
  assert.equal((await pending).reason,'SDK_ABORTED'); assert.equal(count,1); sdk.close();
});
