import { useMemo } from 'react';
import { usePlans, useCurrentUsage, useUsageEvents } from '@/hooks/queries';
import { useWorkspace } from '@/store/workspace';
import { formatCents, formatBytes, formatNumber, formatDate, fromNow } from '@/lib/format';
import { PageLoader, SectionTitle, EmptyState } from '@/components/Ui';
import type { Plan } from '@/api/types';

const PLAN_METERS: Array<{ key: string; label: string; fmt: (n: number) => string; name: keyof Omit<Plan, 'code' | 'name' | 'price_monthly_cents'> }> = [
  { key: 'storage_bytes', label: 'Storage', fmt: formatBytes, name: 'storage_bytes' },
  { key: 'token_to_parse', label: 'Parse tokens', fmt: formatNumber, name: 'token_to_parse' },
  { key: 'seats', label: 'Seats', fmt: (n) => String(n), name: 'seats' },
  { key: 'llm_calls', label: 'LLM calls', fmt: formatNumber, name: 'llm_calls' },
];

export default function BillingPage() {
  const org = useWorkspace((s) => s.org)!;
  const plansQuery = usePlans();
  const usageQuery = useCurrentUsage(org.id);
  const eventsQuery = useUsageEvents(org.id);

  const currentPlan = plansQuery.data?.find((p) => p.code === org.plan_code);

  const meterTone = useMemo(() => {
    const map = new Map<string, number>();
    for (const m of usageQuery.data ?? []) {
      map.set(m.meter, m.limit < 0 ? 0 : Math.round((m.used / m.limit) * 100));
    }
    return map;
  }, [usageQuery.data]);

  return (
    <div>
      <SectionTitle title="Billing & Usage" subtitle={org.name} />

      {/* Current plan */}
      {currentPlan && (
        <div className="setting-section" style={{ maxWidth: 'none' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <h3 style={{ margin: 0 }}>{currentPlan.name} plan</h3>
              {currentPlan.price_monthly_cents > 0 && (
                <p className="muted" style={{ margin: '4px 0 0' }}>
                  {formatCents(currentPlan.price_monthly_cents)}/month
                </p>
              )}
            </div>
            <span className="badge success"><span className="dot" /> Active</span>
          </div>
        </div>
      )}

      {/* Usage meter summary */}
      {usageQuery.isLoading ? (
        <PageLoader />
      ) : (
        <div className="grid grid-4" style={{ marginBottom: 24 }}>
          {(usageQuery.data ?? []).map((m) => {
            const unlimited = m.limit < 0;
            const pct = meterTone.get(m.meter) ?? 0;
            const tone = pct >= 100 ? 'full' : pct >= 80 ? 'warn' : '';
            const label = PLAN_METERS.find((x) => x.key === m.meter);
            return (
              <div key={m.meter} className="meter-card">
                <div className="meter-label">{label?.label ?? m.meter}</div>
                <div className="meter-value">
                  {(label ? label.fmt(m.used) : m.used.toLocaleString())}
                  <span className="muted" style={{ fontSize: 14, fontWeight: 400 }}>
                    {' / '}{unlimited ? '∞' : label ? label.fmt(m.limit) : m.limit}
                  </span>
                </div>
                {!unlimited && (
                  <div className={`bar ${tone}`}><div style={{ width: `${pct}%` }} /></div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Plan comparison */}
      <SectionTitle title="Plans" subtitle="Choose the tier that fits your team." />
      {plansQuery.isLoading ? (
        <PageLoader />
      ) : (
        <div className="plan-grid">
          {(plansQuery.data ?? []).map((p) => {
            const isCurrent = p.code === org.plan_code;
            return (
              <div key={p.code} className={`plan-card${isCurrent ? ' current' : ''}`}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <strong style={{ fontSize: 16 }}>{p.name}</strong>
                  {isCurrent && <span className="badge">Current</span>}
                </div>
                <div className="price">
                  {p.price_monthly_cents === 0 ? 'Free' : formatCents(p.price_monthly_cents)}
                  {p.price_monthly_cents > 0 && <span className="muted" style={{ fontSize: 14 }}>/mo</span>}
                </div>
                <ul>
                  {PLAN_METERS.map((map) => (
                    <li key={map.key}>
                      {map.label}: <strong>{map.fmt(p[map.name])}</strong>
                    </li>
                  ))}
                </ul>
                <button className={`btn plan-cta ${isCurrent ? 'btn-secondary' : ''}`} disabled={isCurrent}>
                  {isCurrent ? 'Your current plan' : p.code === 'enterprise' ? 'Contact sales' : 'Upgrade'}
                </button>
              </div>
            );
          })}
        </div>
      )}

      <hr className="divider" />

      {/* Recent usage */}
      <SectionTitle title="Recent usage" />
      {eventsQuery.isLoading ? (
        <PageLoader />
      ) : (eventsQuery.data ?? []).length === 0 ? (
        <EmptyState text="No usage events yet." />
      ) : (
        <table className="data">
          <thead>
            <tr>
              <th>When</th>
              <th>Meter</th>
              <th>Amount</th>
              <th>Billable</th>
            </tr>
          </thead>
          <tbody>
            {(eventsQuery.data ?? []).slice(0, 15).map((ev) => (
              <tr key={ev.id} className="usage-event-row">
                <td>{formatDate(ev.occurred_at)} · {fromNow(ev.occurred_at)}</td>
                <td>{ev.meter}</td>
                <td>{ev.amount.toLocaleString()}</td>
                <td>{ev.billable ? 'Yes' : 'No'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}