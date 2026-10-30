package store_test

import (
	"context"
	"sync"
	"testing"

	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

// raceFixture is an employer, and an applicant who can work two roles in Cebu City.
func raceFixture(t *testing.T) (st *store.Store, openShift func(roleID, roleLabel, category string) int64, employerID, crewID int64, exec func(string, ...any)) {
	t.Helper()
	st, raw := storetest.Open(t)
	ctx := context.Background()
	must := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	must("catalog", jobs.SyncCatalog(ctx, st))
	employer := &store.User{Email: "e@test.com", PasswordHash: "x", Role: "employer", Name: "Emp", Company: "Co"}
	crew := &store.User{Email: "c@test.com", PasswordHash: "x", Role: "applicant", Name: "Crew"}
	must("employer", st.CreateUser(ctx, employer))
	must("crew", st.CreateUser(ctx, crew))
	active := true
	must("profile", st.SaveApplicantProfile(ctx, crew.ID, store.ApplicantProfile{EmploymentType: "full-time", ExpectedRate: 100,
		RatePeriod: "hour", City: "Cebu City", Availability: "now", Active: &active, Roles: []string{"cook", "waiter-server"}}))
	exec = func(q string, args ...any) {
		t.Helper()
		_, err := raw.Exec(q, args...)
		must(q, err)
	}
	openShift = func(roleID, roleLabel, category string) int64 {
		t.Helper()
		j := &store.Job{EmployerID: employer.ID, Category: category, RoleID: roleID, RoleLabel: roleLabel,
			EmploymentType: "full-time", RateCentavos: 10000, RatePeriod: "hour", City: "Cebu City",
			MapLocation: "IT Park", Timing: "now", CrewNeeded: 1}
		must("create job", st.CreateJob(ctx, j))
		must("start search", st.StartSearch(ctx, employer.ID, j.ID))
		return j.ID
	}
	return st, openShift, employer.ID, crew.ID, exec
}

// One remaining hire must never let the same applicant hold two shifts at once.
func TestAcceptJob_OneHireCoversOneShift(t *testing.T) {
	st, openShift, _, crewID, exec := raceFixture(t)
	ctx := context.Background()
	for round := 0; round < 10; round++ {
		exec(`UPDATE job_assignments SET ended_at=1 WHERE applicant_id=$1 AND ended_at IS NULL`, crewID)
		exec(`UPDATE jobs SET status='cancelled' WHERE status IN ('matching','accepted')`)
		exec(`INSERT INTO applicant_hiring_passes(applicant_id,total_hires,remaining_hires,used_hires,amount_centavos,batches_paid,updated_at)
			VALUES($1,1,1,0,0,0,0) ON CONFLICT(applicant_id) DO UPDATE SET remaining_hires=1, total_hires=applicant_hiring_passes.used_hires+1`, crewID)
		ids := []int64{openShift("cook", "Cook", "kitchen"), openShift("waiter-server", "Waiter / Server", "hospitality")}

		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make([]error, len(ids))
		for i, id := range ids {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs[i] = st.AcceptJob(ctx, id, crewID)
			}()
		}
		close(start)
		wg.Wait()
		accepted := 0
		for _, err := range errs {
			if err == nil {
				accepted++
			}
		}
		if accepted != 1 {
			t.Fatalf("round %d: %d shifts accepted with one hire (errors: %v)", round, accepted, errs)
		}
	}
}
