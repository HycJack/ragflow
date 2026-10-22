package dao

import (
	"path/filepath"
	"testing"
	"time"

	"saas/internal/entity"
)

// TestOpenRestoresEverything verifies that a store reopened from disk rebuilds
// every primary record, every derived index, and the password hash that
// entity.User hides from JSON.
func TestOpenRestoresEverything(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("open empty store: %v", err)
	}

	user := &entity.User{Email: "persist@test.dev", Nickname: "P", PasswordHash: "hash-abc", Status: "1", CreatedAt: time.Now()}
	if err := s.CreateUser(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	org := &entity.Organization{Name: "Persist Co", Slug: "persist-co", PlanCode: "free", Status: "1", OwnerID: user.ID, CreateAt: time.Now()}
	orgID, err := s.CreateOrg(org)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	if err := s.CreateMembership(&entity.Membership{OrgID: orgID, UserID: user.ID, Role: entity.RoleOwner, Status: "1", CreateAt: time.Now()}); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	invID, err := s.CreateInvitation(&entity.Invitation{OrgID: orgID, Code: "CODE123", Role: entity.RoleMember, MaxUses: 3, Status: "1", CreateAt: time.Now()})
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	if ok, err := s.IncrementInvitationUsed(invID, 3); err != nil || !ok {
		t.Fatalf("use invitation: ok=%v err=%v", ok, err)
	}
	if ok, err := s.TryConsumeLedger(orgID, "2026-05", entity.MeterStorageBytes, 42, 100); err != nil || !ok {
		t.Fatalf("consume ledger: ok=%v err=%v", ok, err)
	}
	if err := s.AppendUsageEvent(orgID, &entity.UsageEvent{OrgID: orgID, UserID: user.ID, Meter: entity.MeterStorageBytes, Amount: 42, Billable: true, OccurredAt: time.Now()}); err != nil {
		t.Fatalf("append event: %v", err)
	}

	// Reopen: this is the "server restart" path.
	reloaded, err := Open(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}

	got, err := reloaded.GetUserByEmail("persist@test.dev")
	if err != nil {
		t.Fatalf("user by email after reload: %v", err)
	}
	if got.PasswordHash != "hash-abc" {
		t.Fatalf("password hash not persisted: got %q", got.PasswordHash)
	}

	if _, err := reloaded.GetOrgBySlug("persist-co"); err != nil {
		t.Fatalf("org by slug after reload: %v", err)
	}
	if m, err := reloaded.GetMembership(orgID, user.ID); err != nil || m.Role != entity.RoleOwner {
		t.Fatalf("membership after reload: %+v err=%v", m, err)
	}
	inv, err := reloaded.GetInvitationByCode("CODE123")
	if err != nil {
		t.Fatalf("invitation by code after reload: %v", err)
	}
	if inv.UsedCount != 1 {
		t.Fatalf("invitation used_count not persisted: got %d", inv.UsedCount)
	}
	ledger := reloaded.GetLedger(orgID, "2026-05")
	if len(ledger) != 1 || ledger[0].Used != 42 {
		t.Fatalf("ledger after reload: %+v", ledger)
	}
	if events := reloaded.ListUsageEvents(orgID, "", 0, 10); len(events) != 1 {
		t.Fatalf("events after reload: %+v", events)
	}
}

// TestOpenMissingFileStartsEmpty covers first boot.
func TestOpenMissingFileStartsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "store.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open missing store: %v", err)
	}
	if ids := s.ListUserIDs(); len(ids) != 0 {
		t.Fatalf("expected empty store, got %d users", len(ids))
	}
}

// TestMemoryStoreWritesNothing asserts the zero-value path stays memory-only.
func TestMemoryStoreWritesNothing(t *testing.T) {
	s := NewStore()
	if s.Path() != "" {
		t.Fatalf("expected memory-only store, got path %q", s.Path())
	}
	if err := s.CreateUser(&entity.User{Email: "a@b.c", Status: "1", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("create user: %v", err)
	}
}
