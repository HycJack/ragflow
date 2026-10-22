// Service layer — each function maps to one backend endpoint. This is the
// RAGFlow `src/services/*-service` boundary: hooks call here, never fetch directly.

import { api } from '@/api/client';
import type {
  AdminOrg,
  AdminUser,
  Invitation,
  Member,
  Organization,
  Plan,
  Profile,
  UsageEvent,
  UsageMeter,
} from '@/api/types';

export interface AuthResult {
  token: string;
  profile: Profile;
  organizations?: Organization[];
}

// ---- Auth ----
export const authService = {
  register(params: {
    email: string;
    nickname: string;
    password: string;
    invite_code?: string;
  }) {
    return api.post<AuthResult>('/auth/register', params);
  },
  login(email: string, password: string) {
    return api.post<AuthResult>('/auth/login', { email, password });
  },
  logout() {
    return api.post<boolean>('/auth/logout');
  },
};

// ---- Organizations ----
export const orgService = {
  list() {
    return api.get<Organization[]>('/orgs');
  },
  get(orgId: string) {
    return api.get<Organization>(`/orgs/${orgId}`);
  },
  create(params: { name: string; slug?: string }) {
    return api.post<Organization>('/orgs', params);
  },
  members(orgId: string) {
    return api.get<Member[]>(`/orgs/${orgId}/members`);
  },
  updateMemberRole(orgId: string, userId: string, role: string) {
    return api.put<boolean>(`/orgs/${orgId}/members/${userId}/role`, { role });
  },
  removeMember(orgId: string, userId: string) {
    return api.delete<boolean>(`/orgs/${orgId}/members/${userId}`);
  },
  createInvitation(
    orgId: string,
    params: { role: string; email?: string; expires_at?: string; max_uses?: number },
  ) {
    return api.post<Invitation>(`/orgs/${orgId}/invitations`, params);
  },
  listInvitations(orgId: string) {
    return api.get<Invitation[]>(`/orgs/${orgId}/invitations`);
  },
  acceptInvitation(code: string) {
    return api.post<Organization>('/invitations/accept', { code });
  },
};

// ---- Plans ----
export const planService = {
  list() {
    return api.get<Plan[]>('/plans');
  },
};

// ---- Usage ----
export const usageService = {
  current(orgId: string) {
    return api.get<UsageMeter[]>(`/orgs/${orgId}/usage/current`);
  },
  events(orgId: string, meter?: string) {
    const q = meter ? `?meter=${meter}` : '';
    return api.get<UsageEvent[]>(`/orgs/${orgId}/usage/events${q}`);
  },
  consume(orgId: string, meter: string, amount: number) {
    return api.post<boolean>(`/orgs/${orgId}/usage/consume`, { meter, amount });
  },
};

// ---- Platform admin ----
export const adminService = {
  users() {
    return api.get<AdminUser[]>('/admin/users');
  },
  orgs() {
    return api.get<AdminOrg[]>('/admin/orgs');
  },
  setUserStatus(userId: string, status: string) {
    return api.patch<boolean>(`/admin/users/${userId}/status`, { status });
  },
  setOrgPlan(orgId: string, planCode: string) {
    return api.patch<boolean>(`/admin/orgs/${orgId}/plan`, { plan_code: planCode });
  },
  deleteOrg(orgId: string) {
    return api.delete<boolean>(`/admin/orgs/${orgId}`);
  },
};