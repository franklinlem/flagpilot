import type { EvaluationResult, LocalReason } from './types.js';
import { remoteResult } from './validation.js';
export const timeout = Symbol('timeout');
export const closed = Symbol('closed');
export const cancelled = Symbol('cancelled');
export function local(flagKey: string, reason: LocalReason): EvaluationResult { return { flagKey, value: false, reason, environment: null, revision: null }; }
// Até transportes que ignorem signal devem liberar o consumidor ao abortar.
export function abortable<T>(promise: Promise<T>, signal: AbortSignal): Promise<T> {
  return new Promise((resolve, reject) => {
    const abort = () => reject(cancelled);
    if (signal.aborted) { promise.catch(() => {}); reject(cancelled); return; }
    signal.addEventListener('abort', abort, { once: true });
    promise.then(resolve, reject).finally(() => signal.removeEventListener('abort', abort));
  });
}
function wait(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const finish = () => { clearTimeout(timer); signal.removeEventListener('abort', abort); resolve(); };
    const abort = () => { clearTimeout(timer); signal.removeEventListener('abort', abort); reject(cancelled); };
    const timer = setTimeout(finish, ms);
    if (signal.aborted) abort(); else signal.addEventListener('abort', abort, { once: true });
  });
}
async function readJSON(response: Response, signal: AbortSignal): Promise<unknown> {
  if (response.headers.get('Content-Type')?.split(';')[0].trim().toLowerCase() !== 'application/json' || !response.body) {
    void response.body?.cancel().catch(() => {});
    throw new Error('invalid response');
  }
  const reader = response.body.getReader();
  const parts: Uint8Array[] = []; let size = 0;
  try {
    for (;;) {
      const chunk = await abortable(reader.read(), signal);
      if (chunk.done) break;
      size += chunk.value.length;
      if (size > 65536) throw new Error('invalid response');
      parts.push(chunk.value);
    }
    const body = new Uint8Array(size); let offset = 0;
    for (const part of parts) { body.set(part, offset); offset += part.length; }
    return JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(body));
  } finally { void reader.cancel().catch(() => {}); reader.releaseLock(); }
}
function retryDelay(header: string | null): number {
  const jitter = 25 + Math.floor(Math.random() * 50);
  if (header === null) return jitter;
  if (/^\d+$/.test(header)) return Number(header) * 1000 + jitter;
  const seconds = Date.parse(header) - Date.now();
  return Number.isFinite(seconds) ? Math.max(0, seconds) + jitter : jitter;
}
export async function request(args: { url: string; token: string; key: string; bucketingKey?: string; controller: AbortController; timeoutMs: number; retries: number; fetch: typeof globalThis.fetch }): Promise<EvaluationResult> {
  const { controller, key } = args;
  const signal = controller.signal;
  const deadline = Date.now() + args.timeoutMs;
  const timer = setTimeout(() => controller.abort(timeout), args.timeoutMs);
  try {
    for (let attempt = 0; ; attempt++) {
      let response: Response;
      try {
        response = await abortable(args.fetch(args.url + '/evaluate', { method: 'POST', headers: { Authorization: `Bearer ${args.token}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ flagKey: key, ...(args.bucketingKey === undefined ? {} : { context: { bucketingKey: args.bucketingKey } }) }), signal, redirect: 'error', credentials: 'omit', cache: 'no-store' }), signal);
      } catch {
        if (signal.aborted) throw cancelled;
        if (attempt < args.retries && deadline - Date.now() > 75) { await wait(25 + Math.floor(Math.random() * 50), signal); continue; }
        return local(key, 'SDK_NETWORK_ERROR');
      }
      if (!response.ok) {
        void response.body?.cancel().catch(() => {});
        if (response.status === 401 || response.status === 403) return local(key, 'SDK_UNAUTHORIZED');
        const delay = retryDelay(response.headers.get('Retry-After'));
        if (attempt < args.retries && [408,429,500,502,503,504].includes(response.status) && delay < deadline - Date.now()) { await wait(delay, signal); continue; }
        return local(key, 'SDK_HTTP_ERROR');
      }
      try { return remoteResult(await readJSON(response, signal), key) ?? local(key, 'SDK_INVALID_RESPONSE'); }
      catch { if (signal.aborted) throw cancelled; return local(key, 'SDK_INVALID_RESPONSE'); }
    }
  } catch {
    return local(key, signal.reason === timeout ? 'SDK_TIMEOUT' : signal.reason === closed ? 'SDK_CLOSED' : 'SDK_ABORTED');
  } finally { clearTimeout(timer); }
}
