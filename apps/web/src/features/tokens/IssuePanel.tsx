import { useEffect, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { issueToken } from '../../api/tokens';
import { QueryFailure } from '../organizations/QueryFailure';
const schema = z.object({ name: z.string().trim().min(1, 'Informe um nome.').max(200), days: z.enum(['1', '7', '30', '90', '365']) });
export function IssuePanel({ environment, csrf, refresh, onExpired }: { environment: string; csrf: string; refresh: () => Promise<void>; onExpired: () => void }) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [secret, setSecret] = useState<string | null>(null);
  const [copyMessage, setCopyMessage] = useState('');
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => { controller.current?.abort(); }, []);
  const { register, handleSubmit, reset, formState: { errors } } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { name: '', days: '90' } });
  async function submit(values: z.infer<typeof schema>) {
    const request = new AbortController(); controller.current = request; setPending(true); setError(null);
    try {
      const result = await issueToken(environment, { name: values.name, expiresAt: new Date(Date.now() + Number(values.days) * 86400000).toISOString() }, csrf, request.signal);
      if (!request.signal.aborted) { setSecret(result.secret); reset(); await refresh(); }
    } catch (error) { if (!request.signal.aborted) setError(error instanceof Error ? error : new Error('Não foi possível emitir o token.')); }
    finally { if (!request.signal.aborted) setPending(false); }
  }
  async function copy() { try { await navigator.clipboard.writeText(secret!); setCopyMessage('Token copiado.'); } catch { setCopyMessage('Selecione o valor e copie manualmente.'); } }
  if (secret) return <section className="card" aria-label="Token emitido"><h3>Copie seu token agora</h3><p>O segredo será exibido somente uma vez. Guarde-o no backend da aplicação. Não use em navegador público nem compartilhe em mensagens.</p><label htmlFor="issued-secret">Segredo do token</label><textarea id="issued-secret" readOnly value={secret} autoComplete="off" spellCheck={false} /><div className="member-actions"><button onClick={() => { void copy(); }}>Copiar token</button><button className="secondary" onClick={() => { setSecret(null); setCopyMessage(''); }}>Já guardei, ocultar segredo</button></div>{copyMessage ? <p role="status">{copyMessage}</p> : null}</section>;
  return <div className="card"><form onSubmit={handleSubmit(values => { void submit(values); })} noValidate><fieldset disabled={pending}><legend>Emitir token de avaliação</legend><label htmlFor="token-name">Nome do token</label><input id="token-name" maxLength={200} {...register('name')} aria-invalid={Boolean(errors.name)} aria-describedby={errors.name ? 'token-name-error' : undefined} />{errors.name ? <p id="token-name-error" className="field-error">{errors.name.message}</p> : null}<label htmlFor="token-days">Validade</label><select id="token-days" {...register('days')}>{['1', '7', '30', '90', '365'].map(value => <option key={value} value={value}>{value} {value === '1' ? 'dia' : 'dias'}</option>)}</select><p className="hint">Permite apenas avaliar flags deste ambiente. O segredo não poderá ser consultado novamente.</p><button type="submit" aria-busy={pending}>{pending ? 'Emitindo…' : 'Emitir token'}</button></fieldset></form>{error ? <QueryFailure error={error} onExpired={onExpired} retry={() => setError(null)} buttonLabel="Fechar aviso" /> : null}</div>;
}
