import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { login, type Identity } from '../../api/auth';

const schema = z.object({
  email: z.email('Informe um email válido.'),
  password: z.string().min(1, 'Informe sua senha.').refine(value => new TextEncoder().encode(value).length <= 256, 'A senha excede o limite permitido.'),
});
type Fields = z.infer<typeof schema>;

export function LoginForm({ onLogin }: { onLogin: (identity: Identity) => void }) {
  const [error, setError] = useState('');
  const { register, handleSubmit, resetField, setFocus, formState: { errors, isSubmitting } } = useForm<Fields>({ resolver: zodResolver(schema) });
  useEffect(() => { setFocus('email'); }, [setFocus]);
  const submit = handleSubmit(async values => {
    setError('');
    try { onLogin(await login(values.email, values.password)); }
    catch (failure) { setError(failure instanceof Error ? failure.message : 'Não foi possível entrar.'); resetField('password'); }
  });
  return (
    <section className="card login-card" aria-labelledby="login-title">
      <h2 id="login-title">Entre na sua conta</h2>
      <p className="muted">Acesse sua organização no FlagPilot.</p>
      <form onSubmit={submit} noValidate>
        <fieldset disabled={isSubmitting}>
          <label htmlFor="email">Email</label>
          <input id="email" type="email" autoComplete="username" maxLength={254} required aria-invalid={Boolean(errors.email)} aria-describedby={errors.email ? 'email-error' : undefined} {...register('email')} />
          {errors.email ? <p className="field-error" id="email-error">{errors.email.message}</p> : null}
          <label htmlFor="password">Senha</label>
          <input id="password" type="password" autoComplete="current-password" required aria-invalid={Boolean(errors.password)} aria-describedby={errors.password ? 'password-error' : undefined} {...register('password')} />
          {errors.password ? <p className="field-error" id="password-error">{errors.password.message}</p> : null}
          <button type="submit">{isSubmitting ? 'Entrando…' : 'Entrar'}</button>
        </fieldset>
        {error ? <p role="alert" className="error">{error}</p> : null}
      </form>
      <p className="hint">Primeiro acesso? Utilize a conta criada pelo administrador da instalação.</p>
    </section>
  );
}
