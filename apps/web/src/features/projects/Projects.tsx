import { useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Identity } from '../../api/auth';
import { getOrganization, roleNames } from '../../api/organizations';
import { createProject, listProjects } from '../../api/projects';
import { QueryFailure } from '../organizations/QueryFailure';
import { CreateForm, type CreateValues } from './CreateForm';
export function Projects({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const { orgId = '' } = useParams();
  const client = useQueryClient();
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { heading.current?.focus(); }, [orgId]);
  const [showForm, setShowForm] = useState(false);
  const key = ['projects', identity.user.id, orgId];
  const organization = useQuery({ queryKey: ['organization', identity.user.id, orgId], queryFn: ({ signal }) => getOrganization(orgId, signal) });
  const projects = useInfiniteQuery({ queryKey: key, initialPageParam: '', queryFn: ({ signal, pageParam }) => listProjects(orgId, pageParam, signal), getNextPageParam: page => page.nextCursor ?? undefined });
  const create = useMutation({ mutationFn: ({ name, slug }: CreateValues) => createProject(orgId, { name, slug }, identity.csrfToken), onSuccess: async () => { await client.invalidateQueries({ queryKey: key }); setShowForm(false); } });
  const error = organization.error ?? projects.error;
  const canManage = organization.data?.role === 'Owner' || organization.data?.role === 'Admin';
  return <section className="account"><Link to="/">Voltar à conta e organizações</Link><h2 tabIndex={-1} ref={heading}>Projetos da organização</h2>
    {error ? <QueryFailure error={error} onExpired={onExpired} retry={() => { void Promise.all([organization.refetch(), projects.refetch()]); }} /> : null}
    {organization.isPending || projects.isPending ? <p role="status">Carregando projetos…</p> : null}
    {organization.data && !error ? <><h3>{organization.data.name}</h3><p>Seu papel: {roleNames[organization.data.role]}</p><Link to={`/organizations/${orgId}/members`}>Ver membros</Link>
      {canManage ? <div className="section-heading"><button className="secondary" onClick={() => { create.reset(); setShowForm(value => !value); }}>{showForm ? 'Fechar formulário' : 'Novo projeto'}</button></div> : <p className="hint">Somente Proprietários e Administradores podem criar projetos.</p>}
      {canManage && showForm ? <div className="card"><CreateForm pending={create.isPending} submit={values => create.mutate(values)} />{create.error ? <QueryFailure error={create.error} onExpired={onExpired} retry={() => create.reset()} buttonLabel="Fechar aviso" /> : null}</div> : null}
      <ul className="organizations">{projects.data?.pages.flatMap(page => page.items).map(project => <li className="card" key={project.id}><div><strong>{project.name}</strong><p className="hint">{project.slug}</p></div><Link to={`/projects/${project.id}/environments`}>Ver ambientes</Link></li>)}</ul>
      {projects.data?.pages[0].items.length === 0 ? <p>Nenhum projeto cadastrado nesta organização.</p> : null}
      {projects.hasNextPage ? <button className="secondary" disabled={projects.isFetchingNextPage} aria-busy={projects.isFetchingNextPage} onClick={() => { void projects.fetchNextPage(); }}>Carregar mais projetos</button> : null}
    </> : null}
  </section>;
}
