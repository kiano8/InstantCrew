package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"instantcrew/internal/auth"
	"instantcrew/internal/httpx"
	"instantcrew/internal/store"
)

type Handler struct {
	Store            *store.Store
	MatchFeeCentavos int64         // e.g. 10000 = ₱100.00
	SearchWindow     time.Duration // e.g. 3 * time.Minute
}

func (h *Handler) Register(mux *http.ServeMux, a *auth.Handler) {
	emp := func(f http.HandlerFunc) http.Handler { return a.RequireRole("employer", f) }
	app := func(f http.HandlerFunc) http.Handler { return a.RequireRole("applicant", f) }
	adm := func(f http.HandlerFunc) http.Handler { return a.RequireRole("admin", f) }

	mux.HandleFunc("GET /api/catalog", h.Catalog) // public: the applicant side needs it too

	mux.Handle("POST /api/jobs", emp(h.Create))
	mux.Handle("GET /api/jobs", emp(h.List))
	mux.Handle("GET /api/jobs/{id}", emp(h.Get))
	mux.Handle("GET /api/jobs/{id}/matches", emp(h.Matches))
	mux.Handle("PUT /api/jobs/{id}", emp(h.Update))
	mux.Handle("POST /api/jobs/{id}/pay", emp(h.Pay))
	mux.Handle("POST /api/jobs/{id}/search", emp(h.Search))
	mux.Handle("POST /api/jobs/{id}/confirm", emp(h.Confirm))
	mux.Handle("POST /api/jobs/{id}/end", emp(h.End))
	mux.Handle("POST /api/jobs/{id}/cancel", emp(h.Cancel))
	mux.Handle("POST /api/jobs/{id}/decline-match", emp(h.DeclineMatch))
	mux.Handle("POST /api/jobs/{id}/next-attempt", emp(h.NextAttempt))

	mux.Handle("GET /api/applicant/profile", app(h.ApplicantProfile))
	mux.Handle("GET /api/applicant/rating", app(h.ApplicantRating))
	mux.Handle("GET /api/applicant/hiring-pass", app(h.ApplicantHiringPass))
	mux.Handle("POST /api/applicant/hiring-pass/pay", app(h.PayApplicantHiringPass))
	mux.Handle("PUT /api/applicant/profile", app(h.SaveApplicantProfile))
	mux.Handle("GET /api/applicant/jobs", app(h.ApplicantJobs))
	mux.Handle("GET /api/applicant/jobs/{id}", app(h.ApplicantJob))
	mux.Handle("POST /api/applicant/jobs/{id}/accept", app(h.ApplicantAccept))
	mux.Handle("POST /api/applicant/jobs/{id}/reject", app(h.ApplicantReject))
	mux.Handle("GET /api/applicant/assignments", app(h.ApplicantAssignments))
	mux.Handle("GET /api/applicant/availability", app(h.ApplicantAvailability))
	mux.Handle("PUT /api/applicant/availability", app(h.SaveApplicantAvailability))
	mux.Handle("POST /api/applicant/assignments/{id}/transit", app(h.ApplicantTransit))
	mux.Handle("POST /api/applicant/assignments/{id}/end", app(h.ApplicantEndAssignment))
	mux.Handle("GET /api/notifications", a.RequireAuth(http.HandlerFunc(h.Notifications)))
	mux.Handle("POST /api/notifications/read", a.RequireAuth(http.HandlerFunc(h.ReadNotifications)))
	mux.Handle("DELETE /api/notifications", a.RequireAuth(http.HandlerFunc(h.ClearNotifications)))

	mux.Handle("POST /api/upload-proof", a.RequireAuth(http.HandlerFunc(h.UploadProof)))
	mux.Handle("GET /api/employer/payments", emp(h.EmployerPaymentHistory))
	mux.Handle("GET /api/applicant/payments", app(h.ApplicantPaymentHistory))
	mux.Handle("GET /api/admin/payments", adm(h.AdminPayments))
	mux.Handle("POST /api/admin/payments/verify", adm(h.AdminVerifyPayment))
}

