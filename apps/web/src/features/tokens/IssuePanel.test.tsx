// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { issueToken } from '../../api/tokens';
import { IssuePanel } from './IssuePanel';
vi.mock('../../api/tokens', () => ({ issueToken: vi.fn() }));
afterEach(() => { cleanup(); vi.resetAllMocks(); });
it('exibe segredo uma vez, sem exigir cache de consultas, e limpa ao ocultar', async () => {
  const secret = 'fp_eval_' + 'b'.repeat(64);
  vi.mocked(issueToken).mockResolvedValue({ secret, token: { id: '00000000-0000-4000-8000-000000000001', environmentId: '00000000-0000-4000-8000-000000000002', name: 'Backend', prefix: 'fp_eval_bbbbbbbb', expiresAt: '2027-01-01T00:00:00Z', revokedAt: null } });
  const refresh = vi.fn(async () => {});
  const user = userEvent.setup();
  render(<IssuePanel environment="00000000-0000-4000-8000-000000000002" csrf="test" refresh={refresh} onExpired={() => {}} />);
  await user.click(screen.getByRole('button', { name: 'Emitir token' }));
  expect(await screen.findByText('Informe um nome.')).toBeTruthy();
  expect(issueToken).not.toHaveBeenCalled();
  await user.type(screen.getByLabelText('Nome do token'), 'Backend');
  await user.click(screen.getByRole('button', { name: 'Emitir token' }));
  const field = await screen.findByLabelText('Segredo do token');
  expect((field as HTMLTextAreaElement).value === secret).toBe(true);
  expect(refresh).toHaveBeenCalledOnce();
  expect(localStorage.length + sessionStorage.length).toBe(0);
  await user.click(screen.getByRole('button', { name: 'Já guardei, ocultar segredo' }));
  expect(screen.queryByLabelText('Segredo do token')).toBeNull();
  expect(issueToken).toHaveBeenCalledOnce();
});
