package service

import (
	"errors"
	"testing"

	"saas/internal/dao"
	"saas/internal/entity"
)

func newTestApp() (*dao.Store, *AuthService, *OrganizationService, *UsageService) {
	store := dao.NewStore()
	plans := NewPlanService()
	orgSvc := NewOrganizationService(store, plans)
	return store, NewAuthService(store, plans), orgSvc, NewUsageService(store, orgSvc)
}

func register(t *testing.T, auth *AuthService, email string) *entity.User {
	t.Helper()
	_, user, err := auth.Register(&RegisterRequest{Email: email, Nickname: "User " + email, Password: "secret123"})
	if err != nil {
		t.Fatalf("register %s: %v", email, err)
	}
	return user
}

func TestCreateOrgAndRBAC(t *testing.T) {
	_, auth, orgSvc, _ := newTestApp()
	owner := register(t, auth, "owner@test.dev")
	member := register(t, auth, "member@test.dev")

	org, err := orgSvc.CreateOrg(owner, &CreateOrgRequest{Name: "Acme Inc"})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	if org.MyRole != entity.RoleOwner {
		t.Fatalf("expected owner role, got %s", org.MyRole)
	}

	// Member cannot list members (not a member yet).
	if _, err := orgSvc.ListMembers(member.ID, org.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}

	// Owner invites member, member accepts.
	inv, err := orgSvc.CreateInvitation(owner.ID, org.ID, &CreateInvitationRequest{Role: entity.RoleMember})
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	ov, err := orgSvc.AcceptInvitation(member, inv.Code)
	if err != nil {
		t.Fatalf("accept invitation: %v", err)
	}
	if ov.MyRole != entity.RoleMember {
		t.Fatalf("expected member role after accept, got %s", ov.MyRole)
	}
	// Now member can list.
	if _, err := orgSvc.ListMembers(member.ID, org.ID); err != nil {
		t.Fatalf("member should list after joining: %v", err)
	}

	// Member cannot invite.
	if _, err := orgSvc.CreateInvitation(member.ID, org.ID, &CreateInvitationRequest{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for member invite, got %v", err)
	}

	// Owner promotes member to admin.
	if err := orgSvc.UpdateMemberRole(owner.ID, org.ID, member.ID, entity.RoleAdmin); err != nil {
		t.Fatalf("promote to admin: %v", err)
	}
	// Admin can now invite.
	if _, err := orgSvc.CreateInvitation(member.ID, org.ID, &CreateInvitationRequest{}); err != nil {
		t.Fatalf("admin should invite, got %v", err)
	}
}

func TestInvitationMaxUses(t *testing.T) {
	_, auth, orgSvc, _ := newTestApp()
	owner := register(t, auth, "owner2@test.dev")
	userA := register(t, auth, "a@test.dev")
	userB := register(t, auth, "b@test.dev")

	org, err := orgSvc.CreateOrg(owner, &CreateOrgRequest{Name: "Limited"})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	maxUses := 1
	inv, err := orgSvc.CreateInvitation(owner.ID, org.ID, &CreateInvitationRequest{Role: entity.RoleMember, MaxUses: &maxUses})
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	if _, err := orgSvc.AcceptInvitation(userA, inv.Code); err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if _, err := orgSvc.AcceptInvitation(userB, inv.Code); err == nil {
		t.Fatalf("expected second accept to fail (max_uses=1)")
	}
}

func TestUsageQuotaExceeded(t *testing.T) {
	_, auth, orgSvc, usage := newTestApp()
	owner := register(t, auth, "owner3@test.dev")
	org, err := orgSvc.CreateOrg(owner, &CreateOrgRequest{Name: "Quota"})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	freePlan := NewPlanService().Plan("free")

	if err := usage.Consume(org.ID, entity.MeterStorageBytes, freePlan.StorageBytes-1); err != nil {
		t.Fatalf("consume under limit: %v", err)
	}
	if err := usage.Consume(org.ID, entity.MeterStorageBytes, 2); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected quota exceeded, got %v", err)
	}

	snaps, err := usage.CurrentUsage(owner.ID, org.ID)
	if err != nil {
		t.Fatalf("current usage: %v", err)
	}
	var storage *UsageMeterSnapshot
	for i := range snaps {
		if snaps[i].Meter == entity.MeterStorageBytes {
			storage = snaps[i]
		}
	}
	if storage == nil || storage.Used != freePlan.StorageBytes-1 {
		t.Fatalf("unexpected storage meter: %+v", storage)
	}
}