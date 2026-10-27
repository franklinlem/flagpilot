import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { ApiError, changePassword, type Identity } from '../../api/auth';
import { PageHeading } from '../../components/PageHeading';

const withinBytes = (value: string) => new TextEncoder().encode(value).length <= 256;
const schema = z.object({
  currentPassword: z.string().min(1, 'Informe sua senha atual.').refine(withinBytes, 'Use até 256 bytes UTF-8.'),
  newPassword: z.string().refine(value => [...value].length >= 12 && value.trim() !== '', 'Use pelo menos 12 caracteres.').refine(withinBytes, 'Use até 256 bytes UTF-8.'),
  confirmation: z.string(),
}).refine(value => value.newPassword !== value.currentPassword, { path: ['newPassword'], message: 'Escolha uma senha diferente da atual.' })
  .refine(value => value.newPassword === value.confirmation, { path: ['confirmation'], message: 'As senhas não coincidem.' });
type Fields = z.infer<typeof schema>;
const empty = { currentPassword: '', newPassword: '', confirmation: '' };

export function ChangePassword({ identity, onChanged, onExpired }: { identity: Identity; onChanged: () => void; onExpired: () => void }) {
  const [error, setError] = useState('');
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => { controller.current?.abort(); }, []);
  const { register, handleSubmit, reset, setFocus, formState: { errors, isSubmitting } } = useForm<Fields>({ resolver: zodResolver(schema), defaultValues: empty });
  const submit = handleSubmit(async values => {
    setError('');
    const request = new AbortController(); controller.current = request;
    try {
      await changePassword({ currentPassword: values.currentPassword, newPassword: values.newPassword }, identity.csrfToken, request.signal);
      if (!request.signal.aborted) { reset(empty); onChanged(); }
    } catch (failure) {
      if (request.signal.aborted) return;
      reset(empty);
      if (failure instanceof ApiError && failure.status === 401) { onExpired(); return; }
      setError(failure instanceof ApiError && failure.status === 0 ? 'A conexão falhou. A senha pode ter sido alterada; tente entrar com a nova senha para conferir.' : failure instanceof Error ? failure.message : 'Não foi possível alterar a senha.');
      setFocus('currentPassword');
    } finally { if (controller.current === request) controller.current = null; }
  });
  const labels = { currentPassword: 'Senha atual', newPassword: 'Nova senha', confirmation: 'Confirmar nova senha' };
  return <section aria-label="Segurança da conta">
    <Link to="/">Voltar à conta e organizações</Link>
    <PageHeading>Segurança da conta</PageHeading>
    <p>Altere a senha de <strong>{identity.user.email}</strong>.</p>
    <p id="password-policy">Use pelo menos 12 caracteres e até 256 bytes UTF-8, com uma senha diferente da atual. Espaços são preservados.</p>
    <p id="password-sessions">Todas as suas sessões serão encerradas, inclusive esta. Depois, entre novamente com a nova senha.</p>
    <form className="card" noValidate onSubmit={submit}>
      <fieldset disabled={isSubmitting}>
        <legend>Alterar senha</legend>
        {(Object.keys(labels) as (keyof Fields)[]).map(name => <div key={name}>
          <label htmlFor={name}>{labels[name]}</label>
          <input id={name} type="password" autoComplete={name === 'currentPassword' ? 'current-password' : 'new-password'} required {...register(name)} aria-invalid={Boolean(errors[name])} aria-describedby={`password-policy${errors[name] ? ` ${name}-error` : ''}`} />
          {errors[name] ? <p id={`${name}-error`} className="field-error">{errors[name].message}</p> : null}
        </div>)}
        <button type="submit" aria-busy={isSubmitting} aria-describedby="password-sessions">{isSubmitting ? 'Alterando senha…' : 'Alterar senha e sair'}</button>
      </fieldset>
      {error ? <p role="alert" className="error">{error}</p> : null}
    </form>
  </section>;
}
