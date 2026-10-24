import { afterEach, expect, it, vi } from 'vitest';
import { getIdentity, login } from './auth';
afterEach(() => vi.unstubAllGlobals());
it('rejeita identidade fora do contrato e distingue 401 de falha de rede', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"user":{}}')));
  await expect(getIdentity()).rejects.toThrow('Resposta inesperada');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"title":"Email ou senha inválidos"}', { status: 401 })));
  await expect(login('owner@example.invalid', 'test-password')).rejects.toMatchObject({ status: 401 });
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network internals')));
  await expect(getIdentity()).rejects.toThrow('Não foi possível conectar');
});
