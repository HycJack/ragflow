import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useLogout } from '@/hooks/queries';
import { useWorkspace } from '@/store/workspace';
import { setToken } from '@/api/client';
import { LayoutDashboard, Building2, Users, ShieldCheck, LogOut } from 'lucide-react';

export default function AdminLayout() {
  const navigate = useNavigate();
  const logout = useLogout();
  const profile = useWorkspace((s) => s.profile);

  async function handleLogout() {
    await logout.mutateAsync();
    setToken(null);
    navigate('/login', { replace: true });
  }

  const navItems = [
    { to: '/admin', label: 'Overview', icon: <LayoutDashboard size={18} />, end: true },
    { to: '/admin/orgs', label: 'Organizations', icon: <Building2 size={18} /> },
    { to: '/admin/users', label: 'Users', icon: <Users size={18} /> },
  ];

  return (
    <div className="shell">
      <aside>
        <div className="brand">
          <img src="/logo.svg" width={28} height={28} alt="logo" />
          <span>Admin Console</span>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 4px 12px' }}>
          <ShieldCheck size={16} className="muted" />
          <span className="muted" style={{ fontSize: 13 }}>
            Signed in as {profile?.email}
          </span>
        </div>

        <nav style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
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