func (h *Handler) assignmentID(w http.ResponseWriter, r *http.Request) (*store.User, int64, bool) {
	u, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(w, http.StatusNotFound, "assignment not found")
		return nil, 0, false
	}
	return u, id, true
}

func (h *Handler) ApplicantTransit(w http.ResponseWriter, r *http.Request) {
	u, id, ok := h.assignmentID(w, r)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	if req.Status != "en_route" && req.Status != "arrived" {
		httpx.Error(w, http.StatusBadRequest, "status must be en_route or arrived")
		return
	}
	verb, title, kind := "is on the way", "Crew is on the way", "transit"
	if req.Status == "arrived" {
		verb, title, kind = "has arrived", "Crew has arrived", "arrived"
	}
	if err := h.Store.UpdateApplicantTransit(r.Context(), id, u.ID, req.Status, title, u.Name+" "+verb+".", kind); err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"updated": true})
}

func (h *Handler) ApplicantEndAssignment(w http.ResponseWriter, r *http.Request) {
	u, id, ok := h.assignmentID(w, r)
	if !ok {
		return
	}
	if err := h.Store.EndApplicantAssignment(r.Context(), id, u.ID); err != nil {
		h.fail(w, err)
		return
	}
	if err := h.Store.NotifyJobEmployer(r.Context(), id, "Contract ended", u.Name+" ended the contract.", "end"); err != nil {
		log.Printf("notification: %v", err)
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ended": true})
}

func (h *Handler) Notifications(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	list, err := h.Store.Notifications(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"notifications": list})
}
func (h *Handler) ReadNotifications(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if err := h.Store.MarkNotificationsRead(r.Context(), u.ID); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) ClearNotifications(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if err := h.Store.ClearNotifications(r.Context(), u.ID); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ApplicantAvailability(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	items, err := h.Store.ApplicantAvailability(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"availability": items})
}

