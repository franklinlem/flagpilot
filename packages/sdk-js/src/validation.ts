import type { ClientOptions, EvaluateOptions, RemoteResult } from './types.js';
export function baseURL(value: string): string {
  let url: URL;
  try { url = new URL(value); } catch { throw new TypeError('baseUrl inválida.'); }
  const loopback = ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname);
  if ((url.protocol !== 'https:' && !(url.protocol === 'http:' && loopback)) || url.username || url.password || url.search || url.hash || !url.pathname.replace(/\/+$/, '').endsWith('/api/v1')) throw new TypeError('baseUrl exige HTTPS (HTTP somente loopback), sem credenciais/query/fragmento, terminando em /api/v1.');
  return url.href.replace(/\/+$/, '');
}
export function integer(value: unknown, fallback: number, min: number, max: number): number {
  const selected = value === undefined ? fallback : value;
  if (typeof selected !== 'number' || !Number.isInteger(selected) || selected < min || selected > max) throw new TypeError('Opção numérica fora dos limites do SDK.');
  return selected;
}
export function validateClient(options: ClientOptions): void {
  if (typeof window !== 'undefined' && typeof window.document !== 'undefined') throw new TypeError('Use FlagPilotClient somente no backend; navegador deve usar BFF.');
  if (!options || typeof options.token !== 'string' || !/^fp_eval_[a-f0-9]{64}$/.test(options.token)) throw new TypeError('Token de avaliação inválido.');
  if (options.fetch !== undefined && typeof options.fetch !== 'function') throw new TypeError('Transporte inválido.');
}
export function validateRequest(key: string, options: EvaluateOptions): void {
  if (typeof key !== 'string' || !/^[a-z][a-z0-9_.-]{1,79}$/.test(key)) throw new TypeError('Chave de flag inválida.');
  if (!options || (options.defaultValue !== undefined && typeof options.defaultValue !== 'boolean')) throw new TypeError('Opções de avaliação inválidas.');
  if (options.signal !== undefined && !(options.signal instanceof AbortSignal)) throw new TypeError('AbortSignal inválido.');
  if (options.bucketingKey !== undefined) {
    const value = options.bucketingKey;
    if (typeof value !== 'string') throw new TypeError('bucketingKey deve ser string.');
    const bytes = new TextEncoder().encode(value);
    if (new TextDecoder('utf-8', { ignoreBOM: true }).decode(bytes) !== value || value.includes('\0') || bytes.length < 1 || bytes.length > 256) throw new TypeError('bucketingKey deve conter 1 a 256 bytes UTF-8 válidos, sem NUL.');
  }
}
export function remoteResult(data: unknown, key: string): RemoteResult | null {
  if (!data || typeof data !== 'object') return null;
  const row = data as Record<string, unknown>;
  if (row.flagKey !== key || typeof row.value !== 'boolean' || typeof row.environment !== 'string' || !/^[a-z][a-z0-9-]{1,62}$/.test(row.environment) || !Number.isSafeInteger(row.revision) || (row.revision as number) < 0) return null;
  if (!['DISABLED', 'ENABLED', 'PERCENTAGE_ROLLOUT', 'MISSING_CONTEXT', 'NOT_FOUND'].includes(row.reason as string)) return null;
  if ((row.reason === 'ENABLED' && !row.value) || (['DISABLED','MISSING_CONTEXT','NOT_FOUND'].includes(row.reason as string) && row.value)) return null;
  return { flagKey: key, value: row.value, reason: row.reason as RemoteResult['reason'], environment: row.environment, revision: row.revision as number };
}
