import { useState } from 'react';
import { useAdminUsers, useAdminSetUserStatus } from '@/hooks/queries';
import { PageLoader, SectionTitle, EmptyState, Avatar, ErrorText } from '@/components/Ui';
import { formatDate } from '@/lib/format';
import { Ban, CheckCircle2 } from 'lucide-react';

export default function AdminUsersPage() {
  const usersQuery = useAdminUsers();
  const setStatus = useAdminSetUserStatus();
  const [error, setError] = useState('');

  const users = usersQuery.data ?? [];

  function toggle(userId: string, current: string) {
    setError('');
    const next = current === '1' ? '0' : '1';
    setStatus.mutate(
      { userId, status: next },
      { onError: (err) => setError((err as Error).message) },
    );
  }

  return (
    <div>
      <SectionTitle title="Users" subtitle={`All platform accounts (${users.length})`} />

      {error && <ErrorText>{error}</ErrorText>}

      {usersQuery.isLoading ? (
        <PageLoader />
      ) : users.length === 0 ? (
        <EmptyState text="No users yet." />
      ) : (
        <table className="data">
          <thead>
            <tr>
              <th>User</th>
              <th>Status</th>
              <th>Orgs</th>
              <th>Role</th>
              <th>Joined</th>
              <th style={{ textAlign: 'right' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id}>
                <td>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <Avatar name={u.nickname || u.email} />
                    <div>
                      <div>{u.nickname || u.email}</div>
                      <div className="muted" style={{ fontSize: 12 }}>{u.email}</div>
                    </div>
                  </div>
                </td>
                <td>
                  <span className={`badge ${u.status === '1' ? 'success' : ''}`}>
                    <span className="dot" /> {u.status === '1' ? 'Active' : 'Disabled'}
                  </span>
                </td>
                <td>{u.org_count}</td>
                <td>{u.is_platform_admin ? <span className="badge">Platform admin</span> : 'User'}</td>
                <td className="muted">{formatDate(u.created_at)}</td>
                <td style={{ textAlign: 'right' }}>
                  {u.is_platform_admin ? (
                    <span className="muted" style={{ fontSize: 12 }}>protected</span>
                  ) : (
                    <button
                      className={`btn btn-sm ${u.status === '1' ? 'btn-danger' : 'btn-secondary'}`}
                      disabled={setStatus.isPending}
                      onClick={() => toggle(u.id, u.status)}
                    >
                      {u.status === '1' ? <Ban size={14} /> : <CheckCircle2 size={14} />}
                      {u.status === '1' ? 'Disable' : 'Enable'}
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}