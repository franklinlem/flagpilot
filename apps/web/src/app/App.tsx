import { QueryClient, QueryClientProvider, useQueryClient } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router';
import { Members } from '../features/organizations/Members';
import { useEffect, useState } from 'react';
import { ApiError, getIdentity, type Identity } from '../api/auth';
import { LoginForm } from '../features/auth/LoginForm';
import { Account } from '../features/auth/Account';

type State = { status: 'checking' } | { status: 'anonymous' } | { status: 'authenticated'; identity: Identity } | { status: 'unavailable' };
function SessionApp() {
  const client = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const [state, setState] = useState<State>({ status: 'checking' });
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 8000);
    getIdentity(controller.signal).then(identity => {
      if (active) setState({ status: 'authenticated', identity });
    }).catch(error => {
      if (active) setState({ status: error instanceof ApiError && error.status === 401 ? 'anonymous' : 'unavailable' });
    }).finally(() => clearTimeout(timeout));
    return () => { active = false; controller.abort(); clearTimeout(timeout); };
  }, [attempt]);
  function signedOut() { client.clear(); setState({ status: 'anonymous' }); void navigate('/'); }
  function signedIn(identity: Identity) { client.clear(); setState({ status: 'authenticated', identity }); }
  function retry() { setState({ status: 'checking' }); setAttempt(value => value + 1); }
  return (
    <main>
      <header className="brand-heading"><span className="brand" aria-hidden="true">FP</span><div><h1>FlagPilot</h1><p className="muted">Controle suas entregas, uma flag por vez.</p></div></header>
      {state.status === 'checking' ? <p role="status">Verificando sessão…</p> : null}
      {state.status === 'unavailable' ? <section className="card"><h2>Serviço indisponível</h2><p role="alert">Não foi possível verificar sua sessão.</p><button onClick={retry}>Tentar novamente</button></section> : null}
      {state.status === 'anonymous' ? <LoginForm onLogin={signedIn} /> : null}
      {state.status === 'authenticated' ? <Routes><Route path="/" element={<Account identity={state.identity} onLogout={signedOut} />} /><Route path="/organizations/:orgId/members" element={<Members key={location.pathname} identity={state.identity} onExpired={signedOut} />} /><Route path="*" element={<Navigate to="/" replace />} /></Routes> : null}
    </main>
  );
}

export function App() {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 0 } } }));
  return <QueryClientProvider client={client}><BrowserRouter><SessionApp /></BrowserRouter></QueryClientProvider>;
}
