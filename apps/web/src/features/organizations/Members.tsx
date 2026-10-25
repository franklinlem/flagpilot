import { useEffect, useRef } from 'react';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useParams } from 'react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import type { Identity } from '../../api/auth';
import { addMember, getOrganization, listMembers, listCandidates, roles, roleNames } from '../../api/organizations';
import { MemberRow } from './MemberRow';
import { QueryFailure } from './QueryFailure';
const schema = z.object({ userId: z.uuid('Selecione um usuário.'), role: z.enum(roles) });
export function Members({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const { orgId = '' } = useParams();
  const client = useQueryClient();
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { heading.current?.focus(); }, [orgId]);
  const organization = useQuery({ queryKey: ['organization', identity.user.id, orgId], queryFn: ({ signal }) => getOrganization(orgId, signal) });
  const members = useInfiniteQuery({ queryKey: ['members', identity.user.id, orgId], initialPageParam: '', queryFn: ({ pageParam, signal }) => listMembers(orgId, pageParam, signal), getNextPageParam: page => page.nextCursor ?? undefined });
  const owner = organization.data?.role === 'Owner';
  const candidates = useInfiniteQuery({ queryKey: ['member-candidates', identity.user.id, orgId], enabled: owner, initialPageParam: '', queryFn: ({ pageParam, signal }) => listCandidates(orgId, pageParam, signal), getNextPageParam: page => page.nextCursor ?? undefined });
  async function refresh() { await Promise.all([client.invalidateQueries({ queryKey: ['member-candidates', identity.user.id, orgId] }), client.invalidateQueries({ queryKey: ['organization', identity.user.id, orgId] }), client.invalidateQueries({ queryKey: ['members', identity.user.id, orgId] }), client.invalidateQueries({ queryKey: ['organizations', identity.user.id] })]); }
  const { register, handleSubmit, reset, formState: { errors } } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { userId: '', role: 'Viewer' } });
  const add = useMutation({ mutationFn: (values: z.infer<typeof schema>) => addMember(orgId, values, identity.csrfToken), onSuccess: async () => { reset(); await refresh(); } });
  const error = organization.error ?? members.error;
  return <section className="account">
    <Link to="/">Voltar à conta e organizações</Link><h2 ref={heading} tabIndex={-1}>Membros da organização</h2>
    {error ? <QueryFailure error={error} retry={() => { void Promise.all([organization.refetch(), members.refetch()]); }} onExpired={onExpired} /> : null}
    {organization.isPending || members.isPending ? <p role="status">Carregando membros…</p> : null}
    {organization.data && !error ? <><h3>{organization.data.name}</h3><p className="hint">ID da organização: <code>{organization.data.id}</code></p><p>Seu papel: {roleNames[organization.data.role]}</p>
      {owner ? <form className="card" onSubmit={handleSubmit(values => add.mutate(values))} noValidate><fieldset disabled={add.isPending}><legend>Adicionar usuário existente</legend>
        <label htmlFor="member-user-id">Usuário existente</label><select id="member-user-id" {...register('userId')} disabled={candidates.isPending || candidates.isError} aria-invalid={Boolean(errors.userId)} aria-describedby={errors.userId ? 'member-candidates-hint member-id-error' : 'member-candidates-hint'}><option value="">Selecione um usuário</option>{candidates.data?.pages.flatMap(page => page.items).map(user => <option key={user.userId} value={user.userId}>{user.email}</option>)}</select>
        <p id="member-candidates-hint" className="hint">Usuários ativos de organizações às quais você tem acesso e antigos membros desta organização. Membros atuais não aparecem.</p>
        {candidates.isPending ? <p role="status">Carregando usuários…</p> : null}
        {candidates.error ? <QueryFailure error={candidates.error} retry={() => { void candidates.refetch(); }} onExpired={onExpired} /> : null}
        {candidates.data && !candidates.data.pages.some(page => page.items.length) ? <p>Nenhum usuário disponível para adicionar.</p> : null}
        {candidates.hasNextPage ? <button type="button" className="secondary" disabled={candidates.isFetchingNextPage} onClick={() => { void candidates.fetchNextPage(); }}>Carregar mais usuários</button> : null}
        {errors.userId ? <p id="member-id-error" className="field-error">{errors.userId.message}</p> : null}
        <label htmlFor="member-role">Papel do novo membro</label><select id="member-role" {...register('role')}>{roles.map(role => <option key={role} value={role}>{roleNames[role]}</option>)}</select><button type="submit" aria-busy={add.isPending} disabled={candidates.isPending || candidates.isError}>{add.isPending ? 'Adicionando…' : 'Adicionar membro'}</button>
      </fieldset>{add.error ? <QueryFailure error={add.error} buttonLabel="Fechar aviso" retry={() => add.reset()} onExpired={onExpired} /> : null}</form> : <p className="hint">Somente Proprietários podem gerenciar membros.</p>}
      <ul className="organizations">{members.data?.pages.flatMap(page => page.items).map(member => <MemberRow key={`${member.userId}:${member.role}`} member={member} owner={owner} csrf={identity.csrfToken} refresh={refresh} onExpired={onExpired} />)}</ul>
      {members.hasNextPage ? <button className="secondary" disabled={members.isFetchingNextPage} onClick={() => { void members.fetchNextPage(); }}>Carregar mais membros</button> : null}
    </> : null}
  </section>;
}
