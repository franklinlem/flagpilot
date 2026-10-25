import type { AuditEvent } from '../../api/audit';
const names: Record<string,string> = { enabled: 'Ativada', rolloutBps: 'Rollout (%)', revision: 'Revisão', version: 'Versão', key: 'Chave', flagKey: 'Chave', description: 'Descrição', name: 'Nome', slug: 'Slug', role: 'Papel', archivedAt: 'Arquivada em', expiresAt: 'Validade', revokedAt: 'Revogado em', isProduction: 'Produção' };
function Details({ value }: { value: AuditEvent['before'] }) {
  return value ? <dl>{Object.entries(value).map(([key, raw]) => <div key={key}><dt>{names[key] ?? key}</dt><dd>{raw === null ? '—' : typeof raw === 'boolean' ? raw ? 'Sim' : 'Não' : key === 'rolloutBps' && typeof raw === 'number' ? `${raw / 100}%` : String(raw)}</dd></div>)}</dl> : <p>Sem campos disponíveis.</p>;
}
export function EventDetails({ event }: { event: Pick<AuditEvent, 'before' | 'after'> }) {
  return <details><summary>Ver alterações</summary><h4>Antes</h4><Details value={event.before} /><h4>Depois</h4><Details value={event.after} /></details>;
}
