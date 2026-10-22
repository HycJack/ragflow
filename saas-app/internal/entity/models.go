// Package entity defines the domain models of the SaaS platform.
//
// These mirror RAGFlow's entity layer but are kept framework-free so the
// backend runs on the standard library only (no external deps required).
package entity

import "time"

// Role constants.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// Meter constants.
const (
	MeterStorageBytes = "storage_bytes"
	MeterTokenToParse = "token_to_parse"
	MeterLLMCalls     = "llm_calls"
	MeterSeats        = "seats"
)

// User is a platform account.
type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"-"`
	Nickname        string    `json:"nickname"`
	Avatar          *string   `json:"avatar,omitempty"`
	Status          string    `json:"status"` // "1" active, "0" disabled
	IsPlatformAdmin bool      `json:"is_platform_admin"` // platform-level operator
	CreatedAt       time.Time `json:"created_at"`
}

// Organization is a SaaS workspace (multi-tenant grouping unit).
type Organization struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Slug     string  `json:"slug"`
	Avatar   *string `json:"avatar,omitempty"`
	PlanCode string  `json:"plan_code"`
	Status   string  `json:"status"` // "1" active, "0" suspended
	OwnerID  string  `json:"owner_id"`
	CreateAt time.Time `json:"created_at"`
}

// Membership joins a user to an organization with a role.
type Membership struct {
	ID        string  `json:"id"`
	OrgID     string  `json:"org_id"`
	UserID    string  `json:"user_id"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
	InvitedBy *string `json:"invited_by,omitempty"`
	CreateAt  time.Time `json:"created_at"`
}

// Invitation represents an invite link/code to join an organization.
type Invitation struct {
	ID        string     `json:"id"`
	OrgID     string     `json:"org_id"`
	Code      string     `json:"code"`
	Role      string     `json:"role"`
	Email     *string    `json:"email,omitempty"`
	MaxUses   int        `json:"max_uses"`
	UsedCount int        `json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Status    string     `json:"status"`
	CreateAt  time.Time  `json:"created_at"`
}

// Plan is a sellable plan with per-meter limits (-1 = unlimited).
type Plan struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	PriceMonthlyCents int64  `json:"price_monthly_cents"`
	StorageBytes      int64  `json:"storage_bytes"`
	TokenToParse      int64  `json:"token_to_parse"`
	Seats             int64  `json:"seats"`
	LLMCalls          int64  `json:"llm_calls"`
}

// UsageLedger tracks per-org, per-period meter consumption.
type UsageLedger struct {
	ID     string `json:"id"`
	OrgID  string `json:"org_id"`
	Period string `json:"period"` // "2026-05"
	Meter  string `json:"meter"`
	Used   int64  `json:"used"`
}

// UsageEvent is an append-only usage/audit record.
type UsageEvent struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	UserID     string    `json:"user_id"`
	Meter      string    `json:"meter"`
	Amount     int64     `json:"amount"`
	RefKind    string    `json:"ref_kind,omitempty"`
	RefID      string    `json:"ref_id,omitempty"`
	Billable   bool      `json:"billable"`
	OccurredAt time.Time `json:"occurred_at"`
}