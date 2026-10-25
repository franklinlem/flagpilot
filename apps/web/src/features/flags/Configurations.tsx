import { Link, useParams } from 'react-router';
import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query';
import type { Identity } from '../../api/auth';
import { listConfigs } from '../../api/flags';
import { ConfigRow } from './ConfigRow';
import { QueryFailure } from '../organizations/QueryFailure';
export function Configurations({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const { envId = '' } = useParams();
  const client = useQueryClient();
  const key = ['configs', identity.user.id, envId];
  const query = useInfiniteQuery({ queryKey: key, initialPageParam: '', queryFn: ({ signal, pageParam }) => listConfigs(envId, pageParam, signal), getNextPageParam: page => page.data.nextCursor ?? undefined, refetchOnWindowFocus: false });
  const snapshot = query.data?.pages[0].data;
  async function refresh() { await Promise.all([client.invalidateQueries({ queryKey: key }), client.invalidateQueries({ queryKey: ['environments', identity.user.id, snapshot?.environment.projectId] })]); }
  return <section className="account"><Link to="/">Conta e organizações</Link><h2>Flags do ambiente</h2>
    {query.isPending ? <p role="status">Carregando configurações…</p> : null}
    {query.error ? <QueryFailure error={query.error} onExpired={onExpired} retry={() => { void query.refetch(); }} /> : null}
    {snapshot && !query.error ? <><Link to={`/projects/${snapshot.environment.projectId}/environments`}>Voltar aos ambientes</Link><h3>{snapshot.projectName} / {snapshot.environment.name}</h3><p><code>{snapshot.environment.slug}</code> <span className={`badge${snapshot.environment.isProduction ? ' production' : ''}`}>{snapshot.environment.isProduction ? 'Produção' : 'Não produção'}</span></p><Link to={`/projects/${snapshot.environment.projectId}/flags`}>Gerenciar flags do projeto</Link>
      {snapshot.environment.isProduction ? <p className="hint">Toda alteração neste ambiente exige confirmação explícita.</p> : null}
      <ul className="organizations">{query.data?.pages.flatMap(page => page.data.items.map(config => <ConfigRow key={`${config.flagId}:${config.version}:${config.revision}`} config={config} snapshot={page.data} etag={page.etag} identity={identity} refresh={refresh} onExpired={onExpired} />))}</ul>
      {snapshot.items.length === 0 ? <p>Nenhuma flag ativa no catálogo deste projeto. Crie uma flag para configurar os ambientes.</p> : null}
      {query.hasNextPage ? <button className="secondary" disabled={query.isFetchingNextPage} onClick={() => { void query.fetchNextPage(); }}>Carregar mais configurações</button> : null}
    </> : null}
  </section>;
}
