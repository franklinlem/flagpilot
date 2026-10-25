import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
export type CreateValues = { name: string; slug: string; isProduction: boolean };
export function CreateForm({ environment = false, pending, submit }: { environment?: boolean; pending: boolean; submit: (values: CreateValues) => void }) {
  const kind = environment ? 'ambiente' : 'projeto';
  const schema = z.object({ name: z.string().trim().min(1, 'Informe um nome.').max(200, 'Use até 200 caracteres.'), slug: z.string().regex(/^[a-z][a-z0-9-]{1,62}$/, 'Use 2–63 caracteres: letras minúsculas, números e hífen; comece com letra.'), isProduction: z.boolean() }).refine(value => !environment || value.slug !== 'production' || value.isProduction, { path: ['isProduction'], message: 'O slug production exige a identificação de produção.' });
  const { register, handleSubmit, formState: { errors } } = useForm<CreateValues>({ resolver: zodResolver(schema), defaultValues: { name: '', slug: '', isProduction: false } });
  return <form onSubmit={handleSubmit(submit)} noValidate><fieldset disabled={pending}><legend>Criar {kind}</legend>
    <label htmlFor="resource-name">Nome do {kind}</label><input id="resource-name" maxLength={200} {...register('name')} aria-invalid={Boolean(errors.name)} aria-describedby={errors.name ? 'resource-name-error' : undefined} />
    {errors.name ? <p id="resource-name-error" className="field-error">{errors.name.message}</p> : null}
    <label htmlFor="resource-slug">Slug do {kind}</label><input id="resource-slug" maxLength={63} {...register('slug')} aria-invalid={Boolean(errors.slug)} aria-describedby={errors.slug ? 'resource-slug-hint resource-slug-error' : 'resource-slug-hint'} />
    <p id="resource-slug-hint" className="hint">Identificador único neste {environment ? 'projeto' : 'escopo organizacional'}. Nome e slug não poderão ser editados nesta etapa.</p>
    {errors.slug ? <p id="resource-slug-error" className="field-error">{errors.slug.message}</p> : null}
    {environment ? <><label className="checkbox-label"><input type="checkbox" {...register('isProduction')} aria-invalid={Boolean(errors.isProduction)} aria-describedby={errors.isProduction ? 'production-error' : 'production-hint'} />Este ambiente é de produção</label><p id="production-hint" className="hint">Marque para qualquer ambiente que atenda usuários reais. A identificação fica registrada permanentemente.</p>{errors.isProduction ? <p id="production-error" className="field-error">{errors.isProduction.message}</p> : null}</> : <p className="hint">Serão criados os ambientes development, staging e production, separados entre si.</p>}
    <button type="submit" aria-busy={pending}>{pending ? 'Criando…' : `Criar ${kind}`}</button>
  </fieldset></form>;
}
