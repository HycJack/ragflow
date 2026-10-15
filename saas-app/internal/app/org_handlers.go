package app

import (
	"net/http"

	"saas/internal/service"
)

// ---- Organizations ----

func (a *App) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	orgs, err := a.Organizations.ListByUser(u.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}
	ok(w, orgs, "success")
}

func (a *App) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	org, err := a.Organizations.ViewFor(u.ID, r.PathValue("org_id"))
	returnErr(w, err, org)
}

func (a *App) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	var req service.CreateOrgRequest
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	org, err := a.Organizations.CreateOrg(u, &req)
	returnErr(w, err, org)
}

// ---- Members ----

func (a *App) handleListMembers(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	members, err := a.Organizations.ListMembers(u.ID, r.PathValue("org_id"))
	returnErr(w, err, members)
}

func (a *App) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	err := a.Organizations.UpdateMemberRole(u.ID, r.PathValue("org_id"), r.PathValue("user_id"), req.Role)
	returnErr(w, err, true)
}

func (a *App) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	err := a.Organizations.RemoveMember(u.ID, r.PathValue("org_id"), r.PathValue("user_id"))
	returnErr(w, err, true)
}

// ---- Invitations ----

func (a *App) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	var req service.CreateInvitationRequest
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	inv, err := a.Organizations.CreateInvitation(u.ID, r.PathValue("org_id"), &req)
	returnErr(w, err, inv)
}

func (a *App) handleListInvitations(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	invs, err := a.Organizations.ListInvitations(u.ID, r.PathValue("org_id"))
	returnErr(w, err, invs)
}

func (a *App) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	org, err := a.Organizations.AcceptInvitation(u, req.Code)
	returnErr(w, err, org)
}

// ---- Plans ----

func (a *App) handleListPlans(w http.ResponseWriter, r *http.Request) {
	_ = a.requireUser(w, r) // authenticated, but plans are not org-scoped here
	ok(w, a.plansList(), "success")
}

func (a *App) plansList() []map[string]any {
	plans := a.Plans.List()
	out := make([]map[string]any, 0, len(plans))
	for _, p := range plans {
		out = append(out, map[string]any{
			"code": p.Code, "name": p.Name, "price_monthly_cents": p.PriceMonthlyCents,
			"storage_bytes": p.StorageBytes, "token_to_parse": p.TokenToParse,
			"seats": p.Seats, "llm_calls": p.LLMCalls,
		})
	}
	return out
}

func returnErr(w http.ResponseWriter, err error, data any) {
	if err != nil {
		fail(w, errStatus(err), httpStatusOf(err), err.Error())
		return
	}
	ok(w, data, "success")
}