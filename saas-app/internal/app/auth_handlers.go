package app

import (
	"net/http"

	"saas/internal/entity"
	"saas/internal/service"
)

// ---- Auth / me ----

func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	token, user, err := a.Auth.Register(&req)
	if err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	// If an invite code is present, join the target org.
	myOrgs := []*service.OrganizationView{}
	if req.InviteCode != "" {
		if ov, ierr := a.Organizations.AcceptInvitation(user, req.InviteCode); ierr == nil {
			myOrgs = append(myOrgs, ov)
		}
	}
	writeJSON(w, http.StatusOK, response{Code: codeSuccess, Message: "welcome aboard!",
		Data: map[string]any{"token": token, "profile": profileOf(user), "organizations": myOrgs}})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	token, user, err := a.Auth.LoginByEmail(req.Email, req.Password)
	if err != nil {
		fail(w, http.StatusUnauthorized, http.StatusUnauthorized, err.Error())
		return
	}
	ok(w, map[string]any{"token": token, "profile": profileOf(user)}, "welcome back!")
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.Auth.Logout(authToken(r))
	ok(w, true, "success")
}

// handleMe returns the current user's profile, including platform-admin status.
func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	ok(w, profileOf(u), "success")
}

// requireUser resolves the current user or writes an error and returns nil.
func (a *App) requireUser(w http.ResponseWriter, r *http.Request) *entity.User {
	u, err := a.Auth.GetUserByToken(authToken(r))
	if err != nil {
		fail(w, http.StatusUnauthorized, http.StatusUnauthorized, "invalid or missing token")
		return nil
	}
	if u.Status == "0" {
		fail(w, http.StatusForbidden, http.StatusForbidden, "this account has been disabled")
		return nil
	}
	return u
}

func profileOf(u *entity.User) map[string]any {
	return map[string]any{
		"id": u.ID, "email": u.Email, "nickname": u.Nickname,
		"avatar": u.Avatar, "status": u.Status, "created_at": u.CreatedAt,
		"is_platform_admin": u.IsPlatformAdmin,
	}
}