import { useEffect, useState } from 'react';
import { getHealth } from '../api/health';

export function App() {
  const [status, setStatus] = useState('Verificando conexão…');
  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 5000);
    getHealth(controller.signal)
      .then(() => setStatus('Serviço disponível'))
      .catch(() => { if (active) setStatus('Serviço indisponível'); })
      .finally(() => clearTimeout(timeout));
    return () => { active = false; controller.abort(); clearTimeout(timeout); };
  }, []);
  return (
    <main>
      <span className="brand" aria-hidden="true">FP</span>
      <h1>FlagPilot</h1>
      <p>Controle suas entregas, uma flag por vez.</p>
      <p role="status" className="status">{status}</p>
    </main>
  );
}
