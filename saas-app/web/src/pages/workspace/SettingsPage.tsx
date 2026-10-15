import { useWorkspace } from '@/store/workspace';
import { SectionTitle } from '@/components/Ui';
import { formatDate } from '@/lib/format';

export default function SettingsPage() {
  const org = useWorkspace((s) => s.org);
  const profile = useWorkspace((s) => s.profile);

  if (!org) return null;

  return (
    <div>
      <SectionTitle title="Settings" subtitle="Organization and account details." />

      <div className="setting-section">
        <h3 style={{ marginTop: 0 }}>Organization</h3>
        <dl style={{ display: 'grid', gridTemplateColumns: '140px 1fr', gap: 10, fontSize: 14 }}>
          <dt className="muted">Name</dt>
          <dd>{org.name}</dd>
          <dt className="muted">Slug</dt>
          <dd>{org.slug}</dd>
          <dt className="muted">Plan</dt>
          <dd>{org.plan_code}</dd>
          <dt className="muted">Owner</dt>
          <dd>{org.owner_id}</dd>
          <dt className="muted">Created</dt>
          <dd>{formatDate(org.created_at)}</dd>
        </dl>
      </div>

      <div className="setting-section">
        <h3 style={{ marginTop: 0 }}>Account</h3>
        <dl style={{ display: 'grid', gridTemplateColumns: '140px 1fr', gap: 10, fontSize: 14 }}>
          <dt className="muted">Name</dt>
          <dd>{profile?.nickname ?? '—'}</dd>
          <dt className="muted">Email</dt>
          <dd>{profile?.email ?? '—'}</dd>
          <dt className="muted">Role in org</dt>
          <dd>{org.my_role}</dd>
        </dl>
      </div>

      <div className="setting-section">
        <h3 style={{ marginTop: 0 }}>Danger zone</h3>
        <p className="muted" style={{ fontSize: 14 }}>
          Account management (transfer ownership, delete workspace) is available in the full
          administration console.
        </p>
      </div>
    </div>
  );
}