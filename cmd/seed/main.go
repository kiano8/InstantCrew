// Command seed creates a minimal set of demo employers and crew members with
// specific roles. It is safe to re-run: existing accounts are reused, profiles
// are re-saved, and hires are granted only to crew who have none yet.
//
//	go run ./cmd/seed            # uses the same DB_* env vars as the server
//	SEED_PASSWORD=secret go run ./cmd/seed
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/url"
	"os"

	"golang.org/x/crypto/bcrypt"

	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
)

type seedEmployer struct {
	email, name, company string
}

type seedCrew struct {
	email, name    string
	employmentType string
	rate           int64 // pesos per hour
	roles          []string
}

var employers = []seedEmployer{
	{"kitchen.employer@seed.instantcrew.test", "Maria Santos", "Lechon House Cebu"},
	{"delivery.employer@seed.instantcrew.test", "Carlo Reyes", "QuickHaul Logistics"},
	{"helpers.employer@seed.instantcrew.test", "Ana Dela Cruz", "BuildRight Construction"},
}

// The first role of each crew member is their primary role.
var crew = []seedCrew{
	{"cook@seed.instantcrew.test", "Juan Dela Paz", "full-time", 95, []string{"cook", "all-around-kitchen-crew"}},
	{"waiter@seed.instantcrew.test", "Liza Gomez", "part-time", 85, []string{"waiter-server"}},
	{"barista@seed.instantcrew.test", "Mark Villanueva", "part-time", 90, []string{"barista", "cashier"}},
	{"rider@seed.instantcrew.test", "Ramon Cruz", "full-time", 100, []string{"2-wheel-driver"}},
	{"picker@seed.instantcrew.test", "Joy Mendoza", "full-time", 90, []string{"warehouse-picker-packer", "kargador-loader-unloader"}},
	{"electrician@seed.instantcrew.test", "Paolo Garcia", "full-time", 120, []string{"electrician"}},
	{"builder@seed.instantcrew.test", "Ben Aquino", "full-time", 100, []string{"construction-worker", "carpenter-painter"}},
}

const city = "Cebu City"

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	force := flag.Bool("force", false, "allow seeding when ENV=production")
	flag.Parse()
	if getenv("ENV", "dev") == "production" && !*force {
		log.Fatal("refusing to seed demo accounts with ENV=production; pass -force to override")
	}
	password := getenv("SEED_PASSWORD", "InstantCrew123!")
	if len(password) < 8 || len(password) > 72 {
		log.Fatal("SEED_PASSWORD must be 8 to 72 characters")
	}

	dsn := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(getenv("DB_USER", os.Getenv("USER")), getenv("DB_PASSWORD", "")),
		Host:     getenv("DB_HOST", "localhost") + ":" + getenv("DB_PORT", "5432"),
		Path:     getenv("DB_NAME", "instantcrew_dev"),
		RawQuery: "sslmode=" + url.QueryEscape(getenv("DB_SSLMODE", "disable")),
	}).String()
	st, err := store.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	if err := jobs.SyncCatalog(ctx, st); err != nil {
		log.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range employers {
		u, created, err := ensureUser(ctx, st, &store.User{Email: e.email, PasswordHash: string(hash), Role: "employer", Name: e.name, Company: e.company})
		if err != nil {
			log.Fatalf("employer %s: %v", e.email, err)
		}
		log.Printf("employer  %-42s %-24s %s", u.Email, e.company, status(created))
	}

	active := true
	for _, c := range crew {
		for _, role := range c.roles {
			if _, ok := jobs.RoleByID(role); !ok {
				log.Fatalf("crew %s: role %q is not in the catalog", c.email, role)
			}
		}
		u, created, err := ensureUser(ctx, st, &store.User{Email: c.email, PasswordHash: string(hash), Role: "applicant", Name: c.name})
		if err != nil {
			log.Fatalf("crew %s: %v", c.email, err)
		}
		if err := st.SaveApplicantProfile(ctx, u.ID, store.ApplicantProfile{
			EmploymentType: c.employmentType, ExpectedRate: c.rate, RatePeriod: "hour",
			City: city, Location: "IT Park, Lahug, Cebu City", Availability: "now", Active: &active, Roles: c.roles,
		}); err != nil {
			log.Fatalf("crew %s profile: %v", c.email, err)
		}
		granted, err := ensureHires(ctx, st, u.ID)
		if err != nil {
			log.Fatalf("crew %s hires: %v", c.email, err)
		}
		note := status(created)
		if granted {
			note += ", 5 hires granted"
		}
		log.Printf("crew      %-42s %-24s %s", u.Email, c.roles[0], note)
	}
	log.Printf("done: all seed accounts use the password from SEED_PASSWORD (default InstantCrew123!)")
}

func status(created bool) string {
	if created {
		return "created"
	}
	return "already existed"
}

// ensureUser creates the account, or returns the existing one with the same email and role.
func ensureUser(ctx context.Context, st *store.Store, u *store.User) (*store.User, bool, error) {
	err := st.CreateUser(ctx, u)
	if err == nil {
		return u, true, nil
	}
	if !errors.Is(err, store.ErrEmailTaken) {
		return nil, false, err
	}
	existing, err := st.UserByEmailRole(ctx, u.Email, u.Role)
	return existing, false, err
}

// ensureHires gives crew with no hiring-pass history one verified ₱10 pass
// (5 hires) through the normal receipt flow, so they can accept shifts.
func ensureHires(ctx context.Context, st *store.Store, applicantID int64) (bool, error) {
	pass, err := st.ApplicantHiringPass(ctx, applicantID)
	if err != nil {
		return false, err
	}
	if pass.BatchesPaid > 0 || pass.RemainingHires > 0 || pass.PendingPayment != nil {
		return false, nil
	}
	pass, err = st.AddApplicantHiringPass(ctx, applicantID, "card", "")
	if err != nil {
		return false, err
	}
	if pass.PendingPayment == nil {
		return false, errors.New("hiring pass payment was not recorded")
	}
	return true, st.SetPaymentStatus(ctx, "applicant", pass.PendingPayment.ID, "verified", "Seed data", "")
}
