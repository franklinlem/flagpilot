import { z } from 'zod';
import { ApiError, request } from './auth';
import { roles } from './organizations';
import type { components } from './schema';
const projectSchema = z.object({ id: z.uuid(), organizationId: z.uuid(), name: z.string(), slug: z.string(), role: z.enum(roles) });
const environmentSchema = z.object({ id: z.uuid(), projectId: z.uuid(), name: z.string(), slug: z.string(), isProduction: z.boolean(), revision: z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER) });
const projectsPage = z.object({ items: z.array(projectSchema), nextCursor: z.string().nullable() });
const environmentsPage = z.object({ items: z.array(environmentSchema), nextCursor: z.string().nullable() });
async function parsed<T>(response: Response, schema: z.ZodType<T>): Promise<T> {
  const value = schema.safeParse(await response.json().catch(() => null));
  if (!value.success) throw new ApiError(0, 'Resposta inesperada do serviço.');
  return value.data;
}
const page = (cursor: string) => `?limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`;
const mutation = (csrf: string, body: unknown): RequestInit => ({ method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify(body) });
export async function listProjects(org: string, cursor = '', signal?: AbortSignal) { return parsed(await request(`/organizations/${encodeURIComponent(org)}/projects${page(cursor)}`, {}, signal), projectsPage); }
export async function getProject(project: string, signal?: AbortSignal) { return parsed(await request(`/projects/${encodeURIComponent(project)}`, {}, signal), projectSchema); }
export async function listEnvironments(project: string, cursor = '', signal?: AbortSignal) { return parsed(await request(`/projects/${encodeURIComponent(project)}/environments${page(cursor)}`, {}, signal), environmentsPage); }
export async function createProject(org: string, values: components['schemas']['NamedRequest'], csrf: string) { return parsed(await request(`/organizations/${encodeURIComponent(org)}/projects`, mutation(csrf, values)), projectSchema); }
export async function createEnvironment(project: string, values: components['schemas']['EnvironmentRequest'], csrf: string) { return parsed(await request(`/projects/${encodeURIComponent(project)}/environments`, mutation(csrf, values)), environmentSchema); }
