// Package dao is the data-access layer for the SaaS platform.
//
// It ships an in-memory implementation (Store) so the vertical slice runs with
// zero external dependencies. The method signatures are intentionally DB-shaped
// so a SQL/GORM backend can be added later without touching the service layer.
package dao

import (
	"errors"
	"sort"
	"sync"
	"time"

	"saas/internal/entity"
)

var (
	// ErrNotFound is returned when a record does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when a unique constraint would be violated.
	ErrConflict = errors.New("conflict")
)

// Store is an in-memory, concurrency-safe data store.
type Store struct {
	mu       sync.RWMutex
	now      func() time.Time
	seq      int64
	users    map[string]*entity.User
	orgs     map[string]*entity.Organization
	byEmail  map[string]string // email -> user id
	bySlug   map[string]string // slug -> org id
	members  map[string][]*entity.Membership
	byMember map[string]*entity.Membership // id -> membership
	invites  map[string]*entity.Invitation
	byCode   map[string]string // invite code -> invite id
	ledger   map[string]*entity.UsageLedger
	usageBy  map[string][]*entity.UsageLedger // "orgID/period" -> ledger rows
	events   map[string][]*entity.UsageEvent
}

// NewStore creates an empty in-memory store with default plans seeded.
func NewStore() *Store {
	s := &Store{
		now:      time.Now,
		users:    map[string]*entity.User{},
		orgs:     map[string]*entity.Organization{},
		byEmail:  map[string]string{},
		bySlug:   map[string]string{},
		members:  map[string][]*entity.Membership{},
		byMember: map[string]*entity.Membership{},
		invites:  map[string]*entity.Invitation{},
		byCode:   map[string]string{},
		ledger:   map[string]*entity.UsageLedger{},
		usageBy:  map[string][]*entity.UsageLedger{},
		events:   map[string][]*entity.UsageEvent{},
	}
	return s
}

func (s *Store) nextID(prefix string) string {
	s.seq++
	return prefix + "-" + time.Now().UTC().Format("20060102T150405") + "-" + itoa(s.seq)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// ---- Users ----

// CreateUser inserts a user.
func (s *Store) CreateUser(u *entity.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byEmail[u.Email]; ok {
		return ErrConflict
	}
	u.ID = s.nextID("usr")
	s.users[u.ID] = u
	s.byEmail[u.Email] = u.ID
	return nil
}

// GetUserByEmail loads a user by email.
func (s *Store) GetUserByEmail(email string) (*entity.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byEmail[email]
	if !ok {
		return nil, ErrNotFound
	}
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneUser(u), nil
}

// GetUserByID loads a user by id.
func (s *Store) GetUserByID(id string) (*entity.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneUser(u), nil
}

// ListUserIDs returns all user ids.
func (s *Store) ListUserIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.users))
	for id := range s.users {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// ---- Organizations ----

// CreateOrg inserts an organization (with slug uniqueness).
func (s *Store) CreateOrg(o *entity.Organization) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.bySlug[o.Slug]; ok {
		return "", ErrConflict
	}
	o.ID = s.nextID("org")
	s.orgs[o.ID] = o
	s.bySlug[o.Slug] = o.ID
	return o.ID, nil
}

// GetOrgByID loads an organization.
func (s *Store) GetOrgByID(id string) (*entity.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orgs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneOrg(o), nil
}

// GetOrgBySlug loads an organization by slug.
func (s *Store) GetOrgBySlug(slug string) (*entity.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.bySlug[slug]
	if !ok {
		return nil, ErrNotFound
	}
	o, ok := s.orgs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneOrg(o), nil
}

// ListOrgsByIDs loads many organizations.
func (s *Store) ListOrgsByIDs(ids []string) []*entity.Organization {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*entity.Organization, 0, len(ids))
	for _, id := range ids {
		if o, ok := s.orgs[id]; ok {
			out = append(out, cloneOrg(o))
		}
	}
	return out
}

// ---- Memberships ----

// CreateMembership inserts a membership.
func (s *Store) CreateMembership(m *entity.Membership) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ex := range s.members[m.OrgID] {
		if ex.UserID == m.UserID && ex.Status == "1" {
			return ErrConflict
		}
	}
	m.ID = s.nextID("mem")
	s.members[m.OrgID] = append(s.members[m.OrgID], m)
	s.byMember[m.ID] = m
	return nil
}

// GetMembership loads a membership for an org+user.
func (s *Store) GetMembership(orgID, userID string) (*entity.Membership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.members[orgID] {
		if m.UserID == userID {
			return cloneMember(m), nil
		}
	}
	return nil, ErrNotFound
}

// ListMembershipsByUser loads all active memberships for a user.
func (s *Store) ListMembershipsByUser(userID string) []*entity.Membership {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*entity.Membership
	for _, list := range s.members {
		for _, m := range list {
			if m.UserID == userID && m.Status == "1" {
				out = append(out, cloneMember(m))
			}
		}
	}
	return out
}

