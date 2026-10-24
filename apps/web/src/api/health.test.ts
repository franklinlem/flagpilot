import { afterEach, expect, it, vi } from 'vitest';
import { getHealth } from './health';
afterEach(() => vi.unstubAllGlobals());
it('rejeita resposta inválida ou serviço indisponível', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"status":"wrong"}')));
  await expect(getHealth()).rejects.toThrow('Resposta inválida');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 503 })));
  await expect(getHealth()).rejects.toThrow('Serviço indisponível');
});
it('aceita resposta saudável', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"status":"ok"}')));
  await expect(getHealth()).resolves.toEqual({ status: 'ok' });
});
