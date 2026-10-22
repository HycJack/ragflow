import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useOrganizations, useLogout } from '@/hooks/queries';
import { useWorkspace } from '@/store/workspace';
import { setToken } from '@/api/client';
import { Spinner } from '@/components/Ui';
import { LayoutDashboard, Users, CreditCard, Settings, LogOut } from 'lucide-react';

export default function WorkspaceLayout() {
  const navigate = useNavigate();
  const orgsQuery = useOrganizations();
  const logout = useLogout();
  const { org, setOrg, setOrganizations, profile } = useWorkspace();

  // Platform admins live in the operator console, not the user workspace.
  useEffect(() => {
    if (profile?.is_platform_admin) {
      navigate('/admin', { replace: true });
    }
  }, [profile?.is_platform_admin, navigate]);

  async function handleSwitch(id: string) {
    const next = orgsQuery.data?.find((o) => o.id === id);
    if (next) {
      setOrg(next);
      navigate('/workspace/overview', { replace: true });
    }
  }

  async function handleLogout() {
    await logout.mutateAsync();
    setToken(null);
    setOrganizations([]);
    setOrg(null);
    navigate('/login', { replace: true });
  }

  const navItems = [
    { to: '/workspace/overview', label: 'Overview', icon: <LayoutDashboard size={18} /> },
    { to: '/workspace/members', label: 'Members', icon: <Users size={18} /> },
    { to: '/workspace/billing', label: 'Billing & Usage', icon: <CreditCard size={18} /> },
    { to: '/workspace/settings', label: 'Settings', icon: <Settings size={18} /> },
  ];

  return (
    <div className="shell">
      <aside>
        <div className="brand">
          <img src="/logo.svg" width={28} height={28} alt="logo" />
          <span>RAGFlow SaaS</span>
        </div>

        <OrgSwitcher
          orgs={orgsQuery.data ?? []}
          currentOrgId={org?.id ?? ''}
          loading={orgsQuery.isLoading}
          onSwitch={handleSwitch}
        />

        <nav style={{ display: 'flex', flexDirection: 'column', gap: 4, marginTop: 8 }}>
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}
            >
              {item.icon}
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="shell-footer">
          <button className="btn btn-secondary" onClick={handleLogout} disabled={logout.isPending}>
            <LogOut size={16} />
            Log out
          </button>
        </div>
      </aside>

      <main>
        <Outlet />
      </main>
    </div>
  );
}

interface OrgSwitcherProps {
  orgs: Array<{ id: string; name: string; my_role: string }>;
  currentOrgId: string;
  loading: boolean;
  onSwitch: (id: string) => void | Promise<void>;
}

function OrgSwitcher({ orgs, currentOrgId, loading, onSwitch }: OrgSwitcherProps) {
  if (loading) {
    return (
      <div className="muted" style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0' }}>
        <Spinner /> Loading…
      </div>
    );
  }
  if (orgs.length === 0) {
    return <div className="muted">No workspace selected.</div>;
  }
  return (
    <div className="org-switcher">
      <label className="field-title">Workspace</label>
      <select value={currentOrgId} onChange={(e) => onSwitch(e.target.value)}>
        {orgs.map((o) => (
          <option key={o.id} value={o.id}>
            {o.name} ({o.my_role})
          </option>
        ))}
      </select>
    </div>
  );
}

export function CurrentOrgBadge() {
  const { org } = useWorkspace();
  if (!org) return null;
  return <span className="badge">{org.name}</span>;
}