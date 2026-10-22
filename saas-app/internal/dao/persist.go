package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"saas/internal/entity"
)

// storeVersion guards against silently loading an incompatible layout.
const storeVersion = 1

// snapshot is the durable representation of the store. Only primary
// collections are written; every in-memory index (email, slug, code, member,
// period grouping) is rebuilt from them on load, so the file cannot drift out
// of sync with the indexes.
type snapshot struct {
	Version     int                  `json:"version"`
	SavedAt     time.Time            `json:"saved_at"`
	Seq         int64                `json:"seq"`
	Users       []userRow            `json:"users"`
	Orgs        []*entity.Organization `json:"organizations"`
	Memberships []*entity.Membership `json:"memberships"`
	Invitations []*entity.Invitation `json:"invitations"`
	Ledger      []*entity.UsageLedger `json:"usage_ledger"`
	Events      []*entity.UsageEvent  `json:"usage_events"`
}

// userRow mirrors entity.User plus the password hash. entity.User hides the
// hash with `json:"-"` because the struct is also used for API payloads, so
// persistence needs its own representation rather than relaxing that tag.
type userRow struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"password_hash"`
	Nickname        string    `json:"nickname"`
	Avatar          *string   `json:"avatar,omitempty"`
	Status          string    `json:"status"`
	IsPlatformAdmin bool      `json:"is_platform_admin"`
	CreatedAt       time.Time `json:"created_at"`
}

func userToRow(u *entity.User) userRow {
	return userRow{
		ID: u.ID, Email: u.Email, PasswordHash: u.PasswordHash,
		Nickname: u.Nickname, Avatar: u.Avatar, Status: u.Status,
		IsPlatformAdmin: u.IsPlatformAdmin, CreatedAt: u.CreatedAt,
	}
}

func (r userRow) toEntity() *entity.User {
	return &entity.User{
		ID: r.ID, Email: r.Email, PasswordHash: r.PasswordHash,
		Nickname: r.Nickname, Avatar: r.Avatar, Status: r.Status,
		IsPlatformAdmin: r.IsPlatformAdmin, CreatedAt: r.CreatedAt,
	}
}

// Open loads the store persisted at path. A missing file yields an empty
// store, so first boot needs no provisioning step. From then on every
// mutating method rewrites the file before returning.
//
// Durability model: each mutation writes the whole snapshot to a temporary
// file, fsyncs it, then renames it over the previous one. A crash therefore
// leaves either the old or the new snapshot intact, never a truncated one.
// The cost is O(size of the dataset) per write, which is acceptable for the
// demo dataset this slice ships with; a SQL backend is the answer at scale,
// and the Store method signatures are already shaped for it.
func Open(path string) (*Store, error) {
	s := NewStore()
	s.path = path

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read store %q: %w", path, err)
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("decode store %q: %w", path, err)
	}
	if snap.Version != 0 && snap.Version != storeVersion {
		return nil, fmt.Errorf("store %q has version %d, want %d", path, snap.Version, storeVersion)
	}
	s.restore(&snap)
	return s, nil
}

// Path reports where the store persists ("", for a memory-only store).
func (s *Store) Path() string { return s.path }

// restore rebuilds the maps and indexes from a decoded snapshot.
func (s *Store) restore(snap *snapshot) {
	s.seq = snap.Seq
	for _, row := range snap.Users {
		u := row.toEntity()
		s.users[u.ID] = u
		s.byEmail[u.Email] = u.ID
	}
	for _, o := range snap.Orgs {
		s.orgs[o.ID] = o
		s.bySlug[o.Slug] = o.ID
	}
	for _, m := range snap.Memberships {
		s.members[m.OrgID] = append(s.members[m.OrgID], m)
		s.byMember[m.ID] = m
	}
	for _, inv := range snap.Invitations {
		s.invites[inv.ID] = inv
		s.byCode[inv.Code] = inv.ID
	}
	for _, l := range snap.Ledger {
		s.ledger[l.ID] = l
		key := l.OrgID + "/" + l.Period
		s.usageBy[key] = append(s.usageBy[key], l)
	}
	// AppendUsageEvent prepends, so each per-org slice is already newest-first
	// and concatenating preserves that order.
	for _, ev := range snap.Events {
		s.events[ev.OrgID] = append(s.events[ev.OrgID], ev)
	}
}

// snapshotLocked builds the durable view. Callers must hold s.mu.
func (s *Store) snapshotLocked() *snapshot {
	snap := &snapshot{
		Version: storeVersion,
		SavedAt: time.Now().UTC(),
		Seq:     s.seq,
		Users:   make([]userRow, 0, len(s.users)),
		Orgs:    make([]*entity.Organization, 0, len(s.orgs)),
	}
	for _, u := range s.users {
		snap.Users = append(snap.Users, userToRow(u))
	}
	for _, o := range s.orgs {
		snap.Orgs = append(snap.Orgs, o)
	}
	for _, list := range s.members {
		snap.Memberships = append(snap.Memberships, list...)
	}
	for _, inv := range s.invites {
		snap.Invitations = append(snap.Invitations, inv)
	}
	for _, l := range s.ledger {
		snap.Ledger = append(snap.Ledger, l)
	}
	for _, list := range s.events {
		snap.Events = append(snap.Events, list...)
	}

	// Map iteration is randomised; sort for a stable, diffable file.
	sort.Slice(snap.Users, func(i, j int) bool { return snap.Users[i].ID < snap.Users[j].ID })
	sort.Slice(snap.Orgs, func(i, j int) bool { return snap.Orgs[i].ID < snap.Orgs[j].ID })
	sort.Slice(snap.Memberships, func(i, j int) bool { return snap.Memberships[i].ID < snap.Memberships[j].ID })
	sort.Slice(snap.Invitations, func(i, j int) bool { return snap.Invitations[i].ID < snap.Invitations[j].ID })
	sort.Slice(snap.Ledger, func(i, j int) bool { return snap.Ledger[i].ID < snap.Ledger[j].ID })
	sort.Slice(snap.Events, func(i, j int) bool { return snap.Events[i].ID < snap.Events[j].ID })
	return snap
}

// saveLocked persists the store. Callers must hold s.mu. A memory-only store
// (path == "") is a no-op.
func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.snapshotLocked(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	return writeFileAtomic(s.path, data)
}

// writeFileAtomic replaces path with data without ever exposing a partial file.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create store dir: %w", err)
		}
	}
	tmp, err := os.CreateTemp(dir, ".saas-store-*.tmp")
	if err != nil {
		return fmt.Errorf("create store temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write store temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync store temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close store temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace store file: %w", err)
	}
	return nil
}
