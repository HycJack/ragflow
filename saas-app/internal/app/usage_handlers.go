package app

import (
	"net/http"
)

// ---- Usage ----

func (a *App) handleCurrentUsage(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	snapshots, err := a.Usage.CurrentUsage(u.ID, r.PathValue("org_id"))
	returnErr(w, err, snapshots)
}

func (a *App) handleListUsageEvents(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	meter := r.URL.Query().Get("meter")
	offset := parseInt(r.URL.Query().Get("offset"), 0)
	limit := parseInt(r.URL.Query().Get("limit"), 20)
	events, err := a.Usage.ListEvents(u.ID, r.PathValue("org_id"), meter, offset, limit)
	returnErr(w, err, events)
}

// handleConsume is a demonstration endpoint used by the front-end to exercise
// quota enforcement (e.g. simulate a parse). In production the ingestion path
// would call Usage.Consume before/after work.
func (a *App) handleConsume(w http.ResponseWriter, r *http.Request) {
	u := a.requireUser(w, r)
	if u == nil {
		return
	}
	var req struct {
		Meter  string `json:"meter"`
		Amount int64  `json:"amount"`
	}
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Meter == "" || req.Amount <= 0 {
		fail(w, http.StatusBadRequest, http.StatusBadRequest, "meter and positive amount required")
		return
	}
	// Record an audit event regardless.
	if err := a.Usage.RecordEvent(u.ID, r.PathValue("org_id"), req.Meter, "demo", "", req.Amount); err != nil {
		returnErr(w, err, false)
		return
	}
	err := a.Usage.Consume(r.PathValue("org_id"), req.Meter, req.Amount)
	if err != nil {
		returnErr(w, err, false)
		return
	}
	ok(w, true, "consumed")
}