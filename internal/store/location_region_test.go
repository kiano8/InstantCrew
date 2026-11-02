package store_test

import (
	"context"
	"testing"

	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

func TestApplicantJobs_MatchesWholeRegion(t *testing.T) {
	st, _ := storetest.Open(t)
	ctx := context.Background()
	if err := jobs.SyncCatalog(ctx, st); err != nil {
		t.Fatal(err)
	}
	employer := &store.User{Email: "e@test.com", PasswordHash: "x", Role: "employer", Name: "Emp", Company: "Co"}
	if err := st.CreateUser(ctx, employer); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, applicantCity, jobCity string
		want                         bool
	}{
		{"same Cebu City area", "Lahug, Cebu City", "Lahug, Cebu City", true},
		{"other Cebu City area", "Guadalupe, Cebu City", "Lahug, Cebu City", true},
		{"Cebu town to area", "Oslob, Cebu", "Lahug, Cebu City", true},
		{"region to area", "Cebu", "Talamban, Cebu City", true},
		{"Metro Manila cities", "Pasay, Metro Manila", "Valenzuela, Metro Manila", true},
		{"Cebu is not Metro Manila", "Cebu", "Metro Manila", false},
		{"other cities stay separate", "Davao City", "Iloilo City", false},
		{"Davao is not Cebu", "Davao City", "Cebu", false},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			crew := &store.User{Email: string(rune('a'+i)) + "@test.com", PasswordHash: "x", Role: "applicant", Name: "Crew"}
			if err := st.CreateUser(ctx, crew); err != nil {
				t.Fatal(err)
			}
			active := true
			if err := st.SaveApplicantProfile(ctx, crew.ID, store.ApplicantProfile{EmploymentType: "full-time", ExpectedRate: 100,
				RatePeriod: "hour", City: c.applicantCity, Availability: "now", Active: &active, Roles: []string{"cook"}}); err != nil {
				t.Fatal(err)
			}
			job := &store.Job{EmployerID: employer.ID, Category: "kitchen", RoleID: "cook", RoleLabel: "Cook",
				EmploymentType: "full-time", RateCentavos: 10000, RatePeriod: "hour", City: c.jobCity,
				MapLocation: "Somewhere", Timing: "now", CrewNeeded: 1}
			if err := st.CreateJob(ctx, job); err != nil {
				t.Fatal(err)
			}
			if err := st.StartSearch(ctx, employer.ID, job.ID); err != nil {
				t.Fatal(err)
			}
			got, err := st.ApplicantJobs(ctx, crew.ID)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, j := range got {
				found = found || j.ID == job.ID
			}
			if found != c.want {
				t.Fatalf("%s sees job in %s = %v, want %v", c.applicantCity, c.jobCity, found, c.want)
			}
		})
	}
}
