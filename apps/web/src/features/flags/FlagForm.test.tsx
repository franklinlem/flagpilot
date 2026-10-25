// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FlagForm } from './FlagForm';
afterEach(cleanup);
it('valida chave antes de criar e informa o estado inicial desativado', async () => {
  const submit = vi.fn();
  const user = userEvent.setup();
  render(<FlagForm pending={false} submit={submit} />);
  await user.type(screen.getByLabelText('Chave da flag'), 'Invalid Key');
  await user.click(screen.getByRole('button', { name: 'Criar flag' }));
  expect(await screen.findByText(/Use 2–80 caracteres/)).toBeTruthy();
  expect(submit).not.toHaveBeenCalled();
  await user.clear(screen.getByLabelText('Chave da flag'));
  await user.type(screen.getByLabelText('Chave da flag'), 'checkout.new');
  await user.type(screen.getByLabelText('Descrição'), 'Nova experiência');
  await user.click(screen.getByRole('button', { name: 'Criar flag' }));
  expect(submit).toHaveBeenCalledWith({ key: 'checkout.new', description: 'Nova experiência' });
  expect(screen.getByText('A flag começará desativada em todos os ambientes deste projeto.')).toBeTruthy();
});
