import { useEffect, useRef } from 'react';
import { Link, useParams, useSearchParams } from 'react-router';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import type { Identity } from '../../api/auth';
import { listAudit } from '../../api/audit';
import { getOrganization } from '../../api/organizations';
import { QueryFailure } from '../organizations/QueryFailure';
import { EventDetails } from './EventDetails';
const actions = ['flag.created','flag.updated','flag.archived','flag.config_updated','token.issued','token.revoked','project.created','environment.created','organization.created','member.added','member.removed','member.role_changed','user.provisioned','auth.login','auth.logout','auth.password_reset','auth.password_changed','installation.bootstrap'];
const resources = ['flag','flag_config','api_token','project','environment','organization','membership','user'];
export function Audit({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const { orgId = '' } = useParams();
  const heading = useRef<HTMLHeadingElement>(null);
  const [params, setParams] = useSearchParams();
  const filter = { action: params.get('action') ?? '', resourceType: params.get('resourceType') ?? '' };
  useEffect(() => { heading.current?.focus(); }, [filter.action, filter.resourceType]);
  const org = useQuery({ queryKey: ['organization',identity.user.id,orgId], queryFn: ({ signal }) => getOrganization(orgId,signal) });
  const query = useInfiniteQuery({ queryKey: ['audit',identity.user.id,orgId,filter], initialPageParam: '', queryFn: ({ signal, pageParam }) => listAudit(orgId,filter,pageParam,signal), getNextPageParam: page => page.nextCursor ?? undefined, refetchOnWindowFocus: false });
  const error = org.error ?? query.error;
  return <section className="account"><Link to="/">Conta e organizações</Link><h2 id="audit-title" ref={heading} tabIndex={-1}>Histórico de auditoria</h2>
    {org.isPending || query.isPending ? <p role="status">Carregando histórico…</p> : null}
    {error ? <QueryFailure error={error} onExpired={onExpired} retry={() => { void Promise.all([org.refetch(),query.refetch()]); }} /> : null}
    {org.data && !org.error ? <><h3>{org.data.name}</h3><Link to={`/organizations/${orgId}/projects`}>Ver projetos</Link><p className="hint">Eventos mais recentes primeiro. Detalhes exibem somente campos conhecidos; não permitem restaurar versões.</p>
      <form key={JSON.stringify(filter)} className="card filters" aria-label="Filtrar auditoria" onSubmit={event => { event.preventDefault(); const data = new FormData(event.currentTarget); setParams({ action: String(data.get('action') ?? ''), resourceType: String(data.get('resourceType') ?? '') }); }}>
        <label htmlFor="audit-action">Ação</label><select id="audit-action" name="action" defaultValue={filter.action}><option value="">Todas as ações</option>{actions.map(action => <option key={action}>{action}</option>)}</select>
        <label htmlFor="audit-resource">Tipo de recurso</label><select id="audit-resource" name="resourceType" defaultValue={filter.resourceType}><option value="">Todos os recursos</option>{resources.map(resource => <option key={resource}>{resource}</option>)}</select>
        <button type="submit">Aplicar filtros</button><button type="button" className="secondary" onClick={() => setParams({})}>Limpar filtros</button>
      </form>
      <button className="secondary" disabled={query.isFetching} aria-busy={query.isFetching} onClick={() => { void query.refetch(); }}>Atualizar histórico</button>
      {query.data && !query.error ? <p role="status">{query.data.pages.reduce((total,page) => total+page.items.length,0)} eventos carregados{!query.error && query.hasNextPage ? ' (há mais resultados)' : ''}.</p> : null}
      <ul className="organizations">{!query.error && query.data?.pages.flatMap(page => page.items).map(event => <li className="card member audit-event" key={event.id}><strong>{event.action}</strong><p><time dateTime={event.createdAt}>{new Date(event.createdAt).toLocaleString('pt-BR')}</time></p><p>Recurso: {event.resourceType} · <code>{event.resourceId}</code></p><p>Ator: {event.actorLabel === 'local-operator' ? 'Operador local' : event.actorLabel === 'user' ? 'Usuário' : 'Não identificado'} {event.actorId ? <code>{event.actorId}</code> : null}</p><EventDetails event={event} /></li>)}</ul>
      {!query.error && query.data?.pages[0].items.length === 0 ? <p>Nenhum evento encontrado para os filtros atuais.</p> : null}
      {!query.error && query.hasNextPage ? <button className="secondary" disabled={query.isFetchingNextPage} aria-busy={query.isFetchingNextPage} onClick={() => { void query.fetchNextPage(); }}>Carregar mais eventos</button> : null}
      <p><a href="#audit-title">Voltar ao topo do histórico</a></p>
    </> : null}
  </section>;
}
