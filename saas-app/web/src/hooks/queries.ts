// Query key factory + TanStack Query hooks. Following the RAGFlow convention,
// keys are referenced by a factory (never raw arrays inline).

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  adminService,
  authService,
  orgService,
  planService,
  usageService,
} from '@/services/api';

export const QueryKeys = {
  plans: () => ['plans'] as const,
  orgs: () => ['orgs'] as const,
  org: (orgId: string) => ['org', orgId] as const,
  adminUsers: () => ['admin', 'users'] as const,
  adminOrgs: () => ['admin', 'orgs'] as const,
  members: (orgId: string) => ['org', orgId, 'members'] as const,
  invitations: (orgId: string) => ['org', orgId, 'invitations'] as const,
  usage: (orgId: string) => ['org', orgId, 'usage'] as const,
  usageEvents: (orgId: string, meter?: string) =>
    ['org', orgId, 'usage-events', meter ?? ''] as const,
};

// ---- Queries ----

export function usePlans() {
  return useQuery({
    queryKey: QueryKeys.plans(),
    queryFn: () => planService.list(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useOrganizations() {
  return useQuery({
    queryKey: QueryKeys.orgs(),
    queryFn: () => orgService.list(),
  });
}

export function useMembers(orgId: string) {
  return useQuery({
    queryKey: QueryKeys.members(orgId),
    queryFn: () => orgService.members(orgId),
    enabled: Boolean(orgId),
  });
}

export function useInvitations(orgId: string) {
  return useQuery({
    queryKey: QueryKeys.invitations(orgId),
    queryFn: () => orgService.listInvitations(orgId),
    enabled: Boolean(orgId),
  });
}

export function useCurrentUsage(orgId: string) {
  return useQuery({
    queryKey: QueryKeys.usage(orgId),
    queryFn: () => usageService.current(orgId),
    enabled: Boolean(orgId),
    refetchInterval: 60 * 1000,
  });
}

export function useUsageEvents(orgId: string, meter?: string) {
  return useQuery({
    queryKey: QueryKeys.usageEvents(orgId, meter),
    queryFn: () => usageService.events(orgId, meter),
    enabled: Boolean(orgId),
  });
}

// ---- Platform admin queries ----

export function useAdminUsers() {
  return useQuery({
    queryKey: QueryKeys.adminUsers(),
    queryFn: () => adminService.users(),
  });
}

export function useAdminOrgs() {
  return useQuery({
    queryKey: QueryKeys.adminOrgs(),
    queryFn: () => adminService.orgs(),
  });
}

// ---- Mutations ----

// ---- Platform admin mutations ----

export function useAdminSetUserStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { userId: string; status: string }) =>
      adminService.setUserStatus(params.userId, params.status),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: QueryKeys.adminUsers() }),
  });
}

export function useAdminSetOrgPlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { orgId: string; planCode: string }) =>
      adminService.setOrgPlan(params.orgId, params.planCode),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: QueryKeys.adminOrgs() }),
  });
}

export function useAdminDeleteOrg() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (orgId: string) => adminService.deleteOrg(orgId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.adminOrgs() });
      queryClient.invalidateQueries({ queryKey: QueryKeys.adminUsers() });
    },
  });
}

export function useCreateOrg() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { name: string; slug?: string }) => orgService.create(params),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: QueryKeys.orgs() }),
  });
}

export function useAcceptInvitation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (code: string) => orgService.acceptInvitation(code),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: QueryKeys.orgs() }),
  });
}

export function useLogin() {
  return useMutation({
    mutationFn: (params: { email: string; password: string }) =>
      authService.login(params.email, params.password),
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: (params: {
      email: string;
      nickname: string;
      password: string;
      invite_code?: string;
    }) => authService.register(params),
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => authService.logout(),
    onSettled: () => queryClient.clear(),
  });
}

export function useUpdateMemberRole(orgId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { userId: string; role: string }) =>
      orgService.updateMemberRole(orgId, params.userId, params.role),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: QueryKeys.members(orgId) }),
  });
}

export function useRemoveMember(orgId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => orgService.removeMember(orgId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.members(orgId) });
      queryClient.invalidateQueries({ queryKey: QueryKeys.usage(orgId) });
    },
  });
}

export function useCreateInvitation(orgId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { role: string; email?: string; max_uses?: number }) =>
      orgService.createInvitation(orgId, params),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: QueryKeys.invitations(orgId) }),
  });
}

export function useConsumeUsage(orgId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { meter: string; amount: number }) =>
      usageService.consume(orgId, params.meter, params.amount),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.usage(orgId) });
      queryClient.invalidateQueries({ queryKey: QueryKeys.usageEvents(orgId) });
    },
  });
}