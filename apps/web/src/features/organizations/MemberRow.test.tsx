// @vitest-environment jsdom
import { useState } from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { changeRole, type Member } from '../../api/organizations';
import { MemberRow } from './MemberRow';
vi.mock('../../api/organizations', async original => ({ ...await original<typeof import('../../api/organizations')>(), changeRole: vi.fn() }));
afterEach(() => { cleanup(); vi.resetAllMocks(); });
it('encerra o estado ocupado quando salvar termina e o botão fica sem alterações', async () => {
  const initial: Member = { organizationId: 'org', userId: 'user', email: 'member@example.invalid', role: 'Viewer', disabled: false };
  vi.mocked(changeRole).mockResolvedValue({ ...initial, role: 'Editor' });
  function Harness() {
    const [member, setMember] = useState(initial);
    return <MemberRow key={member.role} member={member} owner csrf="test" refresh={async () => { setMember({ ...initial, role: 'Editor' }); }} onExpired={() => {}} />;
  }
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  render(<QueryClientProvider client={client}><Harness /></QueryClientProvider>);
  const user = userEvent.setup();
  await user.selectOptions(screen.getByLabelText(`Papel de ${initial.email}`), 'Editor');
  await user.click(screen.getByRole('button', { name: 'Salvar papel' }));
  await waitFor(() => expect(changeRole).toHaveBeenCalledOnce());
  await waitFor(() => {
    const button = screen.getByRole('button', { name: 'Salvar papel' }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    expect(button.getAttribute('aria-busy')).toBe('false');
  });
  client.clear();
});
it('indica salvamento somente durante a operação e libera controles após falha', async () => {
  let rejectSave!: (error: Error) => void;
  vi.mocked(changeRole).mockImplementation(() => new Promise((_resolve, reject) => { rejectSave = reject; }));
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  render(<QueryClientProvider client={client}><MemberRow member={{ organizationId: 'org', userId: 'user', email: 'member@example.invalid', role: 'Viewer', disabled: false }} owner csrf="test" refresh={async () => {}} onExpired={() => {}} /></QueryClientProvider>);
  const user = userEvent.setup();
  await user.selectOptions(screen.getByLabelText('Papel de member@example.invalid'), 'Editor');
  await user.click(screen.getByRole('button', { name: 'Salvar papel' }));
  const pending = await screen.findByRole('button', { name: 'Salvando…' });
  expect(pending.getAttribute('aria-busy')).toBe('true');
  expect((pending as HTMLButtonElement).disabled).toBe(true);
  rejectSave(new Error('offline'));
  await screen.findByRole('alert');
  const ready = screen.getByRole('button', { name: 'Salvar papel' });
  expect(ready.getAttribute('aria-busy')).toBe('false');
  expect((ready as HTMLButtonElement).disabled).toBe(false);
  client.clear();
});
