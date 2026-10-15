package service

import (
	"errors"
	"time"

	"saas/internal/dao"
	"saas/internal/entity"
)

var (
	// ErrQuotaExceeded is returned when a meter limit would be exceeded.
	ErrQuotaExceeded = errors.New("usage limit exceeded")
)

// CurrentPeriod returns the "YYYY-MM" billing period for now (UTC).
func CurrentPeriod() string {
	return time.Now().UTC().Format("2006-01")
}

// UsageMeterSnapshot is the API shape of a meter with its plan limit.
type UsageMeterSnapshot struct {
	Meter string `json:"meter"`
	Used  int64  `json:"used"`
	Limit int64  `json:"limit"` // -1 = unlimited
}

// UsageEventView is the API shape of a usage/audit record.
type UsageEventView struct {
	ID         string `json:"id"`
	Meter      string `json:"meter"`
	Amount     int64  `json:"amount"`
	RefKind    string `json:"ref_kind,omitempty"`
	RefID      string `json:"ref_id,omitempty"`
	Billable   bool   `json:"billable"`
	OccurredAt string `json:"occurred_at"`
}

// UsageService provides quota consumption and metering on top of the store.
type UsageService struct {
	store      *dao.Store
	orgService *OrganizationService
}

// NewUsageService creates a UsageService.
func NewUsageService(store *dao.Store, orgService *OrganizationService) *UsageService {
	return &UsageService{store: store, orgService: orgService}
}

// CurrentUsage returns the current period consumption with plan limits.
func (s *UsageService) CurrentUsage(userID, orgID string) ([]*UsageMeterSnapshot, error) {
	if _, err := s.orgService.RequireMembership(userID, orgID); err != nil {
		return nil, err
	}
	limits, err := s.orgService.LimitsFor(orgID)
	if err != nil {
		return nil, err
	}
	ledger := s.store.GetLedger(orgID, CurrentPeriod())
	used := map[string]int64{}
	for _, l := range ledger {
		used[l.Meter] = l.Used
	}
	seats := s.store.CountMembersByOrg(orgID)
	return []*UsageMeterSnapshot{
		{Meter: entity.MeterStorageBytes, Used: used[entity.MeterStorageBytes], Limit: limits.StorageBytes},
		{Meter: entity.MeterTokenToParse, Used: used[entity.MeterTokenToParse], Limit: limits.TokenToParse},
		{Meter: entity.MeterLLMCalls, Used: used[entity.MeterLLMCalls], Limit: limits.LLMCalls},
		{Meter: entity.MeterSeats, Used: seats, Limit: limits.Seats},
	}, nil
}

// Consume attempts to increment a meter for an org; returns ErrQuotaExceeded
// when the plan limit would be crossed.
func (s *UsageService) Consume(orgID, meter string, amount int64) error {
	limits, err := s.orgService.LimitsFor(orgID)
	if err != nil {
		return err
	}
	var limit int64
	switch meter {
	case entity.MeterStorageBytes:
		limit = limits.StorageBytes
	case entity.MeterTokenToParse:
		limit = limits.TokenToParse
	case entity.MeterLLMCalls:
		limit = limits.LLMCalls
	case entity.MeterSeats:
		limit = limits.Seats
	default:
		limit = -1
	}
	ok, err := s.store.TryConsumeLedger(orgID, CurrentPeriod(), meter, amount, limit)
	if err != nil {
		return err
	}
	if !ok {
		return ErrQuotaExceeded
	}
	return nil
}

// RecordEvent appends a usage/audit event.
func (s *UsageService) RecordEvent(userID, orgID, meter, refKind, refID string, amount int64) {
	ev := &entity.UsageEvent{
		OrgID: orgID, UserID: userID, Meter: meter, Amount: amount,
		RefKind: refKind, RefID: refID,
		Billable:   isBillable(meter),
		OccurredAt: time.Now(),
	}
	s.store.AppendUsageEvent(orgID, ev)
}

// ListEvents lists usage events for an org (paginated, optional meter).
func (s *UsageService) ListEvents(userID, orgID, meter string, offset, limit int) ([]*UsageEventView, error) {
	if _, err := s.orgService.RequireMembership(userID, orgID); err != nil {
		return nil, err
	}
	events := s.store.ListUsageEvents(orgID, meter, offset, limit)
	out := make([]*UsageEventView, 0, len(events))
	for _, ev := range events {
		out = append(out, &UsageEventView{ID: ev.ID, Meter: ev.Meter, Amount: ev.Amount,
			RefKind: ev.RefKind, RefID: ev.RefID, Billable: ev.Billable,
			OccurredAt: ev.OccurredAt.Format(time.RFC3339)})
	}
	return out, nil
}

func isBillable(meter string) bool {
	return meter == entity.MeterStorageBytes || meter == entity.MeterTokenToParse ||
		meter == entity.MeterLLMCalls
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}