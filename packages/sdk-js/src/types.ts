export type RemoteReason = 'DISABLED' | 'ENABLED' | 'PERCENTAGE_ROLLOUT' | 'MISSING_CONTEXT' | 'NOT_FOUND';
export type LocalReason = 'SDK_TIMEOUT' | 'SDK_ABORTED' | 'SDK_NETWORK_ERROR' | 'SDK_HTTP_ERROR' | 'SDK_UNAUTHORIZED' | 'SDK_INVALID_RESPONSE' | 'SDK_CLOSED' | 'SDK_BUSY';
export type RemoteResult = { flagKey: string; value: boolean; reason: RemoteReason; environment: string; revision: number };
export type LocalResult = { flagKey: string; value: boolean; reason: LocalReason; environment: null; revision: null };
export type EvaluationResult = RemoteResult | LocalResult;
export interface EvaluateOptions {
  bucketingKey?: string;
  defaultValue?: boolean;
  signal?: AbortSignal;
}
export interface ClientOptions {
  /** URL absoluta terminando em /api/v1. HTTP permitido somente em loopback. */
  baseUrl: string;
  /** Exclusivo para backend confiável; nunca embutir no navegador. */
  token: string;
  timeoutMs?: number;
  cacheTtlMs?: number;
  maxCacheEntries?: number;
  maxConcurrentRequests?: number;
  maxRetries?: 0 | 1;
  /** Transporte confiável, sujeito ao mesmo timeout e validação. */
  fetch?: typeof globalThis.fetch;
}
