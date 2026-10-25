/** Referência interna da ADR-003; não é um cliente nem um snapshot público. */
export async function bucketV1(projectId: string, environmentId: string, flagKey: string, salt: string, bucketingKey: string): Promise<number> {
  const bytes = new TextEncoder().encode(`${projectId}:${environmentId}:${flagKey}:${salt}:${bucketingKey}`);
  const digest = await globalThis.crypto.subtle.digest('SHA-256', bytes);
  return Number(new DataView(digest).getBigUint64(0, false) % 10000n);
}
