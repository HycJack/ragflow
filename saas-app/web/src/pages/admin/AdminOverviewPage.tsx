import { useAdminOrgs, useAdminUsers } from '@/hooks/queries';
import { useWorkspace } from '@/store/workspace';
import { PageLoader, SectionTitle } from '@/components/Ui';
import { Building2, Users, ShieldCheck } from 'lucide-react';

export default function AdminOverviewPage() {
  const profile = useWorkspace((s) => s.profile);
  const orgsQuery = useAdminOrgs();
  const usersQuery = useAdminUsers();

  const orgs = orgsQuery.data ?? [];
  const users = usersQuery.data ?? [];
  const activeUsers = users.filter((u) => u.status === '1').length;
  const admins = users.filter((u) => u.is_platform_admin).length;
  const planCount: Record<string, number> = {};
  for (const o of orgs) planCount[o.plan_code] = (planCount[o.plan_code] ?? 0) + 1;

  const cards = [
    { label: 'Organizations', value: orgs.length, icon: <Building2 size={20} /> },
    { label: 'Users', value: users.length, icon: <Users size={20} /> },
    { label: 'Active users', value: activeUsers, icon: <Users size={20} /> },
    { label: 'Platform admins', value: admins, icon: <ShieldCheck size={20} /> },
  ];

  return (
    <div>
      <SectionTitle
        title="Overview"
        subtitle={`Platform dashboard · signed in as ${profile?.email}`}
      />

      {orgsQuery.isLoading || usersQuery.isLoading ? (
        <PageLoader />
      ) : (
        <>
          <div className="grid grid-4">
            {cards.map((c) => (
              <div key={c.label} className="meter-card">
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: 'var(--text-secondary)' }}>
                  <span className="meter-label" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {c.icon} {c.label}
                  </span>
                </div>
                <div className="meter-value">{c.value}</div>
              </div>
            ))}
          </div>

          <hr className="divider" />

          <SectionTitle title="Plan distribution" subtitle="How workspaces are split by plan" />
          <div className="grid grid-4">
            {Object.keys(planCount).length === 0 ? (
              <div className="muted">No workspaces yet.</div>
            ) : (
              Object.entries(planCount).map(([plan, count]) => (
                <div key={plan} className="meter-card">
                  <div className="meter-label">{plan}</div>
                  <div className="meter-value">{count}</div>
                </div>
              ))
            )}
          </div>
        </>
      )}
    </div>
  );
}