// API data shapes matching the Go backend responses.

export interface Profile {
  id: string;
  email: string;
  nickname: string;
  avatar?: string | null;
  status: string;
  created_at: string;
}

export type OrgRole = 'owner' | 'admin' | 'member' | 'viewer';

export interface Organization {
  id: string;
  name: string;
  slug: string;
  avatar?: string | null;
  status: string;
  plan_code: string;
  owner_id: string;
  my_role: OrgRole;
  seat_used: number;
  seat_limit: number;
  created_at: string;
}

export interface Member {
  user_id: string;
  email: string;
  nickname: string;
  avatar?: string | null;
  role: OrgRole;
  status: string;
  joined_at: string;
}

export interface Invitation {
  id: string;
  org_id: string;
  code: string;
  role: OrgRole;
  email?: string | null;
  max_uses: number;
  used_count: number;
  expires_at?: string | null;
  status: string;
}

export interface UsageMeter {
  meter: string;
  used: number;
  limit: number; // -1 = unlimited
}

export interface UsageEvent {
  id: string;
  meter: string;
  amount: number;
  ref_kind?: string;
  ref_id?: string;
  billable: boolean;
  occurred_at: string;
}

export interface Plan {
  code: string;
  name: string;
  price_monthly_cents: number;
  storage_bytes: number;
  token_to_parse: number;
  seats: number;
  llm_calls: number;
}