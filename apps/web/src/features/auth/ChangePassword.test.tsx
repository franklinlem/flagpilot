// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { ApiError, changePassword, type Identity } from '../../api/auth';
import { ChangePassword } from './ChangePassword';
vi.mock('../../api/auth', async original => ({ ...await original<typeof import('../../api/auth')>(), changePassword: vi.fn() }));
afterEach(() => { cleanup(); vi.resetAllMocks(); });
const identity: Identity = { user: { id: 'test', email: 'security@example.invalid', createdAt: '2026-10-27T12:00:00Z' }, memberships: [], csrfToken: 'a'.repeat(64) };
function setup() {
  const onChanged = vi.fn(); const onExpired = vi.fn();
  render(<MemoryRouter><ChangePassword identity={identity} onChanged={onChanged} onExpired={onExpired} /></MemoryRouter>);
  return { user: userEvent.setup(), onChanged, onExpired };
}
async function fill(user: ReturnType<typeof userEvent.setup>, confirmation = 'new-secure-password') {
  await user.type(screen.getByLabelText('Senha atual'), 'old-secure-password');
  await user.type(screen.getByLabelText('Nova senha'), 'new-secure-password');
  await user.type(screen.getByLabelText('Confirmar nova senha'), confirmation);
}
it('valida confirmação antes de enviar e nunca envia confirmação ao servidor', async () => {
  const { user, onChanged } = setup();
  await fill(user, 'different-confirmation');
  await user.click(screen.getByRole('button', { name: 'Alterar senha e sair' }));
  await screen.findByText('As senhas não coincidem.');
  expect(changePassword).not.toHaveBeenCalled();
  await user.clear(screen.getByLabelText('Confirmar nova senha'));
  await user.type(screen.getByLabelText('Confirmar nova senha'), 'new-secure-password');
  vi.mocked(changePassword).mockResolvedValue();
  await user.click(screen.getByRole('button', { name: 'Alterar senha e sair' }));
  await waitFor(() => expect(onChanged).toHaveBeenCalledOnce());
  expect(changePassword).toHaveBeenCalledWith({ currentPassword: 'old-secure-password', newPassword: 'new-secure-password' }, identity.csrfToken, expect.any(AbortSignal));
  for (const label of ['Senha atual', 'Nova senha', 'Confirmar nova senha']) expect((screen.getByLabelText(label) as HTMLInputElement).value).toBe('');
});
it('limpa campos e libera formulário após senha atual incorreta', async () => {
  const { user, onExpired } = setup();
  vi.mocked(changePassword).mockRejectedValue(new ApiError(403, 'Senha atual incorreta'));
  await fill(user);
  await user.click(screen.getByRole('button', { name: 'Alterar senha e sair' }));
  await screen.findByRole('alert');
  expect(onExpired).not.toHaveBeenCalled();
  for (const label of ['Senha atual', 'Nova senha', 'Confirmar nova senha']) expect((screen.getByLabelText(label) as HTMLInputElement).value).toBe('');
  const button = screen.getByRole('button', { name: 'Alterar senha e sair' });
  expect(button.getAttribute('aria-busy')).toBe('false');
  expect((button as HTMLButtonElement).disabled).toBe(false);
});
it('encerra sessão visual quando servidor rejeita sessão expirada', async () => {
  const { user, onExpired, onChanged } = setup();
  vi.mocked(changePassword).mockRejectedValue(new ApiError(401, 'Sessão inválida ou expirada'));
  await fill(user);
  await user.click(screen.getByRole('button', { name: 'Alterar senha e sair' }));
  await waitFor(() => expect(onExpired).toHaveBeenCalledOnce());
  expect(onChanged).not.toHaveBeenCalled();
});
