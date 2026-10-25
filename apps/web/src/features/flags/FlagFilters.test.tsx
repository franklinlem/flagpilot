// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FlagFilters } from './FlagFilters';
afterEach(cleanup);
it('aplica busca literal normalizada e estado; rejeita curinga e limpa filtros', async () => {
  const apply = vi.fn(); const user = userEvent.setup();
  render(<FlagFilters value={{ q: '', status: 'all' }} apply={apply} />);
  const input = screen.getByLabelText('Buscar chave');
  await user.type(input, '%'); await user.click(screen.getByRole('button',{name:'Aplicar filtros'}));
  expect(await screen.findByRole('alert')).toBeTruthy(); expect(apply).not.toHaveBeenCalled();
  await user.clear(input); await user.type(input, ' NEW_CHECKOUT '); await user.selectOptions(screen.getByLabelText('Estado no catálogo'),'archived');
  await user.click(screen.getByRole('button',{name:'Aplicar filtros'}));
  expect(apply).toHaveBeenCalledWith({q:'new_checkout',status:'archived'});
  await user.click(screen.getByRole('button',{name:'Limpar filtros'})); expect(apply).toHaveBeenLastCalledWith({q:'',status:'all'});
});
