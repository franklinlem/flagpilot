import { z } from 'zod';
import type { components } from './schema';

const identitySchema = z.object({
  user: z.object({ id: z.uuid(), email: z.email(), createdAt: z.iso.datetime() }),
  memberships: z.array(z.object({ organizationId: z.uuid(), userId: z.uuid(), role: z.enum(['Owner', 'Admin', 'Editor', 'Viewer']), organizationName: z.string() })),
  csrfToken: z.string().regex(/^[a-f0-9]{64}$/),
});
export type Identity = z.infer<typeof identitySchema>;
export class ApiError extends Error {
  constructor(public readonly status: number, message: string) { super(message); }
}
async function request(path: string, options: RequestInit = {}, signal?: AbortSignal): Promise<Response> {
  let response: Response;
  try {
    response = await fetch(`/api/v1${path}`, { ...options, credentials: 'same-origin', signal: signal ?? AbortSignal.timeout(8000) });
  } catch { throw new ApiError(0, 'Não foi possível conectar ao serviço. Tente novamente.'); }
  if (!response.ok) {
    const body: unknown = await response.json().catch(() => null);
    const message = typeof body === 'object' && body !== null && 'title' in body && typeof body.title === 'string'
      ? body.title.slice(0, 200) : 'Não foi possível concluir a operação.';
    throw new ApiError(response.status, message);
  }
  return response;
}
async function parseIdentity(response: Response): Promise<Identity> {
  const result = identitySchema.safeParse(await response.json().catch(() => null));
  if (!result.success) throw new ApiError(0, 'Resposta inesperada do serviço. Tente novamente.');
  return result.data;
}
export async function getIdentity(signal?: AbortSignal): Promise<Identity> {
  return parseIdentity(await request('/me', {}, signal));
}
export async function login(email: string, password: string): Promise<Identity> {
  const body: components['schemas']['LoginRequest'] = { email, password };
  return parseIdentity(await request('/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }));
}
export async function logout(csrfToken: string): Promise<void> {
  await request('/auth/logout', { method: 'POST', headers: { 'X-CSRF-Token': csrfToken } });
}
