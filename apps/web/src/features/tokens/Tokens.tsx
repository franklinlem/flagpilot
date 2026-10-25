import { useState } from 'react';
import { Link, useParams } from 'react-router';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { Identity } from '../../api/auth';
import { listTokens, revokeToken, type Token } from '../../api/tokens';
import { IssuePanel } from './IssuePanel';
import { QueryFailure } from '../organizations/QueryFailure';
function TokenRow({ token, identity, refresh, onExpired }: { token: Token; identity: Identity; refresh: () => Promise<void>; onExpired: () => void }) {
  const [confirm, setConfirm] = useState(false);
  const revoke = useMutation({ mutationFn: () => revokeToken(token.environmentId, token.id, identity.csrfToken), onSuccess: refresh });
  const state = token.revokedAt ? 'Revogado' : Date.parse(token.expiresAt) <= Date.now() ? 'Expirado' : 'Válido';
  return <li className="card member"><strong>{token.name}</strong><p className="hint">Prefixo: <code>{token.prefix}</code></p><p>Validade: <time dateTime={token.expiresAt}>{new Date(token.expiresAt).toLocaleString('pt-BR')}</time> <span className="badge">{state}</span></p>{!token.revokedAt ? <>{confirm ? <div className="confirmation" role="group" aria-label="Confirmar revogação"><p>Revogar {token.name}? Aplicações que usam este token perderão acesso à avaliação.</p><button className="secondary danger" aria-busy={revoke.isPending} disabled={revoke.isPending} onClick={() => revoke.mutate()}>{revoke.isPending ? 'Revogando…' : 'Confirmar revogação'}</button><button className="secondary" disabled={revoke.isPending} onClick={() => setConfirm(false)}>Cancelar</button></div> : <button className="secondary danger" onClick={() => setConfirm(true)}>Revogar token</button>}</> : null}{revoke.error ? <QueryFailure error={revoke.error} onExpired={onExpired} retry={() => revoke.reset()} buttonLabel="Fechar aviso" /> : null}</li>;
}
export function Tokens({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const { envId = '' } = useParams();
  const client = useQueryClient();
  const [show, setShow] = useState(false);
  const key = ['tokens', identity.user.id, envId];
  const query = useInfiniteQuery({ queryKey: key, initialPageParam: '', queryFn: ({ signal, pageParam }) => listTokens(envId, pageParam, signal), getNextPageParam: page => page.nextCursor ?? undefined });
  const context = query.data?.pages[0];
  async function refresh() { await client.invalidateQueries({ queryKey: key }); }
  return <section className="account"><Link to="/">Conta e organizações</Link><h2>Tokens de avaliação</h2>{query.isPending ? <p role="status">Carregando tokens…</p> : null}{query.error ? <QueryFailure error={query.error} onExpired={onExpired} retry={() => { void query.refetch(); }} /> : null}
    {context && !query.error ? <><Link to={`/projects/${context.environment.projectId}/environments`}>Voltar aos ambientes</Link><h3>{context.projectName} / {context.environment.name}</h3><p><code>{context.environment.slug}</code> <span className={`badge${context.environment.isProduction ? ' production' : ''}`}>{context.environment.isProduction ? 'Produção' : 'Não produção'}</span></p><p>Tokens de servidor com acesso somente à avaliação deste ambiente.</p><div className="section-heading"><button className="secondary" onClick={() => setShow(value => !value)}>{show ? 'Fechar emissão' : 'Novo token'}</button></div>
      {show ? <IssuePanel environment={envId} csrf={identity.csrfToken} refresh={refresh} onExpired={onExpired} /> : null}
      <ul className="organizations">{query.data?.pages.flatMap(page => page.items).map(token => <TokenRow key={`${token.id}:${token.revokedAt}`} token={token} identity={identity} refresh={refresh} onExpired={onExpired} />)}</ul>
      {context.items.length === 0 ? <p>Nenhum token emitido neste ambiente.</p> : null}
      {query.hasNextPage ? <button className="secondary" disabled={query.isFetchingNextPage} onClick={() => { void query.fetchNextPage(); }}>Carregar mais tokens</button> : null}
    </> : null}
  </section>;
}