func (h *Handler) SaveApplicantAvailability(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var req struct {
		Availability []store.Availability `json:"availability"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	for _, a := range req.Availability {
		if a.StartsAt <= 0 || a.EndsAt <= a.StartsAt || (a.Status != "available" && a.Status != "unavailable") {
			httpx.Error(w, http.StatusBadRequest, "invalid availability range")
			return
		}
	}
	if _, err := h.Store.ApplicantProfile(r.Context(), u.ID); errors.Is(err, store.ErrNotFound) {
		httpx.Error(w, http.StatusConflict, "create your applicant profile first")
		return
	} else if err != nil {
		h.fail(w, err)
		return
	}
	if err := h.Store.SaveApplicantAvailability(r.Context(), u.ID, req.Availability); err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"availability": req.Availability})
}

func (h *Handler) ApplicantProfile(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	p, err := h.Store.ApplicantProfile(r.Context(), u.ID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.JSON(w, http.StatusOK, map[string]any{"profile": nil})
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"profile": p})
}

func (h *Handler) ApplicantRating(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	p, err := h.Store.ApplicantProfile(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]float64{"rating": p.Rating})
}

func (h *Handler) ApplicantHiringPass(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	p, err := h.Store.ApplicantHiringPass(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"hiring_pass": p})
}

func (h *Handler) PayApplicantHiringPass(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var req struct {
		Method     string `json:"method"`
		ProofImage string `json:"proof_image"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method != "gcash" && method != "maya" && method != "card" {
		httpx.Error(w, http.StatusBadRequest, "method must be gcash, maya, or card")
		return
	}
	proofImage := strings.TrimSpace(req.ProofImage)
	if (method == "gcash" || method == "maya") && proofImage == "" {
		httpx.Error(w, http.StatusBadRequest, "proof of payment is required for GCash and Maya")
		return
	}
	p, err := h.Store.AddApplicantHiringPass(r.Context(), u.ID, method, proofImage)
	if err != nil {
		h.fail(w, err)
		return
	}
	pending := p.PendingPayment != nil && p.PendingPayment.Status == "pending"
	rejected := p.PendingPayment != nil && p.PendingPayment.Status == "rejected"
	httpx.JSON(w, http.StatusOK, map[string]any{
		"hiring_pass": p,
		"paid":        !pending && !rejected,
		"pending":     pending,
		"rejected":    rejected,
		"amount":      10,
	})
}

func (h *Handler) SaveApplicantProfile(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var p store.ApplicantProfile
	if !httpx.Decode(w, r, &p) {
		return
	}
	if (p.EmploymentType != "full-time" && p.EmploymentType != "part-time") || p.ExpectedRate < 1 || p.City == "" || (p.RatePeriod != "hour" && p.RatePeriod != "day" && p.RatePeriod != "week" && p.RatePeriod != "month") {
		httpx.Error(w, http.StatusBadRequest, "invalid applicant profile")
		return
	}
	if p.Availability == "" {
		p.Availability = "now"
	}
	if p.Active == nil {
		active := true
		p.Active = &active
	}
	if p.Availability != "now" && p.Availability != "later" {
		httpx.Error(w, http.StatusBadRequest, "invalid availability mode")
		return
	}
	for i, role := range p.Roles {
		if _, ok := RoleByID(role); !ok {
			matched := ""
			for _, category := range Catalog {
				for _, item := range category.Roles {
					if strings.EqualFold(item.Label, role) {
						matched = item.ID
					}
				}
			}
			if matched != "" {
				p.Roles[i] = matched
				continue
			}
			httpx.Error(w, http.StatusBadRequest, "unknown role")
			return
		}
	}
	// Some existing client sessions contain duplicate selected roles. Collapse
	// them after label-to-ID normalization so a profile update cannot violate
	// applicant_roles' (applicant_id, role_id) primary key.
	seenRoles := make(map[string]struct{}, len(p.Roles))
	uniqueRoles := p.Roles[:0]
	for _, role := range p.Roles {
		if _, exists := seenRoles[role]; exists {
			continue
		}
		seenRoles[role] = struct{}{}
		uniqueRoles = append(uniqueRoles, role)
	}
	p.Roles = uniqueRoles
	if err := h.Store.SaveApplicantProfile(r.Context(), u.ID, p); err != nil {
		h.fail(w, err)
		return
	}
	savedProfile, err := h.Store.ApplicantProfile(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"profile": savedProfile})
}

func (h *Handler) ApplicantJobs(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	jobs, err := h.Store.ApplicantJobs(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := h.Store.RecordApplicantOfferViews(r.Context(), u.ID, jobs); err != nil {
		h.fail(w, err)
		return
	}
	out := make([]jobDTO, 0, len(jobs))
	for _, j := range jobs {
		d := h.toDTO(j)
		crew, crewErr := h.Store.Assignments(r.Context(), j.ID)
		if crewErr != nil {
			h.fail(w, crewErr)
			return
		}
		for _, c := range crew {
			d.Crew = append(d.Crew, crewDTO{c.ApplicantID, c.Name, c.AcceptedAt, c.EndedAt, c.Transit})
		}
		out = append(out, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"jobs": out})
}

func (h *Handler) ApplicantJob(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(w, http.StatusNotFound, "job not found")
		return
	}
	j, err := h.Store.ApplicantJob(r.Context(), u.ID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"job": h.toDTO(j)})
}

func (h *Handler) ApplicantAccept(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(w, http.StatusNotFound, "job not found")
		return
	}
	if _, err = h.Store.ApplicantProfile(r.Context(), u.ID); errors.Is(err, store.ErrNotFound) {
		httpx.Error(w, http.StatusConflict, "complete your applicant profile first")
		return
	} else if err != nil {
		h.fail(w, err)
		return
	}
	if err = h.Store.AcceptJob(r.Context(), id, u.ID); err != nil {
		h.fail(w, err)
		return
	}
	if err := h.Store.NotifyJobEmployer(r.Context(), id, "Shift accepted", u.Name+" accepted the shift offer.", "accept"); err != nil {
		log.Printf("notification: %v", err)
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"accepted": true})
}

