import { useState } from 'react';
export function FlagFilters({ value, apply }: { value: { q: string; status: string }; apply: (filter: { q: string; status: string }) => void }) {
  const [q, setQ] = useState(value.q);
  const [status, setStatus] = useState(value.status);
  const [error, setError] = useState('');
  return <form className="card filters" noValidate aria-label="Filtrar flags" onSubmit={event => { event.preventDefault(); const query = q.trim().toLowerCase(); if (query && !/^[a-z0-9_.-]{1,80}$/.test(query)) { setError('Use até 80 caracteres da chave: letras, números, ponto, hífen ou sublinhado.'); return; } setError(''); apply({ q: query, status }); }}>
    <label htmlFor="flag-search">Buscar chave</label><input id="flag-search" maxLength={80} value={q} onChange={event => { setQ(event.target.value); setError(''); }} aria-invalid={Boolean(error)} aria-describedby={error ? 'flag-search-error' : undefined} />
    {error ? <p role="alert" id="flag-search-error">{error}</p> : null}
    <label htmlFor="flag-status">Estado no catálogo</label><select id="flag-status" value={status} onChange={event => setStatus(event.target.value)}><option value="all">Todas</option><option value="active">Ativas no catálogo</option><option value="archived">Arquivadas</option></select>
    <button type="submit">Aplicar filtros</button><button type="button" className="secondary" onClick={() => { setQ(''); setStatus('all'); setError(''); apply({ q: '', status: 'all' }); }}>Limpar filtros</button>
  </form>;
}
