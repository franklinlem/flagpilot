import type { ClientOptions, EvaluateOptions, EvaluationResult, RemoteResult } from './types.js';
import { baseURL, integer, validateClient, validateRequest } from './validation.js';
import { abortable, cancelled, closed, local, request } from './transport.js';
type Entry = { controller: AbortController; users: number; promise: Promise<EvaluationResult> };
export class FlagPilotClient {
  #url: string; #token: string; #fetch: typeof globalThis.fetch;
  #timeout: number; #ttl: number; #max: number; #concurrent: number; #retries: number;
  #closed = false;
  #generation = 0;
  #cache = new Map<string, { result: RemoteResult; until: number }>();
  #pending = new Map<string, Entry>();
  constructor(options: ClientOptions) {
    validateClient(options);
    this.#url = baseURL(options.baseUrl); this.#token = options.token; this.#fetch = options.fetch ?? globalThis.fetch;
    this.#timeout = integer(options.timeoutMs,1500,1,30000); this.#ttl = integer(options.cacheTtlMs,30000,0,300000);
    this.#max = integer(options.maxCacheEntries,1000,0,10000); this.#concurrent = integer(options.maxConcurrentRequests,100,1,1000);
    this.#retries = integer(options.maxRetries,1,0,1);
  }
  async evaluate(key: string, options: EvaluateOptions = {}): Promise<EvaluationResult> {
    validateRequest(key, options);
    const fallback = (result: EvaluationResult): EvaluationResult => ({ ...result, value: result.revision === null || result.reason === 'MISSING_CONTEXT' || result.reason === 'NOT_FOUND' ? (options.defaultValue ?? false) : result.value });
    if (this.#closed) return fallback(local(key,'SDK_CLOSED'));
    if (options.signal?.aborted) return fallback(local(key,'SDK_ABORTED'));
    // Mapas exclusivos desta instância: URL e credencial imutáveis, sem cache global.
    const cacheKey = JSON.stringify([key, options.bucketingKey ?? null]);
    const cached = this.#cache.get(cacheKey);
    if (cached) {
      this.#cache.delete(cacheKey);
      if (cached.until > Date.now()) { this.#cache.set(cacheKey,cached); return fallback(cached.result); }
    }
    let entry = this.#pending.get(cacheKey);
    if (!entry) {
      if (this.#pending.size >= this.#concurrent) return fallback(local(key,'SDK_BUSY'));
      const controller = new AbortController();
      const generation = this.#generation;
      const promise = request({ url: this.#url, token: this.#token, key, bucketingKey: options.bucketingKey, controller, timeoutMs: this.#timeout, retries: this.#retries, fetch: this.#fetch }).then(result => {
        if (result.reason === 'SDK_UNAUTHORIZED') { this.#generation++; this.#cache.clear(); }
        if (generation === this.#generation && !this.#closed && !controller.signal.aborted && result.revision !== null && this.#ttl > 0 && this.#max > 0) {
          for (const [storedKey, stored] of this.#cache) if (stored.until <= Date.now()) this.#cache.delete(storedKey);
          if (this.#cache.size >= this.#max) this.#cache.delete(this.#cache.keys().next().value!);
          this.#cache.set(cacheKey,{ result, until: Date.now()+this.#ttl });
        }
        return result;
      }).finally(() => { if (this.#pending.get(cacheKey)?.controller === controller) this.#pending.delete(cacheKey); });
      entry = { controller, promise, users: 0 }; this.#pending.set(cacheKey,entry);
    }
    entry.users++;
    try { return fallback(await (options.signal ? abortable(entry.promise,options.signal) : entry.promise)); }
    catch { return fallback(local(key,'SDK_ABORTED')); }
    finally {
      entry.users--;
      if (entry.users === 0 && this.#pending.get(cacheKey) === entry) { this.#pending.delete(cacheKey); entry.controller.abort(cancelled); }
    }
  }
  async isEnabled(key: string, options: EvaluateOptions = {}): Promise<boolean> { return (await this.evaluate(key, options)).value; }
  close(): void {
    if (this.#closed) return;
    this.#closed = true; this.#cache.clear(); this.#token = '';
    for (const entry of this.#pending.values()) entry.controller.abort(closed);
    this.#pending.clear();
  }
}