func (h *Handler) ApplicantReject(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(w, http.StatusNotFound, "job not found")
		return
	}
	if err = h.Store.RejectJob(r.Context(), id, u.ID); err != nil {
		h.fail(w, err)
		return
	}
	if err = h.Store.AdjustApplicantRating(r.Context(), id, u.ID, "declined"); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ApplicantAssignments(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	jobs, err := h.Store.ApplicantAssignments(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		assignment, err := h.Store.ApplicantAssignmentState(r.Context(), u.ID, j.ID)
		if err != nil {
			h.fail(w, err)
			return
		}
		out = append(out, map[string]any{"job": h.toDTO(j), "accepted_at": assignment.AcceptedAt, "ended_at": assignment.EndedAt, "transit_status": assignment.Transit, "confirmed": j.Status == "active" || j.Status == "completed"})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"assignments": out})
}

// ── DTOs ──

type crewDTO struct {
	ApplicantID int64      `json:"applicant_id"`
	Name        string     `json:"name"`
	AcceptedAt  time.Time  `json:"accepted_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	Transit     string     `json:"transit_status"`
}

type jobDTO struct {
	ID               int64             `json:"id"`
	Category         string            `json:"category"`
	Role             string            `json:"role"`
	RoleLabel        string            `json:"role_label"`
	EmploymentType   string            `json:"employment_type"`
	Rate             float64           `json:"rate"`
	RatePeriod       string            `json:"rate_period"`
	City             string            `json:"city"`
	MapLocation      string            `json:"map_location"`
	MapsURL          string            `json:"maps_url"`
	Lat              *float64          `json:"lat,omitempty"`
	Lng              *float64          `json:"lng,omitempty"`
	Timing           string            `json:"timing"`
	ScheduledAt      *time.Time        `json:"scheduled_at,omitempty"`
	CrewNeeded       int               `json:"crew_needed"`
	AcceptedCount    int               `json:"accepted_count"`
	Status           string            `json:"status"`
	SearchAttempts   int               `json:"search_attempts"`
	SearchExpiresAt  *time.Time        `json:"search_expires_at,omitempty"`
	SecondsRemaining int64             `json:"seconds_remaining"` // server clock, so the countdown can't be cheated
	CurrentTry       int               `json:"current_try"`
	MaxTries         int               `json:"max_tries"`
	CurrentAttempt   int               `json:"current_attempt"`
	MaxAttempts      int               `json:"max_attempts"`
	TriesRemaining   int               `json:"tries_remaining"`
	MatchFee         float64           `json:"match_fee"`
	Crew             []crewDTO         `json:"crew,omitempty"`
	Payment          *store.JobPayment `json:"payment,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

func (h *Handler) toDTO(j *store.Job) jobDTO {
	curTry := j.CurrentTry
	if curTry < 1 {
		curTry = 1
	}
	maxTries := j.MaxTries
	if maxTries < 1 {
		maxTries = 3
	}
	curAtt := j.CurrentAttempt
	if curAtt < 1 {
		curAtt = 1
	}
	maxAtt := j.MaxAttempts
	if maxAtt < 1 {
		maxAtt = 3
	}
	triesRem := maxTries - curTry
	if triesRem < 0 {
		triesRem = 0
	}

	d := jobDTO{
		ID: j.ID, Category: j.Category, Role: j.RoleID, RoleLabel: j.RoleLabel,
		EmploymentType: j.EmploymentType, Rate: float64(j.RateCentavos) / 100, RatePeriod: j.RatePeriod,
		City: j.City, MapLocation: j.MapLocation, MapsURL: j.MapsURL, Lat: j.Lat, Lng: j.Lng,
		Timing: j.Timing, ScheduledAt: j.ScheduledAt, CrewNeeded: j.CrewNeeded,
		AcceptedCount: j.AcceptedCount, Status: j.Status, SearchAttempts: j.SearchAttempts,
		SearchExpiresAt: j.SearchExpiresAt,
		CurrentTry:      curTry,
		MaxTries:        maxTries,
		CurrentAttempt:  curAtt,
		MaxAttempts:     maxAtt,
		TriesRemaining:  triesRem,
		MatchFee:        float64(h.MatchFeeCentavos) / 100,
		CreatedAt:       j.CreatedAt,
	}
	if j.Status == "matching" && j.SearchExpiresAt != nil {
		if rem := time.Until(*j.SearchExpiresAt); rem > 0 {
			d.SecondsRemaining = int64(rem.Seconds())
		}
	}
	return d
}

// ── helpers ──

func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "job not found")
	case errors.Is(err, store.ErrBadState):
		httpx.Error(w, http.StatusConflict, "this action isn't allowed in the job's current status")
	case errors.Is(err, store.ErrNotConfirmed):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrAlreadyAccepted):
		httpx.Error(w, http.StatusConflict, "already accepted")
	case errors.Is(err, store.ErrHiringPassPending):
		httpx.Error(w, http.StatusPaymentRequired, err.Error())
	case errors.Is(err, store.ErrHiringPassEmpty):
		httpx.Error(w, http.StatusPaymentRequired, err.Error())
	case errors.Is(err, store.ErrPaymentNotVerified), errors.Is(err, store.ErrPaymentRejected):
		httpx.Error(w, http.StatusPaymentRequired, err.Error())
	case errors.Is(err, store.ErrPaymentPendingExists):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		log.Printf("jobs: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

// ids returns the logged-in user, the {id} path value, and the effective employer ID.
// For admin users acting on an employer's job, empID is adopted from the job's creator.
func (h *Handler) ids(w http.ResponseWriter, r *http.Request) (*store.User, int64, int64, bool) {
	u, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Error(w, http.StatusNotFound, "job not found")
		return nil, 0, 0, false
	}
	empID := u.ID
	if u.Role == "admin" {
		if ownerID, err := h.Store.JobOwnerID(r.Context(), id); err == nil && ownerID > 0 {
			empID = ownerID
		}
	}
	return u, id, empID, true
}

