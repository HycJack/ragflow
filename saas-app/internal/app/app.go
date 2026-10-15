// Package app wires the services together and exposes the HTTP handlers as a
// single http.Handler, using only the Go standard library.
package app

import (
	"saas/internal/dao"
	"saas/internal/service"
)

// App holds the service graph.
type App struct {
	Store         *dao.Store
	Plans         *service.PlanService
	Auth          *service.AuthService
	Organizations *service.OrganizationService
	Usage         *service.UsageService
}

// New builds and wires the application.
func New() *App {
	store := dao.NewStore()
	plans := service.NewPlanService()
	orgSvc := service.NewOrganizationService(store, plans)
	return &App{
		Store:         store,
		Plans:         plans,
		Auth:          service.NewAuthService(store, plans),
		Organizations: orgSvc,
		Usage:         service.NewUsageService(store, orgSvc),
	}
}