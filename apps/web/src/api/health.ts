import type { components } from './schema';
export type Health = components['schemas']['Health'];

export async function getHealth(signal?: AbortSignal): Promise<Health> {
  const response = await fetch('/api/v1/health/live', { signal });
  if (!response.ok) throw new Error('Serviço indisponível');
  const data: unknown = await response.json();
  if (typeof data !== 'object' || data === null || !('status' in data) || data.status !== 'ok') {
    throw new Error('Resposta inválida');
  }
  return { status: 'ok' };
}
