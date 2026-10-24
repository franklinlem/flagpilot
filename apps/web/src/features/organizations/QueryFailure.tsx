import { useEffect } from 'react';
import { ApiError } from '../../api/auth';
export function QueryFailure({ error, retry, onExpired, buttonLabel = 'Tentar novamente' }: { error: Error; retry: () => void; onExpired: () => void; buttonLabel?: string }) {
  useEffect(() => { if (error instanceof ApiError && error.status === 401) onExpired(); }, [error, onExpired]);
  return <div><p role="alert" className="error">{error.message}</p><button className="secondary" onClick={retry}>{buttonLabel}</button></div>;
}
