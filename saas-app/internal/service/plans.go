package service

import (
	"saas/internal/entity"
)

// PlanService provides the catalog of sellable plans. Plans are seeded in
// memory; extend to a persisted BillingPlan store when billing is added.
type PlanService struct {
	plans map[string]entity.Plan
}

// NewPlanService returns the default plan catalog.
func NewPlanService() *PlanService {
	return &PlanService{plans: map[string]entity.Plan{
		"free": {
			Code: "free", Name: "Free", PriceMonthlyCents: 0,
			StorageBytes: 5 << 30,     // 5 GB
			TokenToParse: 2_500_000,   // 2.5 M
			Seats:        10,
			LLMCalls:     -1,          // unlimited
		},
		"pro": {
			Code: "pro", Name: "Pro", PriceMonthlyCents: 4_900,
			StorageBytes: 50 << 30,      // 50 GB
			TokenToParse: 25_000_000,    // 25 M
			Seats:        20,
			LLMCalls:     100_000,
		},
		"enterprise": {
			Code: "enterprise", Name: "Enterprise", PriceMonthlyCents: 0,
			StorageBytes: -1, TokenToParse: -1, Seats: -1, LLMCalls: -1,
		},
	}}
}

// Plan returns a plan by code (defaults to free).
func (s *PlanService) Plan(code string) entity.Plan {
	if p, ok := s.plans[code]; ok {
		return p
	}
	return s.plans["free"]
}

// List returns all plans in a stable order.
func (s *PlanService) List() []entity.Plan {
	order := []string{"free", "pro", "enterprise"}
	out := make([]entity.Plan, 0, len(order))
	for _, code := range order {
		out = append(out, s.plans[code])
	}
	return out
}