// writeJob reloads the job (with crew) and sends it. Expired searches are flipped first.
func (h *Handler) writeJob(w http.ResponseWriter, r *http.Request, status int, employerID, id int64) {
	ctx := r.Context()
	_, _ = h.Store.ExpireSearches(ctx)
	j, err := h.Store.GetJob(ctx, employerID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	d := h.toDTO(j)
	crew, err := h.Store.Assignments(ctx, j.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	for _, c := range crew {
		d.Crew = append(d.Crew, crewDTO{c.ApplicantID, c.Name, c.AcceptedAt, c.EndedAt, c.Transit})
	}
	if pay, err := h.Store.LatestJobPayment(ctx, j.ID); err != nil {
		h.fail(w, err)
		return
	} else if pay != nil {
		d.Payment = pay
	}
	httpx.JSON(w, status, map[string]any{"job": d})
}

// ── request parsing / validation ──

type jobRequest struct {
	Category       string   `json:"category"`
	Role           string   `json:"role"`
	EmploymentType string   `json:"employment_type"`
	Rate           float64  `json:"rate"`
	RatePeriod     string   `json:"rate_period"`
	City           string   `json:"city"`
	MapLocation    string   `json:"map_location"`
	Lat            *float64 `json:"lat"`
	Lng            *float64 `json:"lng"`
	Timing         string   `json:"timing"`
	ScheduledAt    string   `json:"scheduled_at"`
	CrewNeeded     int      `json:"crew_needed"`
}

func (h *Handler) parse(w http.ResponseWriter, r *http.Request) (*store.Job, bool) {
	var req jobRequest
	if !httpx.Decode(w, r, &req) {
		return nil, false
	}
	bad := func(msg string) (*store.Job, bool) {
		httpx.Error(w, http.StatusBadRequest, msg)
		return nil, false
	}

	role, ok := LookupRole(req.Category, req.Role)
	if !ok {
		return bad("unknown category or role")
	}
	if req.EmploymentType != "full-time" && req.EmploymentType != "part-time" {
		return bad("employment_type must be full-time or part-time")
	}
	if req.Rate < 1 || req.Rate > 999999 {
		return bad("rate must be between 1 and 999999")
	}
	period := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(req.RatePeriod)), "per ")
	if period == "" {
		period = "hour"
	}
	switch period {
	case "hour", "day", "week", "month":
	default:
		return bad("rate_period must be hour, day, week or month")
	}

	city := strings.TrimSpace(req.City)
	loc := strings.TrimSpace(req.MapLocation)
	if n := utf8.RuneCountInString(city); n < 1 || n > 100 {
		return bad("city is required (max 100 characters)")
	}
	if n := utf8.RuneCountInString(loc); n < 1 || n > 500 {
		return bad("map_location is required (max 500 characters)")
	}
	if (req.Lat == nil) != (req.Lng == nil) {
		return bad("lat and lng must be sent together")
	}
	if req.Lat != nil && (*req.Lat < -90 || *req.Lat > 90 || *req.Lng < -180 || *req.Lng > 180) {
		return bad("lat/lng out of range")
	}

	var sched *time.Time
	switch req.Timing {
	case "now":
	case "later":
		t, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			return bad("scheduled_at must be an RFC 3339 timestamp, e.g. 2026-10-09T08:00:00+08:00")
		}
		if t.Before(time.Now()) || t.After(time.Now().AddDate(1, 0, 0)) {
			return bad("scheduled_at must be in the future (within a year)")
		}
		sched = &t
	default:
		return bad("timing must be now or later")
	}

	crew := req.CrewNeeded
	if crew == 0 {
		crew = 1
	}
	if crew < 1 || crew > 20 {
		return bad("crew_needed must be between 1 and 20")
	}

	return &store.Job{
		Category: req.Category, RoleID: role.ID, RoleLabel: role.Label,
		EmploymentType: req.EmploymentType, RateCentavos: int64(math.Round(req.Rate * 100)),
		RatePeriod: period, City: city, MapLocation: loc, MapsURL: buildMapsURL(loc, req.Lat, req.Lng),
		Lat: req.Lat, Lng: req.Lng, Timing: req.Timing, ScheduledAt: sched, CrewNeeded: crew,
	}, true
}

