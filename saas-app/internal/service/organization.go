package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"saas/internal/crypto"
	"saas/internal/dao"
	"saas/internal/entity"
)

var (
	// ErrForbidden is returned when a caller lacks the required role.
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound is returned when a resource is missing.
	ErrNotFound = dao.ErrNotFound
)

// OrganizationService implements workspace lifecycle, membership and
// invitations with an org-unit RBAC hierarchy (owner > admin > member > viewer).
type OrganizationService struct {
	store *dao.Store
	plans *PlanService
}

// NewOrganizationService creates an OrganizationService.
func NewOrganizationService(store *dao.Store, plans *PlanService) *OrganizationService {
	return &OrganizationService{store: store, plans: plans}
}

// OrganizationView is the API shape of an organization plus the caller's role.
type OrganizationView struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	Avatar    *string `json:"avatar,omitempty"`
	Status    string  `json:"status"`
	PlanCode  string  `json:"plan_code"`
	OwnerID   string  `json:"owner_id"`
	MyRole    string  `json:"my_role"`
	SeatUsed  int64   `json:"seat_used"`
	SeatLimit int64   `json:"seat_limit"`
	CreateAt  string  `json:"created_at"`
}

// PlanLimitsView exposes a workspace's current quota snapshot.
type PlanLimitsView struct {
	PlanCode     string `json:"plan_code"`
	StorageBytes int64  `json:"storage_bytes"`
	TokenToParse int64  `json:"token_to_parse"`
	Seats        int64  `json:"seats"`
	LLMCalls     int64  `json:"llm_calls"`
}

// CreateOrgRequest carries creation input.
type CreateOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreateOrg creates a workspace owned by the user.
func (s *OrganizationService) CreateOrg(user *entity.User, req *CreateOrgRequest) (*OrganizationView, error) {
	// A user who has already been invited into a workspace (holds a non-owner
	// seat) is not allowed to bootstrap their own organization. They belong to
	// that workspace as a member rather than self-provisioning a second one
	// where they would be owner. A brand-new user (no memberships) or an
	// existing owner may still create organizations.
	for _, m := range s.store.ListMembershipsByUser(user.ID) {
		if m.Role != entity.RoleOwner {
			return nil, fmt.Errorf("you are already a member of a workspace; you cannot create your own organization")
		}
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("organization name is required")
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = slugify(name)
	}
	org := &entity.Organization{
		Name:     name,
		Slug:     slug,
		PlanCode: "free",
		Status:   "1",
		OwnerID:  user.ID,
		CreateAt: time.Now(),
	}
	orgID, err := s.store.CreateOrg(org)
	if err != nil {
		if errors.Is(err, dao.ErrConflict) {
			return nil, fmt.Errorf("an organization with slug %q already exists", slug)
		}
		return nil, err
	}
	m := &entity.Membership{
		OrgID: orgID, UserID: user.ID, Role: entity.RoleOwner, Status: "1",
		InvitedBy: &user.ID, CreateAt: time.Now(),
	}
	if err := s.store.CreateMembership(m); err != nil {
		return nil, err
	}
	return s.ViewFor(user.ID, orgID)
}

// ViewFor renders an org view for a user (must be a member).
func (s *OrganizationService) ViewFor(userID, orgID string) (*OrganizationView, error) {
	m, err := s.RequireMembership(userID, orgID)
	if err != nil {
		return nil, err
	}
	o, err := s.store.GetOrgByID(orgID)
	if err != nil {
		return nil, err
	}
	plan := s.plans.Plan(o.PlanCode)
	return &OrganizationView{
		ID: o.ID, Name: o.Name, Slug: o.Slug, Avatar: o.Avatar, Status: o.Status,
		PlanCode: o.PlanCode, OwnerID: o.OwnerID, MyRole: m.Role,
		SeatUsed: s.store.CountMembersByOrg(orgID), SeatLimit: plan.Seats,
		CreateAt: o.CreateAt.Format(time.RFC3339),
	}, nil
}

// ListByUser lists all workspaces the user belongs to.
func (s *OrganizationService) ListByUser(userID string) ([]*OrganizationView, error) {
	ms := s.store.ListMembershipsByUser(userID)
	ids := make([]string, 0, len(ms))
	for _, m := range ms {
		ids = append(ids, m.OrgID)
	}
	orgs := s.store.ListOrgsByIDs(ids)
	out := make([]*OrganizationView, 0, len(orgs))
	for _, o := range orgs {
		v, err := s.ViewByUnchecked(o)
		if err != nil {
			continue
		}
		v.MyRole = roleOf(ms, o.ID)
		out = append(out, v)
	}
	return out, nil
}

