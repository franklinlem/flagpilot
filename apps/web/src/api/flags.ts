import { z } from 'zod';
import { ApiError, request } from './auth';
import { roles } from './organizations';
import type { components } from './schema';
const counter = z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER);
const flagSchema = z.object({ id: z.uuid(), projectId: z.uuid(), key: z.string(), description: z.string(), archivedAt: z.iso.datetime().nullable(), version: counter });
const configSchema = z.object({ flagId: z.uuid(), flagKey: z.string(), environmentId: z.uuid(), enabled: z.boolean(), rolloutBps: z.number().int().min(0).max(10000), version: counter, revision: counter });
const flagsPage = z.object({ items: z.array(flagSchema), nextCursor: z.string().nullable() });
const snapshotSchema = z.object({ items: z.array(configSchema), nextCursor: z.string().nullable(), environment: z.object({ id: z.uuid(), projectId: z.uuid(), name: z.string(), slug: z.string(), isProduction: z.boolean(), revision: counter }), projectName: z.string(), role: z.enum(roles) });
export type Flag = z.infer<typeof flagSchema>;
export type Config = z.infer<typeof configSchema>;
export type Snapshot = z.infer<typeof snapshotSchema>;
async function parsed<T>(response: Response, schema: z.ZodType<T>): Promise<T> {
  const result = schema.safeParse(await response.json().catch(() => null));
  if (!result.success) throw new ApiError(0, 'Resposta inesperada do serviço.');
  return result.data;
}
async function versioned<T>(response: Response, schema: z.ZodType<T>) {
  const etag = response.headers.get('ETag');
  if (!etag) throw new ApiError(0, 'O serviço não informou a revisão do recurso.');
  return { data: await parsed(response, schema), etag };
}
const path = (project: string, key?: string) => `/projects/${encodeURIComponent(project)}/flags${key === undefined ? '' : `/${encodeURIComponent(key)}`}`;
const mutation = (method: string, csrf: string, body: unknown, etag?: string): RequestInit => ({ method, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, ...(etag ? { 'If-Match': etag } : {}) }, body: JSON.stringify(body) });
export async function listFlags(project: string, cursor = '', signal?: AbortSignal, filter: { q: string; status: string } = { q: '', status: 'all' }) { return parsed(await request(`${path(project)}?limit=25&q=${encodeURIComponent(filter.q)}&status=${encodeURIComponent(filter.status)}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`, {}, signal), flagsPage); }
export async function getFlag(project: string, key: string, signal?: AbortSignal) { return versioned(await request(path(project, key), {}, signal), flagSchema); }
export async function createFlag(project: string, body: components['schemas']['CreateFlag'], csrf: string) { return versioned(await request(path(project), mutation('POST', csrf, body)), flagSchema); }
export async function changeFlag(project: string, key: string, body: components['schemas']['UpdateFlag'], csrf: string, etag: string) { return versioned(await request(path(project, key), mutation('PATCH', csrf, body, etag)), flagSchema); }
export async function listConfigs(environment: string, cursor = '', signal?: AbortSignal) { return versioned(await request(`/environments/${encodeURIComponent(environment)}/flags?limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`, {}, signal), snapshotSchema); }
export async function putConfig(environment: string, key: string, body: components['schemas']['UpdateFlagConfig'], csrf: string, etag: string) { return versioned(await request(`/environments/${encodeURIComponent(environment)}/flags/${encodeURIComponent(key)}`, mutation('PUT', csrf, body, etag)), configSchema); }
