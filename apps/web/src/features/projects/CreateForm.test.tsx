// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateForm } from './CreateForm';
afterEach(cleanup);
it('valida nome e slug antes de criar projeto', async () => {
  const submit = vi.fn();
  const user = userEvent.setup();
  render(<CreateForm pending={false} submit={submit} />);
  await user.click(screen.getByRole('button', { name: 'Criar projeto' }));
  expect(await screen.findByText('Informe um nome.')).toBeTruthy();
  expect(submit).not.toHaveBeenCalled();
  await user.type(screen.getByLabelText('Nome do projeto'), ' Loja ');
  await user.type(screen.getByLabelText('Slug do projeto'), 'loja');
  await user.click(screen.getByRole('button', { name: 'Criar projeto' }));
  expect(submit.mock.calls[0]?.[0]).toEqual({ name: 'Loja', slug: 'loja', isProduction: false });
});
it('exige produção para slug reservado e aceita produção com slug personalizado', async () => {
  const submit = vi.fn();
  const user = userEvent.setup();
  render(<CreateForm environment pending={false} submit={submit} />);
  await user.type(screen.getByLabelText('Nome do ambiente'), 'Produção Europa');
  await user.type(screen.getByLabelText('Slug do ambiente'), 'production');
  await user.click(screen.getByRole('button', { name: 'Criar ambiente' }));
  expect(await screen.findByText('O slug production exige a identificação de produção.')).toBeTruthy();
  expect(submit).not.toHaveBeenCalled();
  await user.click(screen.getByLabelText('Este ambiente é de produção'));
  await user.clear(screen.getByLabelText('Slug do ambiente'));
  await user.type(screen.getByLabelText('Slug do ambiente'), 'eu-live');
  await user.click(screen.getByRole('button', { name: 'Criar ambiente' }));
  expect(submit.mock.calls[0]?.[0]).toEqual({ name: 'Produção Europa', slug: 'eu-live', isProduction: true });
});
