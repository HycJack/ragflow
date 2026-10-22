package app

import (
	"net/http"
)

// ---- Platform administrator console ----

// handleAdminListUsers lists every platform account (admin only).
func (a *App) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	users, err := a.Admin.ListUsers(u)
	returnErr(w, err, users)
}

// handleAdminListOrgs lists every organization (admin only).
func (a *App) handleAdminListOrgs(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	orgs, err := a.Admin.ListOrgs(u)
	returnErr(w, err, orgs)
}

// handleAdminSetUserStatus enables or disables an account.
func (a *App) handleAdminSetUserStatus(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	err := a.Admin.SetUserStatus(u, r.PathValue("user_id"), req.Status)
	returnErr(w, err, true)
}

// handleAdminSetOrgPlan moves an organization onto another plan.
func (a *App) handleAdminSetOrgPlan(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	var req struct {
		PlanCode string `json:"plan_code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	err := a.Admin.SetOrgPlan(u, r.PathValue("org_id"), req.PlanCode)
	returnErr(w, err, true)
}

// handleAdminDeleteOrg removes an organization and everything it owns.
func (a *App) handleAdminDeleteOrg(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	err := a.Admin.DeleteOrg(u, r.PathValue("org_id"))
	returnErr(w, err, true)
}