// ListMembersByOrg loads all members of an org.
func (s *Store) ListMembersByOrg(orgID string) []*entity.Membership {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*entity.Membership, 0, len(s.members[orgID]))
	for _, m := range s.members[orgID] {
		out = append(out, cloneMember(m))
	}
	return out
}

// CountMembersByOrg counts active members of an org.
func (s *Store) CountMembersByOrg(orgID string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var n int64
	for _, m := range s.members[orgID] {
		if m.Status == "1" {
			n++
		}
	}
	return n
}

// UpdateMemberRole sets a member's role.
func (s *Store) UpdateMemberRole(orgID, userID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.members[orgID] {
		if m.UserID == userID {
			m.Role = role
			return nil
		}
	}
	return ErrNotFound
}

// RemoveMembership soft-deletes a membership.
func (s *Store) RemoveMembership(orgID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.members[orgID] {
		if m.UserID == userID {
			m.Status = "0"
			return nil
		}
	}
	return ErrNotFound
}

// ---- Invitations ----

// CreateInvitation inserts an invitation.
func (s *Store) CreateInvitation(inv *entity.Invitation) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byCode[inv.Code]; ok {
		return "", ErrConflict
	}
	inv.ID = s.nextID("inv")
	s.invites[inv.ID] = inv
	s.byCode[inv.Code] = inv.ID
	return inv.ID, nil
}

// GetInvitationByCode loads an active invitation by code.
func (s *Store) GetInvitationByCode(code string) (*entity.Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byCode[code]
	if !ok {
		return nil, ErrNotFound
	}
	inv, ok := s.invites[id]
	if !ok || inv.Status != "1" {
		return nil, ErrNotFound
	}
	return cloneInvitation(inv), nil
}

// ListInvitationsByOrg loads invites for an org.
func (s *Store) ListInvitationsByOrg(orgID string) []*entity.Invitation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*entity.Invitation
	for _, inv := range s.invites {
		if inv.OrgID == orgID {
			out = append(out, cloneInvitation(inv))
		}
	}
	return out
}

// IncrementInvitationUsed atomically bumps the used counter within max_uses.
func (s *Store) IncrementInvitationUsed(id string, maxUses int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.invites[id]
	if !ok {
		return false, ErrNotFound
	}
	if inv.UsedCount >= maxUses {
		return false, nil
	}
	inv.UsedCount++
	return true, nil
}

// ---- Usage ----

// TryConsumeLedger consumes `amount` of a meter within `limit`
// (limit < 0 = unlimited). Returns false when the limit would be exceeded.
func (s *Store) TryConsumeLedger(orgID, period, meter string, amount, limit int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := orgID + "/" + period
	for _, l := range s.usageBy[key] {
		if l.Meter == meter {
			if limit >= 0 && l.Used+amount > limit {
				return false, nil
			}
			l.Used += amount
			return true, nil
		}
	}
	if limit >= 0 && amount > limit {
		return false, nil
	}
	l := &entity.UsageLedger{ID: s.nextID("led"), OrgID: orgID, Period: period, Meter: meter, Used: amount}
	s.ledger[l.ID] = l
	s.usageBy[key] = append(s.usageBy[key], l)
	return true, nil
}

// GetLedger returns the current consumption for a period.
func (s *Store) GetLedger(orgID, period string) []*entity.UsageLedger {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := orgID + "/" + period
	out := make([]*entity.UsageLedger, 0, len(s.usageBy[key]))
	for _, l := range s.usageBy[key] {
		lc := *l
		out = append(out, &lc)
	}
	return out
}

// AppendUsageEvent records a usage/audit event.
func (s *Store) AppendUsageEvent(orgID string, ev *entity.UsageEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ev.ID = s.nextID("evt")
	s.events[orgID] = append([]*entity.UsageEvent{ev}, s.events[orgID]...)
}

// ListUsageEvents returns usage events for an org (new->old), optionally
// filtered by meter, paginated.
func (s *Store) ListUsageEvents(orgID, meter string, offset, limit int) []*entity.UsageEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := s.events[orgID]
	out := make([]*entity.UsageEvent, 0, len(all))
	for _, ev := range all {
		if meter != "" && ev.Meter != meter {
			continue
		}
		out = append(out, ev)
	}
	if offset > len(out) {
		offset = len(out)
	}
	out = out[offset:]
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out
}

// clone helpers return deep-enough copies (slices are fine, they're not shared).
func cloneUser(u *entity.User) *entity.User { c := *u; return &c }
func cloneOrg(o *entity.Organization) *entity.Organization { c := *o; return &c }
func cloneMember(m *entity.Membership) *entity.Membership {
	c := *m
	if m.InvitedBy != nil {
		v := *m.InvitedBy
		c.InvitedBy = &v
	}
	return &c
}
func cloneInvitation(i *entity.Invitation) *entity.Invitation {
	c := *i
	if i.Email != nil {
		v := *i.Email
		c.Email = &v
	}
	if i.ExpiresAt != nil {
		v := *i.ExpiresAt
		c.ExpiresAt = &v
	}
	return &c
}