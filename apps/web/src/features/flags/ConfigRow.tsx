import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import type { Identity } from '../../api/auth';
import { putConfig, type Config, type Snapshot } from '../../api/flags';
import { QueryFailure } from '../organizations/QueryFailure';

type Change = { enabled: boolean; rolloutBps: number };
// Conversão decimal exata: até duas casas, sem arredondamento de floats.
export function percentageBps(value: string): number | null {
  if (!/^(?:100(?:[.,]0{1,2})?|(?:[0-9]|[1-9][0-9])(?:[.,][0-9]{1,2})?)$/.test(value)) return null;
  const [whole, fraction = ''] = value.replace(',', '.').split('.');
  return Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
}
export function ConfigRow({ config, snapshot, etag, identity, refresh, onExpired }: { config: Config; snapshot: Snapshot; etag: string; identity: Identity; refresh: () => Promise<void>; onExpired: () => void }) {
  const [confirm, setConfirm] = useState<Change | null>(null);
  const [percentage, setPercentage] = useState(String(config.rolloutBps / 100));
  const [invalid, setInvalid] = useState(false);
  const bps = percentageBps(percentage);
  const save = useMutation({ mutationFn: ({ change, confirmed }: { change: Change; confirmed: boolean }) => putConfig(snapshot.environment.id, config.flagKey, { ...change, confirmProduction: confirmed }, identity.csrfToken, etag), onSuccess: refresh });
  function apply(change: Change) {
    if (snapshot.environment.isProduction) setConfirm(change);
    else save.mutate({ change, confirmed: false });
  }
  const action = config.enabled ? 'Desativar' : 'Ativar';
  const inputID = `percentage-${config.flagId}`;
  return <li className="card member"><strong>{config.flagKey}</strong><p>Estado: <span className="badge">{config.enabled ? 'Ativada' : 'Desativada'}</span></p>
    <p>Rollout: {config.rolloutBps / 100}%</p>
    <p className="hint">{!config.enabled || config.rolloutBps === 0 ? 'Nenhum contexto recebe esta flag.' : config.rolloutBps === 10000 ? 'Todos os contextos recebem esta flag.' : 'Distribuição estável por contexto. A avaliação exige uma bucketingKey.'}</p>
    {snapshot.role !== 'Viewer' ? <>{confirm ? <div className="confirmation" role="group" aria-label="Confirmar alteração em produção"><p>Alterar <strong>{config.flagKey}</strong> em <strong>{snapshot.projectName} / {snapshot.environment.name}</strong> ({snapshot.environment.slug}), ambiente de produção?</p><p>Estado: {config.enabled ? 'Ativada' : 'Desativada'} → {confirm.enabled ? 'Ativada' : 'Desativada'}. Rollout: {config.rolloutBps / 100}% → {confirm.rolloutBps / 100}%.</p><button disabled={save.isPending} aria-busy={save.isPending} onClick={() => save.mutate({ change: confirm, confirmed: true })}>{save.isPending ? 'Salvando…' : 'Confirmar alteração em produção'}</button><button className="secondary" disabled={save.isPending} onClick={() => setConfirm(null)}>Cancelar</button></div> : <>
      <button className="secondary" disabled={save.isPending} aria-busy={save.isPending} onClick={() => apply({ enabled: !config.enabled, rolloutBps: config.rolloutBps })}>{save.isPending ? 'Salvando…' : `${action} flag`}</button>
      <form noValidate onSubmit={event => { event.preventDefault(); if (bps === null) { setInvalid(true); return; } if (bps !== config.rolloutBps) apply({ enabled: config.enabled, rolloutBps: bps }); }}>
        <label htmlFor={inputID}>Porcentagem de {config.flagKey}</label>
        <input id={inputID} inputMode="decimal" value={percentage} disabled={save.isPending} aria-invalid={invalid} aria-describedby={invalid ? `${inputID}-error` : undefined} onChange={event => { setPercentage(event.target.value); setInvalid(false); }} />
        {invalid ? <p id={`${inputID}-error`} role="alert">Informe de 0 a 100, com até duas casas decimais.</p> : null}
        <button type="submit" disabled={save.isPending || bps === config.rolloutBps} aria-busy={save.isPending}>{save.isPending ? 'Salvando…' : 'Salvar porcentagem'}</button>
      </form>
    </>}</> : null}
    {save.error ? <QueryFailure error={save.error} onExpired={onExpired} buttonLabel="Recarregar configurações" retry={() => { save.reset(); void refresh(); }} /> : null}
  </li>;
}
