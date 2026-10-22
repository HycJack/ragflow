import { useAdminOrgs } from '@/hooks/queries';
import { PageLoader, SectionTitle, EmptyState } from '@/components/Ui';
import { formatDate } from '@/lib/format';
import { Building2 } from 'lucide-react';

export default function AdminOrgsPage() {
  const orgsQuery = useAdminOrgs();
  const orgs = orgsQuery.data ?? [];

  return (
    <div>
      <SectionTitle
        title="Organizations"
        subtitle={`All SaaS workspaces (${orgs.length}) · read-only`}
      />

      {orgsQuery.isLoading ? (
        <PageLoader />
      ) : orgs.length === 0 ? (
        <EmptyState text="No organizations yet." />
      ) : (
        <table className="data">
          <thead>
            <tr>
              <th>Workspace</th>
              <th>Plan</th>
              <th>Members</th>
              <th>Owner</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {orgs.map((o) => (
              <tr key={o.id}>
                <td>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <span className="badge"><Building2 size={14} /></span>
                    <div>
                      <div>{o.name}</div>
                      <div className="muted" style={{ fontSize: 12 }}>/{o.slug}</div>
                    </div>
                  </div>
                </td>
                <td><span className="badge">{o.plan_code}</span></td>
                <td>{o.member_count}/{o.seat_limit < 0 ? '∞' : o.seat_limit}</td>
                <td className="muted">{o.owner_id.slice(0, 8)}…</td>
                <td className="muted">{formatDate(o.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}