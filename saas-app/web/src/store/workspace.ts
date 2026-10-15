import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Organization, Profile } from '@/api/types';
import { getToken } from '@/api/client';

interface WorkspaceState {
  // Currently selected organization context.
  org: Organization | null;
  // All organizations the user belongs to.
  organizations: Organization[];
  // Current user profile (set on login/register).
  profile: Profile | null;
  setOrg: (org: Organization | null) => void;
  setOrganizations: (orgs: Organization[]) => void;
  setProfile: (profile: Profile | null) => void;
  // Auth flag mirrors the presence of a token.
  isLoggedIn: () => boolean;
}

// Keep the token read out of the serialized store (persist only org selection).
export const useWorkspace = create<WorkspaceState>()(
  persist(
    (set) => ({
      org: null,
      organizations: [],
      profile: null,
      setOrg: (org) => set({ org }),
      setOrganizations: (organizations) => set({ organizations }),
      setProfile: (profile) => set({ profile }),
      isLoggedIn: () => Boolean(getToken()),
    }),
    {
      name: 'saas-workspace',
      partialize: (state) => ({
        org: state.org,
        organizations: state.organizations,
        profile: state.profile,
      }),
    },
  ),
);

// Convenience helpers selecting pieces.
export const useCurrentOrg = () => useWorkspace((s) => s.org);
export const useOrgList = () => useWorkspace((s) => s.organizations);
export const useProfile = () => useWorkspace((s) => s.profile);