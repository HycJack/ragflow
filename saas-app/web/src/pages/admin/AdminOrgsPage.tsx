import { useState } from 'react';
import { useAdminOrgs, useAdminSetOrgPlan, useAdminDeleteOrg, usePlans } from '@/hooks/queries';
import { PageLoader, SectionTitle, EmptyState, ErrorText } from '@/components/Ui';
import { formatDate } from '@/lib/format';
import { Building2, Trash2 } from 'lucide-react';

export default function AdminOrgsPage() {
  const orgsQuery = useAdminOrgs();
  const plansQuery = usePlans();
  const setPlan = useAdminSetOrgPlan();
  const deleteOrg = useAdminDeleteOrg();
  const [error, setError] = useState('');

  const orgs = orgsQuery.data ?? [];
  const planCodes = (plansQuery.data ?? []).map((p) => p.code);

  function changePlan(orgId: string, planCode: string) {
    setError('');
    setPlan.mutate({ orgId, planCode }, { onError: (err) => setError((err as Error).message) });
  }

  function remove(orgId: string, name: string) {
    if (!window.confirm(`Delete "${name}"? This removes its members, invitations and usage data.`)) {
      return;
    }
    setError('');
    deleteOrg.mutate(orgId, { onError: (err) => setError((err as Error).message) });
  }

  return (
    <div>
      <SectionTitle title="Organizations" subtitle={`All SaaS workspaces (${orgs.length})`} />

      {error && <ErrorText>{error}</ErrorText>}

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
              <th style={{ textAlign: 'right' }}>Actions</th>
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
                <td>
                  <select
                    value={o.plan_code}
                    disabled={setPlan.isPending}
                    onChange={(e) => changePlan(o.id, e.target.value)}
                    style={{ width: 'auto' }}
                    aria-label={`Plan for ${o.name}`}
                  >
                    {planCodes.length === 0 && <option value={o.plan_code}>{o.plan_code}</option>}
                    {planCodes.map((code) => (
                      <option key={code} value={code}>{code}</option>
                    ))}
                  </select>
                </td>
                <td>{o.member_count}/{o.seat_limit < 0 ? '∞' : o.seat_limit}</td>
                <td className="muted">{o.owner_id.slice(0, 8)}…</td>
                <td className="muted">{formatDate(o.created_at)}</td>
                <td style={{ textAlign: 'right' }}>
                  <button
                    className="btn btn-danger btn-sm"
                    disabled={deleteOrg.isPending}
                    onClick={() => remove(o.id, o.name)}
                  >
                    <Trash2 size={14} /> Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}