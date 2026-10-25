import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
const schema = z.object({ key: z.string().regex(/^[a-z][a-z0-9_.-]{1,79}$/, 'Use 2–80 caracteres: letras minúsculas, números, ponto, hífen ou sublinhado; comece com letra.'), description: z.string().max(2000, 'Use até 2000 caracteres.') });
export function FlagForm({ pending, submit }: { pending: boolean; submit: (values: z.infer<typeof schema>) => void }) {
  const { register, handleSubmit, formState: { errors } } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { key: '', description: '' } });
  return <form onSubmit={handleSubmit(values => submit(values))} noValidate><fieldset disabled={pending}><legend>Criar flag booleana</legend>
    <label htmlFor="flag-key">Chave da flag</label><input id="flag-key" maxLength={80} {...register('key')} aria-invalid={Boolean(errors.key)} aria-describedby={errors.key ? 'flag-key-error' : 'flag-key-hint'} /><p id="flag-key-hint" className="hint">A chave não poderá ser alterada nem reutilizada após arquivamento.</p>{errors.key ? <p id="flag-key-error" className="field-error">{errors.key.message}</p> : null}
    <label htmlFor="flag-description">Descrição</label><textarea id="flag-description" maxLength={2000} {...register('description')} aria-invalid={Boolean(errors.description)} />
    <p className="hint">A flag começará desativada em todos os ambientes deste projeto.</p><button type="submit" aria-busy={pending}>{pending ? 'Criando…' : 'Criar flag'}</button>
  </fieldset></form>;
}
