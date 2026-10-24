import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { changeRole, removeMember, roles, roleNames, type Member, type Role } from '../../api/organizations';
import { QueryFailure } from './QueryFailure';
export function MemberRow({ member, owner, csrf, refresh, onExpired }: { member: Member; owner: boolean; csrf: string; refresh: () => Promise<void>; onExpired: () => void }) {
  const [role, setRole] = useState<Role>(member.role);
  const [confirm, setConfirm] = useState(false);
  const mutation = useMutation({ mutationFn: async (remove: boolean): Promise<void> => { if (remove) await removeMember(member.organizationId, member.userId, csrf); else await changeRole(member.organizationId, member.userId, role, csrf); }, onSuccess: refresh });
  return <li className="card member"><div><strong>{member.email}</strong><p className="hint">ID: <code>{member.userId}</code></p>{member.disabled ? <p>Conta desativada</p> : null}</div>
    {owner ? <div className="member-actions"><label htmlFor={`role-${member.userId}`}>Papel de {member.email}</label><select id={`role-${member.userId}`} disabled={mutation.isPending} value={role} onChange={event => setRole(event.target.value as Role)}>{roles.map(value => <option value={value} key={value}>{roleNames[value]}</option>)}</select>
      <button className="secondary" disabled={mutation.isPending || role === member.role} onClick={() => mutation.mutate(false)}>Salvar papel</button>
      {confirm ? <div className="confirmation"><p>Remover o acesso de {member.email} a esta organização?</p><button disabled={mutation.isPending} onClick={() => mutation.mutate(true)}>Confirmar remoção</button><button className="secondary" disabled={mutation.isPending} onClick={() => setConfirm(false)}>Cancelar</button></div> : <button className="secondary danger" disabled={mutation.isPending} onClick={() => { mutation.reset(); setConfirm(true); }}>Remover membro</button>}
    </div> : <span className="badge">{roleNames[member.role]}</span>}
    {mutation.error ? <QueryFailure error={mutation.error} buttonLabel="Fechar aviso" retry={() => mutation.reset()} onExpired={onExpired} /> : null}
  </li>;
}
