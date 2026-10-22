package app

import "net/http"

// Routes returns the app's http.Handler (Go 1.22+ pattern routing).
func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	// Auth (public)
	mux.HandleFunc("POST /api/v1/auth/register", a.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", a.handleLogout)

	// Current user (authenticated) + profile incl. platform-admin flag.
	mux.HandleFunc("GET /api/v1/me", a.handleMe)

	// Plans (public catalog)
	mux.HandleFunc("GET /api/v1/plans", a.handleListPlans)

	// Organizations
	mux.HandleFunc("GET /api/v1/orgs", a.handleListOrgs)
	mux.HandleFunc("POST /api/v1/orgs", a.handleCreateOrg)
	mux.HandleFunc("GET /api/v1/orgs/{org_id}", a.handleGetOrg)

	// Invitation accept
	mux.HandleFunc("POST /api/v1/invitations/accept", a.handleAcceptInvitation)

	// Members
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/members", a.handleListMembers)
	mux.HandleFunc("PUT /api/v1/orgs/{org_id}/members/{user_id}/role", a.handleUpdateMemberRole)
	mux.HandleFunc("DELETE /api/v1/orgs/{org_id}/members/{user_id}", a.handleRemoveMember)

	// Invitations
	mux.HandleFunc("POST /api/v1/orgs/{org_id}/invitations", a.handleCreateInvitation)
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/invitations", a.handleListInvitations)

	// Usage
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/usage/current", a.handleCurrentUsage)
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/usage/events", a.handleListUsageEvents)
	mux.HandleFunc("POST /api/v1/orgs/{org_id}/usage/consume", a.handleConsume)

	// Platform admin console (is_platform_admin only)
	mux.HandleFunc("GET /api/v1/admin/orgs", a.handleAdminListOrgs)
	mux.HandleFunc("GET /api/v1/admin/users", a.handleAdminListUsers)
	mux.HandleFunc("PATCH /api/v1/admin/users/{user_id}/status", a.handleAdminSetUserStatus)
	mux.HandleFunc("PATCH /api/v1/admin/orgs/{org_id}/plan", a.handleAdminSetOrgPlan)
	mux.HandleFunc("DELETE /api/v1/admin/orgs/{org_id}", a.handleAdminDeleteOrg)

	return withMiddleware(mux)
}

// withMiddleware applies CORS + a simple request logger.
func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Workspace-ID")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}