import { Link } from 'react-router';
import { useEffect, useRef, useState } from 'react';
import { ApiError, logout, type Identity } from '../../api/auth';
import { Organizations } from '../organizations/Organizations';

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
      <p><Link to="/account/security">Segurança da conta</Link></p>
      <Organizations identity={identity} onExpired={onLogout} />
      {error ? <p role="alert" className="error">{error}</p> : null}
    </section>
  );
}
