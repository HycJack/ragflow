import { Navigate, Route, Routes } from 'react-router-dom';
import { getToken } from '@/api/client';
import { useCurrentOrg, useProfile } from '@/store/workspace';
import LoginPage from '@/pages/LoginPage';
import RegisterPage from '@/pages/RegisterPage';
import OnboardingPage from '@/pages/OnboardingPage';
import AdminLayout from '@/pages/admin/AdminLayout';
import AdminOverviewPage from '@/pages/admin/AdminOverviewPage';
import AdminOrgsPage from '@/pages/admin/AdminOrgsPage';
import AdminUsersPage from '@/pages/admin/AdminUsersPage';
import WorkspaceLayout from '@/pages/workspace/WorkspaceLayout';
import OverviewPage from '@/pages/workspace/OverviewPage';
import MembersPage from '@/pages/workspace/MembersPage';
import SettingsPage from '@/pages/workspace/SettingsPage';
import BillingPage from '@/pages/workspace/BillingPage';

/** Require an authenticated session; otherwise bounce to /login. */
function RequireAuth({ children }: { children: JSX.Element }) {
  if (!getToken()) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

/** Require platform-admin privileges for the admin console. */
function RequireAdmin({ children }: { children: JSX.Element }) {
  const profile = useProfile();
  if (!profile?.is_platform_admin) {
    return <Navigate to="/onboarding" replace />;
  }
  return children;
}

/** Require a selected organization; otherwise bounce to /onboarding. */
function RequireOrg({ children }: { children: JSX.Element }) {
  const org = useCurrentOrg();
  if (!org) {
    return <Navigate to="/onboarding" replace />;
  }
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route
        path="/onboarding"
        element={
          <RequireAuth>
            <OnboardingPage />
          </RequireAuth>
        }
      />
      <Route
        path="/admin"
        element={
          <RequireAuth>
            <RequireAdmin>
              <AdminLayout />
            </RequireAdmin>
          </RequireAuth>
        }
      >
        <Route index element={<AdminOverviewPage />} />
        <Route path="orgs" element={<AdminOrgsPage />} />
        <Route path="users" element={<AdminUsersPage />} />
      </Route>
      <Route
        path="/workspace"
        element={
          <RequireAuth>
            <WorkspaceLayout />
          </RequireAuth>
        }
      >
        <Route index element={<Navigate to="overview" replace />} />
        <Route
          path="overview"
          element={
            <RequireOrg>
              <OverviewPage />
            </RequireOrg>
          }
        />
        <Route
          path="members"
          element={
            <RequireOrg>
              <MembersPage />
            </RequireOrg>
          }
        />
        <Route
          path="billing"
          element={
            <RequireOrg>
              <BillingPage />
            </RequireOrg>
          }
        />
        <Route
          path="settings"
          element={
            <RequireOrg>
              <SettingsPage />
            </RequireOrg>
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  );
}