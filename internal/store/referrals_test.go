package store_test

import (
	"context"
	"testing"

	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

func TestReferralAndShiftReward(t *testing.T) {
	st, raw := storetest.Open(t)
	ctx := context.Background()
	must := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	remaining := func(id int64) int {
		t.Helper()
		p, err := st.ApplicantHiringPass(ctx, id)
		must("hiring pass", err)
		return p.RemainingHires
	}
	must("catalog", jobs.SyncCatalog(ctx, st))

	employer := &store.User{Email: "emp@test.com", PasswordHash: "x", Role: "employer", Name: "Employer"}
	referrer := &store.User{Email: "09170000001", PasswordHash: "x", Role: "applicant", Name: "Ana"}
	crew := &store.User{Email: "09170000002", PasswordHash: "x", Role: "applicant", Name: "Juan"}
	must("create employer", st.CreateUser(ctx, employer))
	must("create referrer", st.CreateUser(ctx, referrer))
	must("create crew", st.CreateUser(ctx, crew))

	// Referral: a valid code credits the referrer exactly once.
	if _, err := st.ReferrerByCode(ctx, store.ReferralCode(employer.ID)); err != store.ErrReferralCode {
		t.Fatalf("employer code must be rejected, got %v", err)
	}
	if _, err := st.ReferrerByCode(ctx, "nope"); err != store.ErrReferralCode {
		t.Fatalf("bad code must be rejected, got %v", err)
	}
	id, err := st.ReferrerByCode(ctx, " "+store.ReferralCode(referrer.ID)+" ")
	must("referrer by code", err)
	if id != referrer.ID {
		t.Fatalf("code resolved to %d, want %d", id, referrer.ID)
	}
	must("credit", st.CreditReferral(ctx, crew.ID, referrer.ID, crew.Name))
	must("credit again", st.CreditReferral(ctx, crew.ID, referrer.ID, crew.Name))
	if got := remaining(referrer.ID); got != 1 {
		t.Fatalf("referrer hires = %d, want 1", got)
	}

	// Shift reward: given once, on the 5th completed shift.
	_, err = raw.Exec(`UPDATE applicant_hiring_passes SET total_hires=10, remaining_hires=10 WHERE applicant_id=$1`, crew.ID)
	must("hires", err)
	active := true
	must("profile", st.SaveApplicantProfile(ctx, crew.ID, store.ApplicantProfile{
		EmploymentType: "full-time", ExpectedRate: 100, RatePeriod: "hour", City: "Cebu City",
		Availability: "now", Active: &active, Roles: []string{"cook"},
	}))
	shift := func() {
		t.Helper()
		j := &store.Job{EmployerID: employer.ID, Category: "kitchen", RoleID: "cook", RoleLabel: "Cook",
			EmploymentType: "full-time", RateCentavos: 10000, RatePeriod: "hour", City: "Cebu City",
			MapLocation: "IT Park", Timing: "now", CrewNeeded: 1}
		must("create job", st.CreateJob(ctx, j))
		must("start search", st.StartSearch(ctx, employer.ID, j.ID))
		must("accept", st.AcceptJob(ctx, j.ID, crew.ID, nil))
		must("fee", st.RecordMatchFee(ctx, employer.ID, j.ID, "gcash", "", "/uploads/fee.jpg", 10000, "ref"))
		pay, err := st.LatestJobPayment(ctx, j.ID)
		must("latest payment", err)
		must("verify fee", st.SetPaymentStatus(ctx, "employer", pay.ID, "verified", "Admin", ""))
		must("en route", st.UpdateApplicantTransit(ctx, j.ID, crew.ID, "en_route", "t", "m", "transit"))
		must("arrived", st.UpdateApplicantTransit(ctx, j.ID, crew.ID, "arrived", "t", "m", "arrived"))
		must("end job", st.EndJob(ctx, employer.ID, j.ID))
	}
	for i := 1; i < store.ShiftRewardAfter; i++ {
		shift()
	}
	if got := remaining(crew.ID); got != 10-(store.ShiftRewardAfter-1) {
		t.Fatalf("before reward: hires = %d", got)
	}
	shift()
	if got, want := remaining(crew.ID), 10-store.ShiftRewardAfter+store.ShiftRewardHires; got != want {
		t.Fatalf("after reward: hires = %d, want %d", got, want)
	}
	shift()
	if got, want := remaining(crew.ID), 10-store.ShiftRewardAfter-1+store.ShiftRewardHires; got != want {
		t.Fatalf("reward must be given once: hires = %d, want %d", got, want)
	}
}
