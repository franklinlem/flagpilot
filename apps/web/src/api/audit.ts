import { z } from 'zod';
import { ApiError, request } from './auth';
const details = z.record(z.string(), z.union([z.string(),z.number(),z.boolean(),z.null()])).nullable();
const event = z.object({ id: z.uuid(), organizationId: z.uuid(), actorId: z.uuid().nullable(), actorLabel: z.enum(['user','local-operator','unknown']), action: z.string(), resourceType: z.string(), resourceId: z.uuid(), before: details, after: details, createdAt: z.iso.datetime() });
const page = z.object({ items: z.array(event), nextCursor: z.string().nullable() });
export type AuditEvent = z.infer<typeof event>;
export async function listAudit(organization: string, filter: { action: string; resourceType: string }, cursor = '', signal?: AbortSignal) {
  const params = new URLSearchParams({ limit: '25', ...filter, ...(cursor ? { cursor } : {}) });
  const response = await request(`/organizations/${encodeURIComponent(organization)}/audit-events?${params}`,{},signal);
  const result = page.safeParse(await response.json().catch(() => null));
  if (!result.success) throw new ApiError(0,'Resposta inesperada do serviço.');
  return result.data;
}
