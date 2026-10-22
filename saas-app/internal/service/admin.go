// Package service: platform-admin views for the SaaS operator console.
package service

import (
	"fmt"
	"time"

	"saas/internal/dao"
	"saas/internal/entity"
)

// ErrNotPlatformAdmin is returned when a non-admin calls an admin endpoint.
var ErrNotPlatformAdmin = fmt.Errorf("platform administrator privileges required")

// AdminUserView is the platform-scope view of a user account.
type AdminUserView struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Nickname        string    `json:"nickname"`
	Status          string    `json:"status"`
	IsPlatformAdmin bool      `json:"is_platform_admin"`
	CreatedAt       time.Time `json:"created_at"`
	OrgCount        int       `json:"org_count"`
}

// AdminOrgView is the platform-scope view of an organization.
type AdminOrgView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	PlanCode    string `json:"plan_code"`
	OwnerID     string `json:"owner_id"`
	MemberCount int64  `json:"member_count"`
	SeatLimit   int64  `json:"seat_limit"`
	CreatedAt   string `json:"created_at"`
}

// PlatformAdminService exposes read-only platform overviews. Every method
// requires the caller to be a platform administrator.
type PlatformAdminService struct {
	store *dao.Store
	plans *PlanService
}

// NewPlatformAdminService creates the admin service.
func NewPlatformAdminService(store *dao.Store, plans *PlanService) *PlatformAdminService {
	return &PlatformAdminService{store: store, plans: plans}
}

// ensureAdmin returns an error unless the caller is a platform admin.
func (s *PlatformAdminService) ensureAdmin(user *entity.User) error {
	if user == nil || !user.IsPlatformAdmin {
		return ErrNotPlatformAdmin
	}
	return nil
}

// ListUsers returns every platform account with its org membership count.
func (s *PlatformAdminService) ListUsers(actor *entity.User) ([]*AdminUserView, error) {
	if err := s.ensureAdmin(actor); err != nil {
		return nil, err
	}
	users := s.store.ListUsers()
	out := make([]*AdminUserView, 0, len(users))
	for _, u := range users {
		out = append(out, &AdminUserView{
			ID: u.ID, Email: u.Email, Nickname: u.Nickname,
			Status: u.Status, IsPlatformAdmin: u.IsPlatformAdmin,
			CreatedAt: u.CreatedAt, OrgCount: len(s.store.ListMembershipsByUser(u.ID)),
		})
	}
	return out, nil
}

// ListOrgs returns every organization with member counts and plan limits.
func (s *PlatformAdminService) ListOrgs(actor *entity.User) ([]*AdminOrgView, error) {
	if err := s.ensureAdmin(actor); err != nil {
		return nil, err
	}
	orgs := s.store.ListOrgsByIDs(s.store.ListOrgIDs())
	out := make([]*AdminOrgView, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, &AdminOrgView{
			ID: o.ID, Name: o.Name, Slug: o.Slug, PlanCode: o.PlanCode,
			OwnerID: o.OwnerID, MemberCount: s.store.CountMembersByOrg(o.ID),
			SeatLimit: s.plans.Plan(o.PlanCode).Seats,
			CreatedAt: o.CreateAt.Format(time.RFC3339),
		})
	}
	return out, nil
}

// SetUserStatus enables ("1") or disables ("0") an account. A platform
// administrator cannot be disabled here: locking the operator console out of
// its own accounts needs a deliberate out-of-band change, not a UI click.
func (s *PlatformAdminService) SetUserStatus(actor *entity.User, userID, status string) error {
	if err := s.ensureAdmin(actor); err != nil {
		return err
	}
	if status != "0" && status != "1" {
		return invalid("status must be %q or %q", "0", "1")
	}
	target, err := s.store.GetUserByID(userID)
	if err != nil {
		return ErrNotFound
	}
	if target.IsPlatformAdmin {
		return invalid("cannot change the status of a platform administrator")
	}
	return s.store.SetUserStatus(userID, status)
}

// SetOrgPlan moves an organization onto another plan.
func (s *PlatformAdminService) SetOrgPlan(actor *entity.User, orgID, planCode string) error {
	if err := s.ensureAdmin(actor); err != nil {
		return err
	}
	if !s.plans.Has(planCode) {
		return invalid("unknown plan %q", planCode)
	}
	if _, err := s.store.GetOrgByID(orgID); err != nil {
		return ErrNotFound
	}
	return s.store.SetOrgPlan(orgID, planCode)
}

// DeleteOrg removes an organization and its memberships, invitations and usage.
func (s *PlatformAdminService) DeleteOrg(actor *entity.User, orgID string) error {
	if err := s.ensureAdmin(actor); err != nil {
		return err
	}
	return s.store.DeleteOrg(orgID)
}