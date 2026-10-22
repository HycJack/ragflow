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