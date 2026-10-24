// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ApiError, getIdentity, login, logout, type Identity } from '../api/auth';
import { App } from './App';
vi.mock('../api/auth', async importOriginal => {
  const original = await importOriginal<typeof import('../api/auth')>();
  return { ...original, getIdentity: vi.fn(), login: vi.fn(), logout: vi.fn() };
});
const identity: Identity = {
  user: { id: '00000000-0000-4000-8000-000000000001', email: 'owner@example.invalid', createdAt: '2026-10-24T12:00:00Z' },
  memberships: [{ organizationId: '00000000-0000-4000-8000-000000000002', userId: '00000000-0000-4000-8000-000000000001', role: 'Owner', organizationName: 'Minha equipe' }],
  csrfToken: 'a'.repeat(64),
};
beforeEach(() => { vi.mocked(getIdentity).mockRejectedValue(new ApiError(401, 'Sessão inválida')); });
afterEach(() => { cleanup(); vi.resetAllMocks(); });
it('valida formulário, trata credenciais inválidas e permite entrar/sair', async () => {
  const user = userEvent.setup();
  vi.mocked(login).mockRejectedValueOnce(new ApiError(401, 'Email ou senha inválidos')).mockResolvedValue(identity);
  vi.mocked(logout).mockResolvedValue();
  render(<App />);
  await user.click(await screen.findByRole('button', { name: 'Entrar' }));
  await screen.findByText('Informe um email válido.');
  expect(login).not.toHaveBeenCalled();
  await user.type(screen.getByLabelText('Email'), identity.user.email);
  await user.type(screen.getByLabelText('Senha'), 'test-password-123');
  await user.click(screen.getByRole('button', { name: 'Entrar' }));
  await screen.findByRole('alert');
  expect((screen.getByLabelText('Senha') as HTMLInputElement).value).toBe('');
  await user.type(screen.getByLabelText('Senha'), 'test-password-123');
  await user.click(screen.getByRole('button', { name: 'Entrar' }));
  await screen.findByText('Minha equipe');
  expect(screen.getByText('Proprietário')).toBeTruthy();
  await user.click(screen.getByRole('button', { name: 'Sair' }));
  await screen.findByRole('button', { name: 'Entrar' });
  expect(logout).toHaveBeenCalledWith(identity.csrfToken);
});
it('permite recuperar conexão sem mostrar formulário falso', async () => {
  const user = userEvent.setup();
  vi.mocked(getIdentity).mockRejectedValueOnce(new ApiError(503, 'Offline')).mockRejectedValueOnce(new ApiError(401, 'Sem sessão'));
  render(<App />);
  await screen.findByText('Serviço indisponível');
  expect(screen.queryByRole('button', { name: 'Entrar' })).toBeNull();
  await user.click(screen.getByRole('button', { name: 'Tentar novamente' }));
  await screen.findByRole('button', { name: 'Entrar' });
});
it('restaura identidade e mantém sessão visível quando logout falha', async () => {
  const user = userEvent.setup();
  vi.mocked(getIdentity).mockResolvedValue(identity);
  vi.mocked(logout).mockRejectedValue(new ApiError(503, 'Serviço indisponível'));
  render(<App />);
  await user.click(await screen.findByRole('button', { name: 'Sair' }));
  await screen.findByRole('alert');
  expect(screen.getByText('Minha equipe')).toBeTruthy();
});