// ── endpoints ──

// GET /api/catalog
func (h *Handler) Catalog(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Store.ListRoles(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	labels := map[string]string{}
	for _, c := range Catalog {
		labels[c.ID] = c.Label
	}
	groups := map[string]*Category{}
	ordered := []*Category{}
	for _, row := range rows {
		c := groups[row.Category]
		if c == nil {
			c = &Category{ID: row.Category, Label: labels[row.Category], Roles: []Role{}}
			groups[row.Category] = c
			ordered = append(ordered, c)
		}
		c.Roles = append(c.Roles, Role{ID: row.ID, Label: row.Label})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"categories": ordered})
}

// POST /api/jobs  → creates a draft (the "Find a Crew" click, before payment)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	j, ok := h.parse(w, r)
	if !ok {
		return
	}
	j.EmployerID = u.ID
	if err := h.Store.CreateJob(r.Context(), j); err != nil {
		h.fail(w, err)
		return
	}
	h.writeJob(w, r, http.StatusCreated, u.ID, j.ID)
}

// PUT /api/jobs/{id}  → "Edit Shift Details" (draft or unfulfilled only)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	j, ok := h.parse(w, r)
	if !ok {
		return
	}
	j.ID, j.EmployerID = id, empID
	if err := h.Store.UpdateJob(r.Context(), j); err != nil {
		h.fail(w, err)
		return
	}
	h.writeJob(w, r, http.StatusOK, empID, id)
}

