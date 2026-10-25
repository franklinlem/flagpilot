// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { putConfig, type Config, type Snapshot } from '../../api/flags';
import type { Identity } from '../../api/auth';
import { ConfigRow, percentageBps } from './ConfigRow';
vi.mock('../../api/flags', () => ({ putConfig: vi.fn() }));
afterEach(() => { cleanup(); vi.resetAllMocks(); });
const config: Config = { flagId: 'flag', flagKey: 'checkout', environmentId: 'env', enabled: true, rolloutBps: 2500, version: 1, revision: 1 };
const snapshot: Snapshot = { items: [config], nextCursor: null, environment: { id: 'env', projectId: 'project', name: 'Produção', slug: 'production', isProduction: true, revision: 1 }, projectName: 'Loja', role: 'Editor' };
function mount(data = snapshot) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  render(<QueryClientProvider client={client}><ConfigRow config={config} snapshot={data} etag='"revision"' identity={{ csrfToken: 'csrf' } as Identity} refresh={async () => {}} onExpired={() => {}} /></QueryClientProvider>);
  return client;
}
it('converte percentuais exatamente e rejeita excesso de precisão e limites', () => {
  for (const [value, expected] of [['0',0],['0,01',1],['25.37',2537],['99,99',9999],['100.00',10000]] as const) expect(percentageBps(value)).toBe(expected);
  for (const value of ['', '-1', '100.01', '101', '1.001', '1e2', 'Infinity', ' 20']) expect(percentageBps(value)).toBeNull();
});
it('exige confirmação com percentuais anterior/novo e permite cancelar sem salvar', async () => {
  const client = mount(); const user = userEvent.setup();
  const input = screen.getByLabelText('Porcentagem de checkout');
  await user.clear(input); await user.type(input, '37,25');
  await user.click(screen.getByRole('button', { name: 'Salvar porcentagem' }));
  expect(screen.getByRole('group').textContent).toContain('25% → 37.25%');
  expect(putConfig).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button', { name: 'Cancelar' }));
  expect(putConfig).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button', { name: 'Salvar porcentagem' }));
  vi.mocked(putConfig).mockResolvedValue({ data: { ...config, rolloutBps: 3725 }, etag: 'next' });
  await user.click(screen.getByRole('button', { name: 'Confirmar alteração em produção' }));
  await waitFor(() => expect(putConfig).toHaveBeenCalledWith('env', 'checkout', { enabled: true, rolloutBps: 3725, confirmProduction: true }, 'csrf', '"revision"'));
  client.clear();
});
it('valida entrada e preserva porcentagem ao desativar', async () => {
  const client = mount({ ...snapshot, environment: { ...snapshot.environment, isProduction: false } }); const user = userEvent.setup();
  const input = screen.getByLabelText('Porcentagem de checkout');
  await user.clear(input); await user.type(input, '101');
  await user.click(screen.getByRole('button', { name: 'Salvar porcentagem' }));
  expect((await screen.findByRole('alert')).textContent).toContain('Informe de 0 a 100');
  expect(putConfig).not.toHaveBeenCalled();
  vi.mocked(putConfig).mockResolvedValue({ data: { ...config, enabled: false }, etag: 'next' });
  await user.click(screen.getByRole('button', { name: 'Desativar flag' }));
  await waitFor(() => expect(putConfig).toHaveBeenCalledWith('env', 'checkout', { enabled: false, rolloutBps: 2500, confirmProduction: false }, 'csrf', '"revision"'));
  client.clear();
});
it('Viewer consulta porcentagem sem controles de escrita', () => {
  const client = mount({ ...snapshot, role: 'Viewer' });
  expect(screen.getByText('Rollout: 25%')).toBeTruthy();
  expect(screen.queryByRole('button')).toBeNull();
  expect(screen.queryByRole('textbox')).toBeNull();
  client.clear();
});
