import { z } from 'zod';
import { ApiError, request } from './auth';
import { roles } from './organizations';
import type { components } from './schema';
const tokenSchema = z.object({ id: z.uuid(), environmentId: z.uuid(), name: z.string(), prefix: z.string(), expiresAt: z.iso.datetime(), revokedAt: z.iso.datetime().nullable() });
const pageSchema = z.object({ items: z.array(tokenSchema), nextCursor: z.string().nullable(), environment: z.object({ id: z.uuid(), projectId: z.uuid(), name: z.string(), slug: z.string(), isProduction: z.boolean(), revision: z.number().int().nonnegative() }), projectName: z.string(), role: z.enum(roles) });
const issuedSchema = z.object({ token: tokenSchema, secret: z.string().regex(/^fp_eval_[a-f0-9]{64}$/) });
export type Token = z.infer<typeof tokenSchema>;
async function parsed<T>(response: Response, schema: z.ZodType<T>): Promise<T> { const result = schema.safeParse(await response.json().catch(() => null)); if (!result.success) throw new ApiError(0, 'Resposta inesperada do serviço.'); return result.data; }
const path = (env: string) => `/environments/${encodeURIComponent(env)}/tokens`;
export async function listTokens(env: string, cursor = '', signal?: AbortSignal) { return parsed(await request(`${path(env)}?limit=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`, {}, signal), pageSchema); }
// O chamador mantém o segredo apenas em estado local; não usar em Query/Mutation cache.
export async function issueToken(env: string, body: components['schemas']['TokenRequest'], csrf: string, signal: AbortSignal) { return parsed(await request(path(env), { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify(body) }, signal), issuedSchema); }
export async function revokeToken(env: string, id: string, csrf: string) { await request(`${path(env)}/${encodeURIComponent(id)}`, { method: 'DELETE', headers: { 'X-CSRF-Token': csrf } }); }
