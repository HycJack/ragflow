import { useState, type FormEvent } from 'react';
import { useMembers, useInvitations, useCreateInvitation, useUpdateMemberRole, useRemoveMember, useCurrentUsage } from '@/hooks/queries';
import { useWorkspace } from '@/store/workspace';
import { formatDate } from '@/lib/format';
import { Avatar, PageLoader, SectionTitle, EmptyState, ErrorText } from '@/components/Ui';
import type { Member } from '@/api/types';
import { UserPlus, Trash2, Copy, Check } from 'lucide-react';

const ROLE_OPTIONS = ['owner', 'admin', 'member', 'viewer'];

export default function MembersPage() {
  const org = useWorkspace((s) => s.org)!;
  const currentUserId = useWorkspace((s) => s.profile?.id ?? '');
  const canManage = org.my_role === 'owner' || org.my_role === 'admin';

  const membersQuery = useMembers(org.id);
  const invitationsQuery = useInvitations(org.id);
  const usageQuery = useCurrentUsage(org.id);

  const updateRole = useUpdateMemberRole(org.id);
  const removeMember = useRemoveMember(org.id);

  const [inviteOpen, setInviteOpen] = useState(false);
  const [copied, setCopied] = useState('');

  const seatSnapshot = (usageQuery.data ?? []).find((m) => m.meter === 'seats');
  const seatsUsed = seatSnapshot?.used ?? org.seat_used ?? 0;
  const seatsLimit = seatSnapshot?.limit ?? org.seat_limit;

  function copy(code: string) {
    const url = `${window.location.origin}/register?invite=${code}`;
    void navigator.clipboard.writeText(url);
    setCopied(code);
    setTimeout(() => setCopied(''), 1500);
  }

  return (
    <div>
      <SectionTitle
        title="Members"
        subtitle={`${seatsUsed}/${seatsLimit < 0 ? '∞' : seatsLimit} seats`}
        actions={
          canManage && (seatsLimit < 0 || seatsUsed < seatsLimit) ? (
            <button className="btn" onClick={() => setInviteOpen(true)}>
              <UserPlus size={16} /> Invite member
            </button>
          ) : undefined
        }
      />

      {membersQuery.isLoading ? (
        <PageLoader />
      ) : (membersQuery.data ?? []).length === 0 ? (
        <EmptyState text="No members yet." />
      ) : (
        <table className="data">
          <thead>
            <tr>
              <th>Member</th>
              <th>Role</th>
              <th>Joined</th>
              <th>Status</th>
              {canManage && <th style={{ textAlign: 'right' }}>Actions</th>}
            </tr>
          </thead>
          <tbody>
            {(membersQuery.data ?? []).map((m) => (
              <MemberRow
                key={m.user_id}
                member={m}
                currentUserId={currentUserId}
                canManage={canManage}
                onChangeRole={(role) => updateRole.mutate({ userId: m.user_id, role })}
                onRemove={() => removeMember.mutate(m.user_id)}
              />
            ))}
          </tbody>
        </table>
      )}

      <hr className="divider" />

      {canManage && (
        <>
          <SectionTitle title="Invitations" />
          {invitationsQuery.isLoading ? (
            <PageLoader />
          ) : (invitationsQuery.data ?? []).length === 0 ? (
            <EmptyState text="No active invitations." />
          ) : (
            <table className="data">
              <thead>
                <tr>
                  <th>Code</th>
                  <th>Role</th>
                  <th>Email</th>
                  <th>Uses</th>
                  <th>Expires</th>
                  <th style={{ textAlign: 'right' }}>Invite link</th>
                </tr>
              </thead>
              <tbody>
                {(invitationsQuery.data ?? []).map((inv) => (
                  <tr key={inv.id}>
                    <td><code>{inv.code.slice(0, 12)}…</code></td>
                    <td>{inv.role}</td>
                    <td className="muted">{inv.email || 'anyone'}</td>
                    <td>{inv.used_count}/{inv.max_uses}</td>
                    <td>{inv.expires_at ? formatDate(inv.expires_at) : '—'}</td>
                    <td style={{ textAlign: 'right' }}>
                      <button className="btn btn-secondary btn-sm" onClick={() => copy(inv.code)}>
                        {copied === inv.code ? <Check size={14} /> : <Copy size={14} />}
                        {copied === inv.code ? 'Copied' : 'Copy'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}

      {inviteOpen && <InviteDialog orgId={org.id} onClose={() => setInviteOpen(false)} onCopy={copy} />}
    </div>
  );
}

function MemberRow({
  member,
  currentUserId,
  canManage,
  onChangeRole,
  onRemove,
}: {
  member: Member;
  currentUserId: string;
  canManage: boolean;
  onChangeRole: (role: string) => void;
  onRemove: () => void;
}) {
  return (
    <tr>
      <td>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <Avatar name={member.nickname || member.email} />
          <div>
            <div>{member.nickname || member.email}</div>
            <div className="muted" style={{ fontSize: 12 }}>{member.email}</div>
          </div>
        </div>
      </td>
      <td>
        {canManage ? (
          <select
            value={member.role}
            onChange={(e) => onChangeRole(e.target.value)}
            disabled={member.user_id === currentUserId}
            style={{ width: 'auto' }}
          >
            {ROLE_OPTIONS.map((r) => (
              <option key={r} value={r}>{r}</option>
            ))}
          </select>
        ) : (
          <span className="badge">{member.role}</span>
        )}
      </td>
      <td className="muted">{formatDate(member.joined_at)}</td>
      <td>
        <span className={`badge ${member.status === '1' ? 'success' : ''}`}>
          <span className="dot" /> {member.status === '1' ? 'Active' : 'Removed'}
        </span>
      </td>
      {canManage && (
        <td style={{ textAlign: 'right' }}>
          {member.user_id !== currentUserId && (
            <button className="btn btn-danger btn-sm" onClick={onRemove}>
              <Trash2 size={14} />
            </button>
          )}
        </td>
      )}
    </tr>
  );
}

function InviteDialog({ orgId, onClose, onCopy }: { orgId: string; onClose: () => void; onCopy: (code: string) => void }) {
  const createInvitation = useCreateInvitation(orgId);
  const [role, setRole] = useState('member');
  const [email, setEmail] = useState('');
  const [result, setResult] = useState<{ code: string } | null>(null);
  const [error, setError] = useState('');

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    createInvitation.mutate(
      {
        role,
        email: email || undefined,
        max_uses: email ? 1 : 10,
      },
      {
        onSuccess: (inv) => setResult({ code: inv.code }),
        onError: (err) => setError((err as Error).message),
      },
    );
  }

  return (
    <div className="modal-backdrop" onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="card" style={{ width: 460 }}>
        <div className="card-header">
          <h1>Invite member</h1>
          <p>Share a link or restrict to a single email.</p>
        </div>

        {result ? (
          <>
            <p className="muted">Share this invite link with your teammate:</p>
            <div className="invite-box">
              <code>{`${window.location.origin}/register?invite=${result.code}`}</code>
              <button className="btn btn-secondary btn-sm" onClick={() => onCopy(result.code)}>
                <Copy size={14} /> Copy
              </button>
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16, gap: 8 }}>
              <button className="btn btn-secondary" onClick={onClose}>Done</button>
            </div>
          </>
        ) : (
          <form onSubmit={handleSubmit}>
            <div className="field">
              <label htmlFor="invRole">Role</label>
              <select id="invRole" value={role} onChange={(e) => setRole(e.target.value)}>
                {['member', 'viewer', 'admin'].map((r) => (
                  <option key={r} value={r}>{r}</option>
                ))}
              </select>
            </div>
            <div className="field">
              <label htmlFor="invEmail">Email (optional)</label>
              <input id="invEmail" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="restrict to one address" />
            </div>

            {error && <ErrorText>{error}</ErrorText>}

            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 8, gap: 8 }}>
              <button className="btn btn-secondary" type="button" onClick={onClose}>Cancel</button>
              <button className="btn" type="submit" disabled={createInvitation.isPending}>
                <UserPlus size={16} /> Generate invite
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}