// ViewByUnchecked renders a view for an org (membership already known).
func (s *OrganizationService) ViewByUnchecked(o *entity.Organization) (*OrganizationView, error) {
	plan := s.plans.Plan(o.PlanCode)
	return &OrganizationView{
		ID: o.ID, Name: o.Name, Slug: o.Slug, Avatar: o.Avatar, Status: o.Status,
		PlanCode: o.PlanCode, OwnerID: o.OwnerID,
		SeatUsed: s.store.CountMembersByOrg(o.ID), SeatLimit: plan.Seats,
		CreateAt: o.CreateAt.Format(time.RFC3339),
	}, nil
}

// RequireMembership loads the membership and rejects non-members/non-active orgs.
func (s *OrganizationService) RequireMembership(userID, orgID string) (*entity.Membership, error) {
	m, err := s.store.GetMembership(orgID, userID)
	if err != nil {
		return nil, ErrForbidden
	}
	if m.Status != "1" {
		return nil, ErrForbidden
	}
	o, err := s.store.GetOrgByID(orgID)
	if err != nil {
		return nil, ErrNotFound
	}
	if o.Status == "0" {
		return nil, fmt.Errorf("this organization has been suspended")
	}
	return m, nil
}

// MemberView is the API shape of a membership joined with user identity.
type MemberView struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	Avatar    *string `json:"avatar,omitempty"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	JoinAt    string `json:"joined_at"`
}

// ListMembers lists members of an org (any member can view).
func (s *OrganizationService) ListMembers(userID, orgID string) ([]*MemberView, error) {
	if _, err := s.RequireMembership(userID, orgID); err != nil {
		return nil, err
	}
	ms := s.store.ListMembersByOrg(orgID)
	out := make([]*MemberView, 0, len(ms))
	for _, m := range ms {
		u, _ := s.store.GetUserByID(m.UserID)
		v := &MemberView{UserID: m.UserID, Role: m.Role, Status: m.Status,
			JoinAt: m.CreateAt.Format(time.RFC3339)}
		if u != nil {
			v.Email = u.Email
			v.Nickname = u.Nickname
			v.Avatar = u.Avatar
		}
		out = append(out, v)
	}
	return out, nil
}

// CreateInvitationRequest carries input for minting an invite.
type CreateInvitationRequest struct {
	Role      string  `json:"role"`
	Email     *string `json:"email,omitempty"`
	ExpiresAt *string `json:"expires_at,omitempty"`
	MaxUses   *int    `json:"max_uses,omitempty"`
}

// InvitationView is the API shape of an invitation.
type InvitationView struct {
	ID        string  `json:"id"`
	OrgID     string  `json:"org_id"`
	Code      string  `json:"code"`
	Role      string  `json:"role"`
	Email     *string `json:"email,omitempty"`
	MaxUses   int     `json:"max_uses"`
	UsedCount int     `json:"used_count"`
	ExpiresAt *string `json:"expires_at,omitempty"`
	Status    string  `json:"status"`
}

// CreateInvitation mints an invite (owner/admin only).
func (s *OrganizationService) CreateInvitation(actorID, orgID string, req *CreateInvitationRequest) (*InvitationView, error) {
	m, err := s.RequireMembership(actorID, orgID)
	if err != nil {
		return nil, err
	}
	if !canInvite(m.Role) {
		return nil, ErrForbidden
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = entity.RoleMember
	}
	if !validRole(role) {
		return nil, fmt.Errorf("invalid role: %s", role)
	}
	maxUses := 1
	if req.MaxUses != nil && *req.MaxUses > 0 {
		maxUses = *req.MaxUses
	}
	expiresAt := time.Now().AddDate(0, 0, 7)
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		if t, perr := time.Parse(time.RFC3339, *req.ExpiresAt); perr == nil {
			expiresAt = t
		}
	}
	inv := &entity.Invitation{
		OrgID: orgID, Code: crypto.GenerateCode(), Role: role,
		Email: req.Email, MaxUses: maxUses, UsedCount: 0,
		ExpiresAt: &expiresAt, Status: "1", CreateAt: time.Now(),
	}
	if _, err := s.store.CreateInvitation(inv); err != nil {
		return nil, err
	}
	return s.toInvitationView(inv), nil
}

// ListInvitations lists invites of an org (owner/admin only).
func (s *OrganizationService) ListInvitations(actorID, orgID string) ([]*InvitationView, error) {
	m, err := s.RequireMembership(actorID, orgID)
	if err != nil {
		return nil, err
	}
	if !canInvite(m.Role) {
		return nil, ErrForbidden
	}
	invs := s.store.ListInvitationsByOrg(orgID)
	out := make([]*InvitationView, 0, len(invs))
	for _, inv := range invs {
		out = append(out, s.toInvitationView(inv))
	}
	return out, nil
}

// AcceptInvitation joins the current user to the org referenced by the code.
func (s *OrganizationService) AcceptInvitation(user *entity.User, code string) (*OrganizationView, error) {
	inv, err := s.store.GetInvitationByCode(code)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired invitation code")
	}
	if inv.ExpiresAt != nil && time.Now().After(*inv.ExpiresAt) {
		return nil, fmt.Errorf("invitation has expired")
	}
	if inv.Email != nil && *inv.Email != "" && !strings.EqualFold(*inv.Email, user.Email) {
		return nil, fmt.Errorf("this invitation is for a specific email address")
	}
	// Idempotent: already a member.
	if _, err := s.store.GetMembership(inv.OrgID, user.ID); err == nil {
		return s.ViewFor(user.ID, inv.OrgID)
	}
	ok, err := s.store.IncrementInvitationUsed(inv.ID, inv.MaxUses)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invitation has been used up")
	}
	err = s.store.CreateMembership(&entity.Membership{
		OrgID: inv.OrgID, UserID: user.ID, Role: inv.Role, Status: "1",
		InvitedBy: nil, CreateAt: time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return s.ViewFor(user.ID, inv.OrgID)
}

// UpdateMemberRole changes a member's role (owner only).
func (s *OrganizationService) UpdateMemberRole(actorID, orgID, targetUserID, role string) error {
	acting, err := s.RequireMembership(actorID, orgID)
	if err != nil {
		return err
	}
	if !validRole(role) {
		return fmt.Errorf("invalid role: %s", role)
	}
	if acting.Role != entity.RoleOwner {
		return ErrForbidden
	}
	if _, err := s.store.GetMembership(orgID, targetUserID); err != nil {
		return ErrNotFound
	}
	return s.store.UpdateMemberRole(orgID, targetUserID, role)
}

// RemoveMember removes a user from a workspace (owner/admin only; not self).
func (s *OrganizationService) RemoveMember(actorID, orgID, targetUserID string) error {
	acting, err := s.RequireMembership(actorID, orgID)
	if err != nil {
		return err
	}
	if !canInvite(acting.Role) {
		return ErrForbidden
	}
	if acting.UserID == targetUserID {
		return fmt.Errorf("you cannot remove yourself")
	}
	return s.store.RemoveMembership(orgID, targetUserID)
}

// LimitsFor returns the plan quota for an org.
func (s *OrganizationService) LimitsFor(orgID string) (PlanLimitsView, error) {
	o, err := s.store.GetOrgByID(orgID)
	if err != nil {
		return PlanLimitsView{}, err
	}
	p := s.plans.Plan(o.PlanCode)
	return PlanLimitsView{PlanCode: o.PlanCode, StorageBytes: p.StorageBytes,
		TokenToParse: p.TokenToParse, Seats: p.Seats, LLMCalls: p.LLMCalls}, nil
}

func (s *OrganizationService) toInvitationView(inv *entity.Invitation) *InvitationView {
	v := &InvitationView{ID: inv.ID, OrgID: inv.OrgID, Code: inv.Code, Role: inv.Role,
		Email: inv.Email, MaxUses: inv.MaxUses, UsedCount: inv.UsedCount, Status: inv.Status}
	if inv.ExpiresAt != nil {
		str := inv.ExpiresAt.Format(time.RFC3339)
		v.ExpiresAt = &str
	}
	return v
}

func roleOf(ms []*entity.Membership, orgID string) string {
	for _, m := range ms {
		if m.OrgID == orgID {
			return m.Role
		}
	}
	return ""
}

func canInvite(role string) bool {
	return role == entity.RoleOwner || role == entity.RoleAdmin
}

func validRole(role string) bool {
	return role == entity.RoleOwner || role == entity.RoleAdmin ||
		role == entity.RoleMember || role == entity.RoleViewer
}

func slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r == ' ', r == '\t':
			return '-'
		default:
			return -1
		}
	}, slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = fmt.Sprintf("org-%d", time.Now().Unix())
	}
	return slug
}