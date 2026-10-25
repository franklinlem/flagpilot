import { PageHeading } from '../../components/PageHeading';
import { useState } from 'react';
import { Link, useParams } from 'react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Identity } from '../../api/auth';
import { changeFlag, getFlag, type Flag } from '../../api/flags';
import { getProject } from '../../api/projects';
import { QueryFailure } from '../organizations/QueryFailure';
function Editor({ flag, etag, identity, refresh, onExpired }: { flag: Flag; etag: string; identity: Identity; refresh: () => Promise<void>; onExpired: () => void }) {
  const [description, setDescription] = useState(flag.description);
  const [confirm, setConfirm] = useState(false);
  const save = useMutation({ mutationFn: (archive: boolean) => changeFlag(flag.projectId, flag.key, archive ? { archived: true, confirmProduction: true } : { description }, identity.csrfToken, etag), onSuccess: refresh });
  return <div className="card"><form onSubmit={event => { event.preventDefault(); save.mutate(false); }}><fieldset disabled={save.isPending}><label htmlFor="edit-description">Descrição da flag</label><textarea id="edit-description" maxLength={2000} value={description} onChange={event => setDescription(event.target.value)} /><button type="submit" aria-busy={save.isPending} disabled={description === flag.description}>{save.isPending ? 'Salvando…' : 'Salvar descrição'}</button></fieldset></form>
    {confirm ? <div className="confirmation" role="group" aria-label="Confirmar arquivamento"><p>Arquivar <strong>{flag.key}</strong> remove a flag das configurações de todos os ambientes deste projeto, inclusive produção. A chave continuará reservada.</p><button className="danger secondary" disabled={save.isPending} aria-busy={save.isPending} onClick={() => save.mutate(true)}>Confirmar arquivamento</button><button className="secondary" disabled={save.isPending} onClick={() => setConfirm(false)}>Cancelar</button></div> : <button className="secondary danger" disabled={save.isPending} onClick={() => setConfirm(true)}>Arquivar flag</button>}
    {save.error ? <QueryFailure error={save.error} onExpired={onExpired} buttonLabel="Recarregar flag" retry={() => { save.reset(); void refresh(); }} /> : null}
  </div>;
}
export function FlagDetail({ identity, onExpired }: { identity: Identity; onExpired: () => void }) {
  const { projectId = '', flagKey = '' } = useParams();
  const client = useQueryClient();
  const project = useQuery({ queryKey: ['project', identity.user.id, projectId], queryFn: ({ signal }) => getProject(projectId, signal) });
  const flag = useQuery({ queryKey: ['flag', identity.user.id, projectId, flagKey], queryFn: ({ signal }) => getFlag(projectId, flagKey, signal), refetchOnWindowFocus: false });
  async function refresh() { await Promise.all([client.invalidateQueries({ queryKey: ['flag', identity.user.id, projectId, flagKey] }), client.invalidateQueries({ queryKey: ['flags', identity.user.id, projectId] }), client.invalidateQueries({ queryKey: ['configs', identity.user.id] }), client.invalidateQueries({ queryKey: ['environments', identity.user.id, projectId] })]); }
  const error = project.error ?? flag.error;
  return <section className="account"><Link to={`/projects/${projectId}/flags`}>Voltar às flags</Link><PageHeading>Detalhes da flag</PageHeading>
    {error ? <QueryFailure error={error} onExpired={onExpired} retry={() => { void Promise.all([project.refetch(), flag.refetch()]); }} /> : null}
    {flag.isPending || project.isPending ? <p role="status">Carregando flag…</p> : null}
    {project.data && flag.data && !error ? <><h3>{project.data.name} / {flag.data.data.key}</h3>{flag.data.data.archivedAt ? <><p className="badge">Arquivada</p><p>{flag.data.data.description}</p></> : project.data.role !== 'Viewer' ? <Editor key={flag.data.etag} flag={flag.data.data} etag={flag.data.etag} identity={identity} refresh={refresh} onExpired={onExpired} /> : <p>{flag.data.data.description}</p>}</> : null}
  </section>;
}
