package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

func TestNoShowPenaltyEscalation(t *testing.T) {
	st, raw := storetest.Open(t)
	ctx := context.Background()
	must := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	must("catalog", jobs.SyncCatalog(ctx, st))

	employer := &store.User{Email: "emp@test.com", PasswordHash: "x", Role: "employer", Name: "Employer"}
	crew := &store.User{Email: "crew@test.com", PasswordHash: "x", Role: "applicant", Name: "Juan Crew"}
	must("create employer", st.CreateUser(ctx, employer))
	must("create crew", st.CreateUser(ctx, crew))
	active := true
	must("profile", st.SaveApplicantProfile(ctx, crew.ID, store.ApplicantProfile{
		EmploymentType: "full-time", ExpectedRate: 100, RatePeriod: "hour", City: "Cebu City",
		Availability: "now", Active: &active, Roles: []string{"cook"},
	}))

	grantHires := func() {
		t.Helper()
		_, err := st.AddApplicantHiringPass(ctx, crew.ID, "gcash", "/uploads/pass.jpg")
		must("hiring pass", err)
		all, err := st.ListAllPayments(ctx)
		must("list payments", err)
		for _, p := range all {
			if p.Type == "applicant" && p.Status == "pending" {
				must("verify pass", st.SetPaymentStatus(ctx, "applicant", p.RawID, "verified", "Admin", ""))
			}
		}
	}
	grantHires()
	grantHires() // 10 hires: one on-time shift plus five no-shows

	// openShift creates a job with a verified fee and starts its crew search.
	openShift := func() int64 {
		t.Helper()
		j := &store.Job{EmployerID: employer.ID, Category: "kitchen", RoleID: "cook", RoleLabel: "Cook",
			EmploymentType: "full-time", RateCentavos: 10000, RatePeriod: "hour", City: "Cebu City",
			MapLocation: "IT Park", Timing: "now", CrewNeeded: 1}
		must("create job", st.CreateJob(ctx, j))
		must("fee", st.RecordMatchFee(ctx, employer.ID, j.ID, "gcash", "", "/uploads/fee.jpg", 10000, "ref"))
		pay, err := st.LatestJobPayment(ctx, j.ID)
		must("latest payment", err)
		must("verify fee", st.SetPaymentStatus(ctx, "employer", pay.ID, "verified", "Admin", ""))
		must("start search", st.StartSearch(ctx, employer.ID, j.ID, 3*time.Minute))
		return j.ID
	}
	// bookShift opens a shift, has the crew accept it and the employer confirm it.
	bookShift := func() int64 {
		t.Helper()
		id := openShift()
		must("accept", st.AcceptJob(ctx, id, crew.ID))
		must("confirm", st.ConfirmJob(ctx, employer.ID, id))
		return id
	}
	expireDeadline := func(jobID int64) {
		t.Helper()
		_, err := raw.Exec(`UPDATE job_assignments SET arrival_deadline_at=$1 WHERE job_id=$2`, time.Now().Unix()-1, jobID)
		must("expire deadline", err)
	}
	rating := func() float64 {
		t.Helper()
		p, err := st.ApplicantProfile(ctx, crew.ID)
		must("profile", err)
		return p.Rating
	}
	standing := func() store.ApplicantStanding {
		t.Helper()
		s, err := st.ApplicantStanding(ctx, crew.ID)
		must("standing", err)
		return s
	}
	// noShow books a shift, lets its arrival limit lapse and checks the outcome.
	noShow := func(strike int) {
		t.Helper()
		id := bookShift()
		before := rating()
		expireDeadline(id)
		n, err := st.ExpireArrivals(ctx)
		must("expire arrivals", err)
		if n != 1 {
			t.Fatalf("strike %d: expected 1 auto-cancellation, got %d", strike, n)
		}
		j, err := st.GetJob(ctx, employer.ID, id)
		must("get job", err)
		if j.Status != "cancelled" {
			t.Fatalf("strike %d: expected cancelled job, got %s", strike, j.Status)
		}
		crewList, err := st.Assignments(ctx, id)
		must("assignments", err)
		if len(crewList) != 1 || crewList[0].Transit != "no_show" || crewList[0].EndedAt == nil {
			t.Fatalf("strike %d: expected concluded no_show assignment, got %+v", strike, crewList)
		}
		if got := before - rating(); got < 0.195 || got > 0.205 {
			t.Fatalf("strike %d: expected rating to drop by 0.20, dropped by %.2f", strike, got)
		}
		if s := standing(); s.NoShowCount != strike {
			t.Fatalf("expected %d strikes, got %d", strike, s.NoShowCount)
		}
	}

	// On-time arrival: the deadline is set on confirmation and no penalty applies.
	onTime := bookShift()
	crewList, err := st.Assignments(ctx, onTime)
	must("assignments", err)
	if d := crewList[0].ArrivalDeadlineAt; d == nil || time.Until(*d) < 59*time.Minute || time.Until(*d) > time.Hour {
		t.Fatalf("expected a 1-hour arrival deadline, got %v", d)
	}
	must("en route", st.UpdateApplicantTransit(ctx, onTime, crew.ID, "en_route", "t", "m", "transit"))
	must("arrived", st.UpdateApplicantTransit(ctx, onTime, crew.ID, "arrived", "t", "m", "arrived"))
	expireDeadline(onTime)
	if n, err := st.ExpireArrivals(ctx); err != nil || n != 0 {
		t.Fatalf("arrived crew must not be penalised: n=%d err=%v", n, err)
	}
	must("end job", st.EndJob(ctx, employer.ID, onTime))

	// Marking transit after the limit lapsed (before the sweep runs) is refused.
	late := bookShift()
	expireDeadline(late)
	if err := st.UpdateApplicantTransit(ctx, late, crew.ID, "en_route", "t", "m", "transit"); !errors.Is(err, store.ErrArrivalExpired) {
		t.Fatalf("expected ErrArrivalExpired, got %v", err)
	}
	if n, err := st.ExpireArrivals(ctx); err != nil || n != 1 {
		t.Fatalf("expected the late shift to be auto-cancelled: n=%d err=%v", n, err)
	}
	notes, err := st.Notifications(ctx, crew.ID)
	must("crew notifications", err)
	if len(notes) == 0 || notes[0].Type != "no_show" || !strings.Contains(notes[0].Message, "0.20 stars") {
		t.Fatalf("expected a no-show notification for the crew, got %+v", notes)
	}
	empNotes, err := st.Notifications(ctx, employer.ID)
	must("employer notifications", err)
	if len(empNotes) == 0 || empNotes[0].Type != "no_show" {
		t.Fatalf("expected a no-show notification for the employer, got %+v", empNotes)
	}
	if s := standing(); s.NoShowCount != 1 || s.Suspended {
		t.Fatalf("strike 1 must not suspend: %+v", s)
	}

	// Strikes 2 and 3: deductions only.
	noShow(2)
	noShow(3)
	if s := standing(); s.Suspended || s.PermanentlyDeactivated {
		t.Fatalf("3 strikes must not suspend: %+v", s)
	}

	// Strike 4 (> 3): 5-day suspension.
	noShow(4)
	s := standing()
	if !s.Suspended || s.DeactivatedUntil == nil {
		t.Fatalf("strike 4 must suspend: %+v", s)
	}
	if until := time.Unix(*s.DeactivatedUntil, 0); time.Until(until) < 5*24*time.Hour-time.Minute || time.Until(until) > 5*24*time.Hour {
		t.Fatalf("expected a 5-day suspension, got until %v", until)
	}
	blocked := openShift()
	if err := st.AcceptJob(ctx, blocked, crew.ID); !errors.Is(err, store.ErrAccountSuspended) {
		t.Fatalf("expected ErrAccountSuspended, got %v", err)
	}
	if m, err := st.MatchingApplicants(ctx, blocked); err != nil || len(m) != 0 {
		t.Fatalf("suspended crew must not be matched: %v %v", m, err)
	}
	must("cancel blocked", st.CancelJob(ctx, employer.ID, blocked))

	// The suspension ends: the account reactivates and records the served suspension.
	_, err = raw.Exec(`UPDATE applicant_profiles SET deactivated_until=$1 WHERE user_id=$2`, time.Now().Unix()-1, crew.ID)
	must("end suspension", err)
	_, err = st.ExpireArrivals(ctx)
	must("sweep", err)
	if s := standing(); s.Suspended || !s.ServedTemporaryDeactivation || s.PermanentlyDeactivated {
		t.Fatalf("expected reactivated account with served suspension: %+v", s)
	}

	// Strike 5 after the served suspension: permanent deactivation.
	noShow(5)
	if s := standing(); !s.PermanentlyDeactivated {
		t.Fatalf("strike 5 must permanently deactivate: %+v", s)
	}
	final := openShift()
	if err := st.AcceptJob(ctx, final, crew.ID); !errors.Is(err, store.ErrAccountDeactivated) {
		t.Fatalf("expected ErrAccountDeactivated, got %v", err)
	}
}