// GET /api/jobs?status=active,completed&limit=20&offset=0
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	valid := map[string]bool{
		"draft": true, "matching": true, "accepted": true, "active": true,
		"completed": true, "unfulfilled": true, "cancelled": true,
	}

	var statuses []string
	if q := r.URL.Query().Get("status"); q != "" {
		for _, s := range strings.Split(q, ",") {
			s = strings.TrimSpace(s)
			if !valid[s] {
				httpx.Error(w, http.StatusBadRequest, "unknown status: "+s)
				return
			}
			statuses = append(statuses, s)
		}
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	_, _ = h.Store.ExpireSearches(r.Context())
	empID := u.ID
	if u.Role == "admin" {
		empID = 0
	}
	jobs, err := h.Store.ListJobs(r.Context(), empID, statuses, limit, offset)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := make([]jobDTO, 0, len(jobs))
	for _, j := range jobs {
		d := h.toDTO(j)
		crew, crewErr := h.Store.Assignments(r.Context(), j.ID)
		if crewErr != nil {
			h.fail(w, crewErr)
			return
		}
		for _, c := range crew {
			d.Crew = append(d.Crew, crewDTO{c.ApplicantID, c.Name, c.AcceptedAt, c.EndedAt, c.Transit})
		}
		if pay, payErr := h.Store.LatestJobPayment(r.Context(), j.ID); payErr != nil {
			h.fail(w, payErr)
			return
		} else if pay != nil {
			d.Payment = pay
		}
		out = append(out, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"jobs": out})
}

// GET /api/jobs/{id}  → the endpoint the matching screen polls
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	h.writeJob(w, r, http.StatusOK, empID, id)
}

// GET /api/jobs/{id}/matches returns limited profile data for active crew who
// match this employer's currently open search. The job lookup enforces ownership.
func (h *Handler) Matches(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	if _, err := h.Store.GetJob(r.Context(), empID, id); err != nil {
		h.fail(w, err)
		return
	}
	matches, err := h.Store.MatchingApplicants(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"matches": matches})
}

// POST /api/jobs/{id}/search opens a crew search after its fee has been recorded.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	if err := h.Store.StartSearch(r.Context(), empID, id, h.SearchWindow); err != nil {
		h.fail(w, err)
		return
	}
	h.writeJob(w, r, http.StatusOK, empID, id)
}

