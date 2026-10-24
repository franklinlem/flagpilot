import { z } from 'zod';
import { ApiError, request } from './auth';
import type { components } from './schema';
export const roles = ['Owner', 'Admin', 'Editor', 'Viewer'] as const;
export type Role = typeof roles[number];
export const roleNames: Record<Role, string> = { Owner: 'Proprietário', Admin: 'Administrador', Editor: 'Editor', Viewer: 'Visualizador' };
const organizationSchema = z.object({ id: z.uuid(), name: z.string(), slug: z.string(), role: z.enum(roles) });
const memberSchema = z.object({ organizationId: z.uuid(), userId: z.uuid(), email: z.email(), role: z.enum(roles), disabled: z.boolean() });
const organizationsPageSchema = z.object({ items: z.array(organizationSchema), nextCursor: z.string().nullable() });
const membersPageSchema = z.object({ items: z.array(memberSchema), nextCursor: z.string().nullable() });
export type Organization = z.infer<typeof organizationSchema>;
export type Member = z.infer<typeof memberSchema>;
async function parsed<T>(response: Response, schema: z.ZodType<T>): Promise<T> {
  const value = schema.safeParse(await response.json().catch(() => null));
  if (!value.success) throw new ApiError(0, 'Resposta inesperada do serviço.');
  return value.data;
}
const page = (cursor: string) => `?limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`;
export async function listOrganizations(cursor = '', signal?: AbortSignal) { return parsed(await request(`/organizations${page(cursor)}`, {}, signal), organizationsPageSchema); }
export async function getOrganization(id: string, signal?: AbortSignal) { return parsed(await request(`/organizations/${encodeURIComponent(id)}`, {}, signal), organizationSchema); }
export async function listMembers(orgId: string, cursor = '', signal?: AbortSignal) { return parsed(await request(`/organizations/${encodeURIComponent(orgId)}/members${page(cursor)}`, {}, signal), membersPageSchema); }
const mutation = (method: string, csrf: string, body?: unknown): RequestInit => ({ method, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, ...(body === undefined ? {} : { body: JSON.stringify(body) }) });
export async function createOrganization(body: components['schemas']['NamedRequest'], csrf: string) { return parsed(await request('/organizations', mutation('POST', csrf, body)), organizationSchema); }
export async function addMember(org: string, body: components['schemas']['MemberRequest'], csrf: string) { return parsed(await request(`/organizations/${encodeURIComponent(org)}/members`, mutation('POST', csrf, body)), memberSchema); }
export async function changeRole(org: string, user: string, role: Role, csrf: string) { return parsed(await request(`/organizations/${encodeURIComponent(org)}/members/${encodeURIComponent(user)}`, mutation('PATCH', csrf, { role })), memberSchema); }
export async function removeMember(org: string, user: string, csrf: string) { await request(`/organizations/${encodeURIComponent(org)}/members/${encodeURIComponent(user)}`, mutation('DELETE', csrf)); }
