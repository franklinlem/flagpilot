import { useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Identity } from '../../api/auth';
import { roleNames } from '../../api/organizations';
import { createEnvironment, getProject, listEnvironments } from '../../api/projects';
import { QueryFailure } from '../organizations/QueryFailure';
import { CreateForm, type CreateValues } from './CreateForm';
export function Environments({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const { projectId = '' } = useParams();
  const client = useQueryClient();
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { heading.current?.focus(); }, [projectId]);
  const [showForm, setShowForm] = useState(false);
  const key = ['environments', identity.user.id, projectId];
  const project = useQuery({ queryKey: ['project', identity.user.id, projectId], queryFn: ({ signal }) => getProject(projectId, signal) });
  const environments = useInfiniteQuery({ queryKey: key, initialPageParam: '', queryFn: ({ signal, pageParam }) => listEnvironments(projectId, pageParam, signal), getNextPageParam: page => page.nextCursor ?? undefined });
  const create = useMutation({ mutationFn: (values: CreateValues) => createEnvironment(projectId, values, identity.csrfToken), onSuccess: async () => { await client.invalidateQueries({ queryKey: key }); setShowForm(false); } });
  const error = project.error ?? environments.error;
  const canManage = project.data?.role === 'Owner' || project.data?.role === 'Admin';
  return <section className="account"><Link to="/">Conta e organizações</Link><h2 tabIndex={-1} ref={heading}>Ambientes do projeto</h2>
    {error ? <QueryFailure error={error} onExpired={onExpired} retry={() => { void Promise.all([project.refetch(), environments.refetch()]); }} /> : null}
    {project.isPending || environments.isPending ? <p role="status">Carregando ambientes…</p> : null}
    {project.data && !error ? <><Link to={`/organizations/${project.data.organizationId}/projects`}>Voltar aos projetos</Link><h3>{project.data.name}</h3><p>Seu papel: {roleNames[project.data.role]}</p><Link to={`/projects/${projectId}/flags`}>Ver flags do projeto</Link>
      {canManage ? <div className="section-heading"><button className="secondary" onClick={() => { create.reset(); setShowForm(value => !value); }}>{showForm ? 'Fechar formulário' : 'Novo ambiente'}</button></div> : <p className="hint">Somente Proprietários e Administradores podem criar ambientes.</p>}
      {canManage && showForm ? <div className="card"><CreateForm environment pending={create.isPending} submit={values => create.mutate(values)} />{create.error ? <QueryFailure error={create.error} onExpired={onExpired} retry={() => create.reset()} buttonLabel="Fechar aviso" /> : null}</div> : null}
      <ul className="organizations">{environments.data?.pages.flatMap(page => page.items).map(environment => <li className="card" key={environment.id}><div><strong>{environment.name}</strong><p className="hint">{environment.slug}</p></div><span className={`badge${environment.isProduction ? ' production' : ''}`}>{environment.isProduction ? 'Produção' : 'Não produção'}</span><Link to={`/environments/${environment.id}/flags`}>Configurar flags</Link></li>)}</ul>
      {environments.data?.pages[0].items.length === 0 ? <p>Nenhum ambiente cadastrado.</p> : null}
      {environments.hasNextPage ? <button className="secondary" disabled={environments.isFetchingNextPage} aria-busy={environments.isFetchingNextPage} onClick={() => { void environments.fetchNextPage(); }}>Carregar mais ambientes</button> : null}
    </> : null}
  </section>;
}