// POST /api/jobs/{id}/pay body: {"method":"gcash", "proof_image":"..."} records the search fee before crew are notified.
func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	var req struct {
		Method     string `json:"method"`
		Details    string `json:"details"`
		ProofImage string `json:"proof_image"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	req.Method = strings.ToLower(strings.TrimSpace(req.Method))
	if req.Method != "cash" && req.Method != "gcash" && req.Method != "maya" && req.Method != "card" {
		httpx.Error(w, http.StatusBadRequest, "method must be cash, gcash, maya or card")
		return
	}
	req.Details = strings.TrimSpace(req.Details)
	req.ProofImage = strings.TrimSpace(req.ProofImage)
	if len(req.Details) > 255 {
		httpx.Error(w, http.StatusBadRequest, "payment details must be 255 characters or fewer")
		return
	}
	if (req.Method == "gcash" || req.Method == "maya") && req.ProofImage == "" && req.Details == "" {
		httpx.Error(w, http.StatusBadRequest, "proof of payment screenshot is required for GCash and Maya")
		return
	}

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	ref := "pay_" + hex.EncodeToString(b)

	if err := h.Store.RecordMatchFee(r.Context(), empID, id, req.Method, req.Details, req.ProofImage, h.MatchFeeCentavos, ref); err != nil {
		h.fail(w, err)
		return
	}
	h.writeJob(w, r, http.StatusOK, empID, id)
}

// POST /api/jobs/{id}/confirm  → "Confirm Booking" after a crew member accepted
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	crew, err := h.Store.Assignments(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := h.Store.ConfirmJob(r.Context(), empID, id); err != nil {
		h.fail(w, err)
		return
	}
	for _, c := range crew {
		if err := h.Store.AddNotification(r.Context(), c.ApplicantID, "Booking confirmed", "The employer confirmed your shift booking.", "accept", "{}"); err != nil {
			log.Printf("notification: %v", err)
		}
	}
	h.writeJob(w, r, http.StatusOK, empID, id)
}

// POST /api/jobs/{id}/end  → "End Contract"
func (h *Handler) End(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	crew, err := h.Store.Assignments(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := h.Store.EndJob(r.Context(), empID, id); err != nil {
		h.fail(w, err)
		return
	}
	for _, c := range crew {
		if err := h.Store.AddNotification(r.Context(), c.ApplicantID, "Contract ended", "The employer ended your shift contract.", "end", "{}"); err != nil {
			log.Printf("notification: %v", err)
		}
	}
	h.writeJob(w, r, http.StatusOK, empID, id)
}

// POST /api/jobs/{id}/cancel
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	if err := h.Store.CancelJob(r.Context(), empID, id); err != nil {
		h.fail(w, err)
		return
	}
	h.writeJob(w, r, http.StatusOK, empID, id)
}

// POST /api/jobs/{id}/decline-match
func (h *Handler) DeclineMatch(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	job, err := h.Store.DeclineMatch(r.Context(), empID, id, h.SearchWindow)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"job": h.toDTO(job)})
}

// POST /api/jobs/{id}/next-attempt
func (h *Handler) NextAttempt(w http.ResponseWriter, r *http.Request) {
	_, id, empID, ok := h.ids(w, r)
	if !ok {
		return
	}
	job, err := h.Store.AdvanceSearchAttempt(r.Context(), empID, id, h.SearchWindow)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"job": h.toDTO(job)})
}

// POST /api/upload-proof
func (h *Handler) UploadProof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Limit to 15MB
	r.Body = http.MaxBytesReader(w, r.Body, 15<<20)
	if err := r.ParseMultipartForm(15 << 20); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid file upload or file too large (max 15MB)")
		return
	}
	file, header, err := r.FormFile("proof")
	if err != nil {
		file, header, err = r.FormFile("file")
	}
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "no proof file provided")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" {
		ext = ".jpg"
	}

	if err := os.MkdirAll("uploads", 0o755); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to create uploads directory")
		return
	}

	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)
	filename := fmt.Sprintf("proof_%d_%s%s", time.Now().Unix(), hex.EncodeToString(randBytes), ext)
	dstPath := filepath.Join("uploads", filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to save uploaded file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"url":      "/uploads/" + filename,
		"filename": filename,
	})
}

// GET /api/admin/payments
func (h *Handler) AdminPayments(w http.ResponseWriter, r *http.Request) {
	records, err := h.Store.ListAllPayments(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"payments": records,
		"count":    len(records),
	})
}

// POST /api/admin/payments/verify
func (h *Handler) AdminVerifyPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type     string `json:"type"` // "employer" or "applicant"
		ID       int64  `json:"id"`
		Verified *bool  `json:"verified,omitempty"`
		Status   string `json:"status,omitempty"` // "verified", "rejected", "pending"
		Reason   string `json:"reason,omitempty"` // optional note shown when rejected
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if req.Type != "employer" && req.Type != "applicant" {
		httpx.Error(w, http.StatusBadRequest, "type must be employer or applicant")
		return
	}
	if req.ID < 1 {
		httpx.Error(w, http.StatusBadRequest, "invalid payment id")
		return
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		if req.Verified != nil && *req.Verified {
			status = "verified"
		} else {
			status = "pending"
		}
	}
	verifier := "Accounting Staff"
	if u, ok := auth.UserFrom(r.Context()); ok && u != nil && u.Name != "" {
		verifier = u.Name
	}
	reason := strings.TrimSpace(req.Reason)
	if len(reason) > 500 {
		reason = reason[:500]
	}
	if status != "rejected" {
		reason = ""
	}
	if err := h.Store.SetPaymentStatus(r.Context(), req.Type, req.ID, status, verifier, reason); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "payment not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"status":   status,
		"verified": status == "verified",
		"type":     req.Type,
		"id":       req.ID,
		"reason":   reason,
	})
}
