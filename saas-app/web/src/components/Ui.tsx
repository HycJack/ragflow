import { type ReactNode } from 'react';
import { initialOf } from '@/lib/format';

export function Spinner() {
  return (
    <span className="spinner" aria-label="Loading" />
  );
}

export function PageLoader() {
  return (
    <div className="empty">
      <Spinner />
    </div>
  );
}

export function ErrorText({ children }: { children: ReactNode }) {
  return <div className="error-text">{children}</div>;
}

export function Avatar({ name, className }: { name: string; className?: string }) {
  return (
    <span className={`avatar ${className ?? ''}`} aria-hidden>
      {initialOf(name)}
    </span>
  );
}

export function EmptyState({ text }: { text: string }) {
  return (
    <div className="empty">
      <div className="muted">{text}</div>
    </div>
  );
}

export function SectionTitle({ title, subtitle, actions }: { title: string; subtitle?: string; actions?: ReactNode }) {
  return (
    <div className="section-title">
      <div>
        <h2>{title}</h2>
        {subtitle && <p className="muted">{subtitle}</p>}
      </div>
      {actions && <div className="section-actions">{actions}</div>}
    </div>
  );
}