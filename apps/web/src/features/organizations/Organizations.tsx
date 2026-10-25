import { useState } from 'react';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import type { Identity } from '../../api/auth';
import { createOrganization, listOrganizations, roleNames } from '../../api/organizations';
import { QueryFailure } from './QueryFailure';
const schema = z.object({ name: z.string().trim().min(1, 'Informe um nome.').max(200, 'Use até 200 caracteres.'), slug: z.string().regex(/^[a-z][a-z0-9-]{1,62}$/, 'Use 2–63 caracteres: letras minúsculas, números e hífen; comece com letra.') });
export function Organizations({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const client = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const { register, handleSubmit, reset, formState: { errors } } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema) });
  const query = useInfiniteQuery({ queryKey: ['organizations', identity.user.id], initialPageParam: '', queryFn: ({ pageParam, signal }) => listOrganizations(pageParam, signal), getNextPageParam: page => page.nextCursor ?? undefined });
  const create = useMutation({ mutationFn: (values: z.infer<typeof schema>) => createOrganization(values, identity.csrfToken), onSuccess: async () => { await client.invalidateQueries({ queryKey: ['organizations', identity.user.id] }); reset(); setShowForm(false); } });
  return <section aria-labelledby="organizations-title">
    <div className="section-heading"><h3 id="organizations-title">Suas organizações</h3><button className="secondary" onClick={() => { create.reset(); setShowForm(value => !value); }}>{showForm ? 'Fechar formulário' : 'Nova organização'}</button></div>
    {showForm ? <form className="card" onSubmit={handleSubmit(values => create.mutate(values))} noValidate>
      <fieldset disabled={create.isPending}><legend>Criar organização</legend>
        <label htmlFor="organization-name">Nome da organização</label><input id="organization-name" maxLength={200} {...register('name')} aria-invalid={Boolean(errors.name)} aria-describedby={errors.name ? 'org-name-error' : undefined} />
        {errors.name ? <p className="field-error" id="org-name-error">{errors.name.message}</p> : null}
        <label htmlFor="organization-slug">Slug da organização</label><input id="organization-slug" maxLength={63} {...register('slug')} aria-invalid={Boolean(errors.slug)} aria-describedby={errors.slug ? 'org-slug-error' : undefined} />
        {errors.slug ? <p className="field-error" id="org-slug-error">{errors.slug.message}</p> : null}
        <button type="submit">{create.isPending ? 'Criando…' : 'Criar organização'}</button>
      </fieldset>{create.error ? <QueryFailure error={create.error} buttonLabel="Fechar aviso" retry={() => create.reset()} onExpired={onExpired} /> : null}
    </form> : null}
    {query.isPending ? <p role="status">Carregando organizações…</p> : null}
    {query.error ? <QueryFailure error={query.error} retry={() => { void query.refetch(); }} onExpired={onExpired} /> : null}
    {query.data ? <><ul className="organizations">{query.data.pages.flatMap(page => page.items).map(org => <li className="card" key={org.id}><div><strong>{org.name}</strong><p className="hint">{org.slug}</p></div><span className="badge">{roleNames[org.role]}</span><Link to={`/organizations/${org.id}/projects`}>Ver projetos</Link><Link to={`/organizations/${org.id}/members`}>Ver membros</Link></li>)}</ul>
      {query.data.pages[0].items.length === 0 ? <p>Você ainda não faz parte de uma organização.</p> : null}
      {query.hasNextPage ? <button className="secondary" disabled={query.isFetchingNextPage} onClick={() => { void query.fetchNextPage(); }}>Carregar mais organizações</button> : null}
    </> : null}
  </section>;
}
