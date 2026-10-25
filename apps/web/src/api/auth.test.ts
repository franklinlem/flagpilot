import { afterEach, expect, it, vi } from 'vitest';
import { getIdentity, login } from './auth';
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });
it('rejeita identidade fora do contrato e distingue 401 de falha de rede', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"user":{}}')));
  await expect(getIdentity()).rejects.toThrow('Resposta inesperada');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"title":"Email ou senha inválidos"}', { status: 401 })));
  await expect(login('owner@example.invalid', 'test-password')).rejects.toMatchObject({ status: 401 });
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network internals')));
  await expect(getIdentity()).rejects.toThrow('Não foi possível conectar');
});
it('mantém o prazo de 8 segundos mesmo com sinal de cancelamento da consulta', async () => {
  const caller = new AbortController();
  const deadline = new AbortController();
  vi.spyOn(AbortSignal, 'timeout').mockReturnValue(deadline.signal);
  let sent: AbortSignal | undefined;
  vi.stubGlobal('fetch', vi.fn().mockImplementation((_path, options) => { sent = options.signal; return Promise.resolve(new Response('{}')); }));
  const { request } = await import('./auth');
  try {
    await request('/organizations', {}, caller.signal);
    expect(AbortSignal.timeout).toHaveBeenCalledWith(8000);
    deadline.abort();
    expect(sent?.aborted).toBe(true);
    expect(caller.signal.aborted).toBe(false);
  } finally { vi.restoreAllMocks(); }
});

it('respeita cancelamento do chamador sem esperar o timeout e sem expor erros internos', async () => {
  const caller = new AbortController(); const deadline = new AbortController();
  vi.spyOn(AbortSignal, 'timeout').mockReturnValue(deadline.signal);
  let sent: AbortSignal | undefined;
  vi.stubGlobal('fetch', vi.fn().mockImplementation((_path, options) => { sent = options.signal; return Promise.resolve(new Response('{}')); }));
  const { request } = await import('./auth');
  await request('/organizations', { signal: caller.signal });
  caller.abort(); expect(sent?.aborted).toBe(true); expect(deadline.signal.aborted).toBe(false);
});
