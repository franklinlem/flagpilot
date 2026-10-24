import { useEffect, useRef, useState } from 'react';
import { ApiError, logout, type Identity } from '../../api/auth';
const roles = { Owner: 'Proprietário', Admin: 'Administrador', Editor: 'Editor', Viewer: 'Visualizador' };

export function Account({ identity, onLogout }: { identity: Identity; onLogout: () => void }) {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { heading.current?.focus(); }, []);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  async function signOut() {
    setPending(true); setError('');
    try { await logout(identity.csrfToken); onLogout(); }
    catch (failure) {
      if (failure instanceof ApiError && failure.status === 401) { onLogout(); return; }
      setError(failure instanceof Error ? failure.message : 'Não foi possível sair.');
    } finally { setPending(false); }
  }
  return (
    <section className="account" aria-labelledby="account-title">
      <div className="account-heading"><div><p className="eyebrow">Sua conta</p><h2 id="account-title" ref={heading} tabIndex={-1}>Bem-vindo ao FlagPilot</h2></div><button className="secondary" disabled={pending} onClick={signOut}>{pending ? 'Saindo…' : 'Sair'}</button></div>
      <p className="muted">Conectado como <strong>{identity.user.email}</strong></p>
      <h3>Suas organizações</h3>
      {identity.memberships.length === 0 ? <p>Você ainda não faz parte de uma organização.</p> : (
        <ul className="organizations">{identity.memberships.map(member => (
          <li className="card" key={member.organizationId}><strong>{member.organizationName}</strong><span className="badge">{roles[member.role]}</span></li>
        ))}</ul>
      )}
      {error ? <p role="alert" className="error">{error}</p> : null}
    </section>
  );
}
