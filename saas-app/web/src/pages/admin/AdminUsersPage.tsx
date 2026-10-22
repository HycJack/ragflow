import { useAdminUsers } from '@/hooks/queries';
import { PageLoader, SectionTitle, EmptyState, Avatar } from '@/components/Ui';
import { formatDate } from '@/lib/format';

export default function AdminUsersPage() {
  const usersQuery = useAdminUsers();
  const users = usersQuery.data ?? [];

  return (
    <div>
      <SectionTitle
        title="Users"
        subtitle={`All platform accounts (${users.length}) · read-only`}
      />

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
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}