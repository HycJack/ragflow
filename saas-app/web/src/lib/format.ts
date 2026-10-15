// Formatting helpers (dayjs-based per RAGFlow convention).
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

export function formatBytes(bytes: number): string {
  if (bytes < 0) return '∞';
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

export function formatNumber(n: number): string {
  if (n < 0) return '∞';
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return String(n);
}

export function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

export function fromNow(iso: string): string {
  return dayjs(iso).fromNow();
}

export function formatDate(iso?: string | null): string {
  if (!iso) return '—';
  return dayjs(iso).format('YYYY-MM-DD');
}

export function initialOf(name: string): string {
  return (name || '?').trim().charAt(0).toUpperCase();
}