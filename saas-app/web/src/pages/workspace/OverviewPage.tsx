import { useMemo } from 'react';
import { useCurrentUsage, useUsageEvents } from '@/hooks/queries';
import { useWorkspace } from '@/store/workspace';
import { formatBytes, formatNumber, fromNow } from '@/lib/format';
import { PageLoader, SectionTitle, EmptyState } from '@/components/Ui';
import { HardDrive, FileText, Bot, Users } from 'lucide-react';

const METER_LABELS: Record<string, { label: string; icon: React.ReactNode; fmt: (n: number) => string }> = {
  storage_bytes: { label: 'Storage', icon: <HardDrive size={18} />, fmt: formatBytes },
  token_to_parse: { label: 'Tokens to parse', icon: <FileText size={18} />, fmt: formatNumber },
  llm_calls: { label: 'LLM calls', icon: <Bot size={18} />, fmt: formatNumber },
  seats: { label: 'Seats', icon: <Users size={18} />, fmt: (n) => String(n) },
};

export default function OverviewPage() {
  const org = useWorkspace((s) => s.org)!;
  const usageQuery = useCurrentUsage(org.id);
  const eventsQuery = useUsageEvents(org.id);

  const meters = useMemo(() => {
    const data = usageQuery.data ?? [];
    const byMeter = new Map(data.map((m) => [m.meter, m]));
    return ['storage_bytes', 'token_to_parse', 'llm_calls', 'seats'].map((meter) => {
      const def = METER_LABELS[meter];
      const snapshot = byMeter.get(meter);
      return { meter, ...def, used: snapshot?.used ?? 0, limit: snapshot?.limit ?? -1 };
    });
  }, [usageQuery.data]);

  return (
    <div>
      <SectionTitle title={org.name} subtitle={`Plan: ${org.plan_code} · ${org.my_role}`} />

      {usageQuery.isLoading ? (
        <PageLoader />
      ) : (
        <div className="grid grid-4">
          {meters.map((m) => {
            const unlimited = m.limit < 0;
            const pct = unlimited ? 0 : Math.min(100, Math.round((m.used / m.limit) * 100));
            const tone = pct >= 100 ? 'full' : pct >= 80 ? 'warn' : '';
            return (
              <div key={m.meter} className="meter-card">
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: 'var(--text-secondary)' }}>
                  {m.icon}
                  <span className="meter-label">{m.label}</span>
                </div>
                <div className="meter-value">
                  {m.fmt(m.used)}
                  <span className="muted" style={{ fontSize: 14, fontWeight: 400 }}> / {unlimited ? '∞' : m.fmt(m.limit)}</span>
                </div>
                {!unlimited && (
                  <div className={`bar ${tone}`}>
                    <div style={{ width: `${pct}%` }} />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      <hr className="divider" />

      <SectionTitle title="Recent activity" />
      {eventsQuery.isLoading ? (
        <PageLoader />
      ) : (eventsQuery.data ?? []).length === 0 ? (
        <EmptyState text="No usage yet. Upload content or query a bot to see activity here." />
      ) : (
        <table className="data">
          <thead>
            <tr>
              <th>When</th>
              <th>Meter</th>
              <th>Amount</th>
              <th>Source</th>
            </tr>
          </thead>
          <tbody>
            {(eventsQuery.data ?? []).slice(0, 10).map((ev) => (
              <tr key={ev.id}>
                <td>{fromNow(ev.occurred_at)}</td>
                <td>{METER_LABELS[ev.meter]?.label ?? ev.meter}</td>
                <td>{ev.amount.toLocaleString()}</td>
                <td className="muted">{ev.ref_kind || ev.meter}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}