import { useState } from 'react';
import { Link, useParams } from 'react-router';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Identity } from '../../api/auth';
import { createFlag, listFlags } from '../../api/flags';
import { getProject } from '../../api/projects';
import { QueryFailure } from '../organizations/QueryFailure';
import { FlagForm } from './FlagForm';
export function Flags({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const { projectId = '' } = useParams();
  const client = useQueryClient();
  const [show, setShow] = useState(false);
  const key = ['flags', identity.user.id, projectId];
  const project = useQuery({ queryKey: ['project', identity.user.id, projectId], queryFn: ({ signal }) => getProject(projectId, signal) });
  const flags = useInfiniteQuery({ queryKey: key, initialPageParam: '', queryFn: ({ signal, pageParam }) => listFlags(projectId, pageParam, signal), getNextPageParam: page => page.nextCursor ?? undefined });
  const create = useMutation({ mutationFn: (values: { key: string; description: string }) => createFlag(projectId, values, identity.csrfToken), onSuccess: async () => { await Promise.all([client.invalidateQueries({ queryKey: key }), client.invalidateQueries({ queryKey: ['environments', identity.user.id, projectId] }), client.invalidateQueries({ queryKey: ['configs', identity.user.id] })]); setShow(false); } });
  const error = project.error ?? flags.error;
  const canManage = project.data && project.data.role !== 'Viewer';
  return <section className="account"><Link to={`/projects/${projectId}/environments`}>Voltar aos ambientes</Link><h2 tabIndex={-1}>Flags do projeto</h2>
    {error ? <QueryFailure error={error} onExpired={onExpired} retry={() => { void Promise.all([project.refetch(), flags.refetch()]); }} /> : null}
    {project.isPending || flags.isPending ? <p role="status">Carregando flags…</p> : null}
    {project.data && !error ? <><h3>{project.data.name}</h3>{canManage ? <div className="section-heading"><button className="secondary" onClick={() => { create.reset(); setShow(value => !value); }}>{show ? 'Fechar formulário' : 'Nova flag'}</button></div> : <p className="hint">Visualizadores podem consultar flags e configurações.</p>}
      {canManage && show ? <div className="card"><FlagForm pending={create.isPending} submit={values => create.mutate(values)} />{create.error ? <QueryFailure error={create.error} onExpired={onExpired} retry={() => create.reset()} buttonLabel="Fechar aviso" /> : null}</div> : null}
      <ul className="organizations">{flags.data?.pages.flatMap(page => page.items).map(flag => <li className="card" key={flag.id}><div><strong>{flag.key}</strong><p>{flag.description}</p></div><span className="badge">{flag.archivedAt ? 'Arquivada' : 'Ativa no catálogo'}</span><Link to={`/projects/${projectId}/flags/${encodeURIComponent(flag.key)}`}>Ver flag</Link></li>)}</ul>
      {flags.data?.pages[0].items.length === 0 ? <p>Nenhuma flag cadastrada.</p> : null}
      {flags.hasNextPage ? <button className="secondary" disabled={flags.isFetchingNextPage} onClick={() => { void flags.fetchNextPage(); }}>Carregar mais flags</button> : null}
    </> : null}
  </section>;
}
