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