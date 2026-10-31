package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

func TestStore_OpenAndMigrateNew(t *testing.T) {
	st, _ := storetest.Open(t)
	if err := st.EnsureAdminUser(context.Background(), "admin@instantcrew.com", "admin123", "Instant Crew Administrator"); err != nil {
		t.Fatalf("EnsureAdminUser failed: %v", err)
	}
}

func TestStore_PaymentVerificationFlows(t *testing.T) {
	st, raw := storetest.Open(t)

	ctx := context.Background()
	if err := jobs.SyncCatalog(ctx, st); err != nil {
		t.Fatalf("SyncCatalog failed: %v", err)
	}

	// 1. Create Employer, Applicant, Admin
	employer := &store.User{
		Email:        "employer@test.com",
		PasswordHash: "hash1",
		Role:         "employer",
		Name:         "Test Employer",
		Company:      "Test Co",
	}
	if err := st.CreateUser(ctx, employer); err != nil {
		t.Fatalf("create employer: %v", err)
	}

	applicant := &store.User{
		Email:        "applicant@test.com",
		PasswordHash: "hash2",
		Role:         "applicant",
		Name:         "Test Applicant",
	}
	if err := st.CreateUser(ctx, applicant); err != nil {
		t.Fatalf("create applicant: %v", err)
	}

	if err := st.EnsureAdminUser(ctx, "admin@instantcrew.com", "admin123", "System Admin"); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}

	// Setup applicant profile
	appActive := true
	profile := store.ApplicantProfile{
		EmploymentType: "full-time",
		ExpectedRate:   150,
		RatePeriod:     "hour",
		City:           "Cebu City",
		Location:       "IT Park",
		Availability:   "now",
		Active:         &appActive,
		Roles:          []string{"waiter-server"},
	}
	if err := st.SaveApplicantProfile(ctx, applicant.ID, profile); err != nil {
		t.Fatalf("save applicant profile: %v", err)
	}

	// 2. Create Job Draft
	job := &store.Job{
		EmployerID:     employer.ID,
		Category:       "hospitality",
		RoleID:         "waiter-server",
		RoleLabel:      "Waiter / Server",
		EmploymentType: "full-time",
		RateCentavos:   15000,
		RatePeriod:     "hour",
		City:           "Cebu City",
		MapLocation:    "IT Park, Cebu",
		Timing:         "now",
		CrewNeeded:     1,
	}
	if err := st.CreateJob(ctx, job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	// 3. The search starts without any fee; the fee is only due once crew accept.
	if err := st.StartSearch(ctx, employer.ID, job.ID); err != nil {
		t.Fatalf("StartSearch failed: %v", err)
	}
	if err := st.RecordMatchFee(ctx, employer.ID, job.ID, "gcash", "early", "/uploads/early.jpg", 10000, "pay_early"); !errors.Is(err, store.ErrBadState) {
		t.Fatalf("expected ErrBadState paying before a crew member accepted, got %v", err)
	}

	// 4. Test Applicant Hiring Pass Flow
	// Check initial hiring pass (0 remaining)
	pass, err := st.ApplicantHiringPass(ctx, applicant.ID)
	if err != nil {
		t.Fatalf("ApplicantHiringPass failed: %v", err)
	}
	if pass.RemainingHires != store.InitialFreeHires {
		t.Fatalf("expected %d free starter hires, got %d", store.InitialFreeHires, pass.RemainingHires)
	}
	if pass.PaidAmount != 0 || pass.BatchesPaid != 0 {
		t.Fatalf("free hires must not count as a paid batch: %+v", pass)
	}

	// Simulate the free hires being used up so the paid top-up flow below applies.
	if _, err := raw.Exec(`UPDATE applicant_hiring_passes SET total_hires=0, remaining_hires=0, used_hires=0 WHERE applicant_id=$1`, applicant.ID); err != nil {
		t.Fatalf("exhaust free hires: %v", err)
	}

	// Attempting AcceptJob fails with ErrHiringPassEmpty
	err = st.AcceptJob(ctx, job.ID, applicant.ID)
	if !errors.Is(err, store.ErrHiringPassEmpty) {
		t.Fatalf("expected ErrHiringPassEmpty, got %v", err)
	}

	// Applicant pays ₱10 for hiring pass (uploads proof)
	pass, err = st.AddApplicantHiringPass(ctx, applicant.ID, "gcash", "/uploads/pass_proof1.jpg", 0)
	if err != nil {
		t.Fatalf("AddApplicantHiringPass failed: %v", err)
	}
	if pass.PendingPayment == nil || pass.PendingPayment.Status != "pending" {
		t.Fatalf("expected pending payment on hiring pass, got %+v", pass.PendingPayment)
	}
	if pass.RemainingHires != 0 {
		t.Fatalf("expected 0 remaining hires before verification, got %d", pass.RemainingHires)
	}
	// A second receipt while the first is pending must not overwrite it.
	if _, err := st.AddApplicantHiringPass(ctx, applicant.ID, "gcash", "/uploads/pass_proof_dup.jpg", 0); !errors.Is(err, store.ErrPaymentPendingExists) {
		t.Fatalf("expected ErrPaymentPendingExists, got %v", err)
	}

	// Attempting AcceptJob now returns ErrHiringPassPending
	err = st.AcceptJob(ctx, job.ID, applicant.ID)
	if !errors.Is(err, store.ErrHiringPassPending) {
		t.Fatalf("expected ErrHiringPassPending, got %v", err)
	}

	// Admin finds applicant payment in admin list
	allPayments, err := st.ListAllPayments(ctx)
	if err != nil {
		t.Fatalf("ListAllPayments failed: %v", err)
	}
	var appPayRecord *store.AdminPaymentRecord
	for i := range allPayments {
		if allPayments[i].Type == "applicant" && allPayments[i].UserID == applicant.ID {
			appPayRecord = &allPayments[i]
			break
		}
	}
	if appPayRecord == nil {
		t.Fatalf("applicant payment record not found in admin list")
	}

	// Admin rejects payment
	err = st.SetPaymentStatus(ctx, "applicant", appPayRecord.RawID, "rejected", "Admin Jane", "Invalid reference number")
	if err != nil {
		t.Fatalf("SetPaymentStatus reject applicant payment failed: %v", err)
	}

	err = st.AcceptJob(ctx, job.ID, applicant.ID)
	if !errors.Is(err, store.ErrPaymentRejected) {
		t.Fatalf("expected ErrPaymentRejected, got %v", err)
	}

	// Applicant re-uploads
	pass, err = st.AddApplicantHiringPass(ctx, applicant.ID, "gcash", "/uploads/pass_proof2.jpg", 0)
	if err != nil {
		t.Fatalf("AddApplicantHiringPass re-upload failed: %v", err)
	}
	if pass.PendingPayment.Status != "pending" {
		t.Fatalf("expected pending status, got %s", pass.PendingPayment.Status)
	}

	// Admin verifies payment
	err = st.SetPaymentStatus(ctx, "applicant", appPayRecord.RawID, "verified", "Admin Jane", "")
	if err != nil {
		t.Fatalf("SetPaymentStatus verify applicant payment failed: %v", err)
	}

	// Pass should now have 5 hires!
	pass, err = st.ApplicantHiringPass(ctx, applicant.ID)
	if err != nil {
		t.Fatalf("ApplicantHiringPass failed: %v", err)
	}
	if pass.RemainingHires != 5 {
		t.Fatalf("expected 5 remaining hires, got %d", pass.RemainingHires)
	}

	// Verify payment again (idempotent; should not grant another 5)
	err = st.SetPaymentStatus(ctx, "applicant", appPayRecord.RawID, "verified", "Admin Jane", "")
	if err != nil {
		t.Fatalf("second verify failed: %v", err)
	}
	pass, err = st.ApplicantHiringPass(ctx, applicant.ID)
	if err != nil || pass.RemainingHires != 5 {
		t.Fatalf("expected still 5 remaining hires, got %d", pass.RemainingHires)
	}
	// Rejecting after hires were granted would leave them free, so it is refused.
	if err := st.SetPaymentStatus(ctx, "applicant", appPayRecord.RawID, "rejected", "Admin Jane", "oops"); !errors.Is(err, store.ErrAlreadyVerified) {
		t.Fatalf("expected ErrAlreadyVerified, got %v", err)
	}

	// Applicant accepts job -> succeeds, but NO hire is spent yet (employer has not confirmed).
	err = st.AcceptJob(ctx, job.ID, applicant.ID)
	if err != nil {
		t.Fatalf("AcceptJob failed after hiring pass verified: %v", err)
	}
	pass, err = st.ApplicantHiringPass(ctx, applicant.ID)
	if err != nil || pass.RemainingHires != 5 {
		t.Fatalf("expected 5 remaining hires before employer confirmation, got %d", pass.RemainingHires)
	}

	// 5. Employer Search Fee Flow, now that a crew member accepted.
	// The crew member can't walk away while the employer is paying for them.
	if err := st.EndApplicantAssignment(ctx, job.ID, applicant.ID); !errors.Is(err, store.ErrBookingPending) {
		t.Fatalf("expected ErrBookingPending ending before confirmation, got %v", err)
	}
	if err := st.ConfirmJob(ctx, employer.ID, job.ID); !errors.Is(err, store.ErrPaymentNotVerified) {
		t.Fatalf("expected ErrPaymentNotVerified confirming before paying, got %v", err)
	}
	if err := st.RecordMatchFee(ctx, employer.ID, job.ID, "gcash", "Receipt 123", "/uploads/proof1.jpg", 10000, "pay_ref_1"); err != nil {
		t.Fatalf("RecordMatchFee failed: %v", err)
	}
	pay, err := st.LatestJobPayment(ctx, job.ID)
	if err != nil || pay == nil || pay.Status != "pending" {
		t.Fatalf("expected pending fee, got %+v (%v)", pay, err)
	}
	if err := st.ConfirmJob(ctx, employer.ID, job.ID); !errors.Is(err, store.ErrPaymentNotVerified) {
		t.Fatalf("expected ErrPaymentNotVerified while pending, got %v", err)
	}
	if err := st.SetPaymentStatus(ctx, "employer", pay.ID, "rejected", "Admin Jane", "Receipt screenshot blurry"); err != nil {
		t.Fatalf("SetPaymentStatus reject failed: %v", err)
	}
	pay, err = st.LatestJobPayment(ctx, job.ID)
	if err != nil || pay.Status != "rejected" || pay.RejectionReason != "Receipt screenshot blurry" {
		t.Fatalf("expected rejected with reason, got: %+v", pay)
	}
	if err := st.ConfirmJob(ctx, employer.ID, job.ID); !errors.Is(err, store.ErrPaymentRejected) {
		t.Fatalf("expected ErrPaymentRejected, got %v", err)
	}
	if err := st.RecordMatchFee(ctx, employer.ID, job.ID, "gcash", "Receipt 456", "/uploads/proof2.jpg", 10000, "pay_ref_2"); err != nil {
		t.Fatalf("re-upload RecordMatchFee failed: %v", err)
	}
	// Verifying the fee confirms the booking and spends the crew member's hire.
	if err := st.SetPaymentStatus(ctx, "employer", pay.ID, "verified", "Admin Jane", ""); err != nil {
		t.Fatalf("SetPaymentStatus verify failed: %v", err)
	}
	got, err := st.GetJob(ctx, employer.ID, job.ID)
	if err != nil || got.Status != "active" {
		t.Fatalf("expected active job after fee verification, got %+v (%v)", got, err)
	}
	pass, err = st.ApplicantHiringPass(ctx, applicant.ID)
	if err != nil || pass.RemainingHires != 4 || pass.UsedHires != 1 {
		t.Fatalf("expected 4 remaining / 1 used after confirmation, got %+v", pass)
	}

	// 6. Crew hold at most 5 hires and pay ₱2 per hire: with 4 held only 1 more fits.
	if _, err := st.AddApplicantHiringPass(ctx, applicant.ID, "gcash", "/uploads/pass_proof3.jpg", 2); !errors.Is(err, store.ErrHiringPassFull) {
		t.Fatalf("buying past 5 held hires: want ErrHiringPassFull, got %v", err)
	}
	if _, err := st.AddApplicantHiringPass(ctx, applicant.ID, "gcash", "/uploads/pass_proof3.jpg", 0); err != nil {
		t.Fatalf("filling up to 5: %v", err)
	}
	var amount int64
	var topupID int64
	if err := raw.QueryRow(`SELECT id, amount_centavos FROM applicant_hiring_pass_payments WHERE applicant_id=$1 AND status='pending'`, applicant.ID).Scan(&topupID, &amount); err != nil || amount != 200 {
		t.Fatalf("1 hire should cost ₱2.00 (200), got %d (%v)", amount, err)
	}
	if err := st.SetPaymentStatus(ctx, "applicant", topupID, "verified", "Admin Jane", ""); err != nil {
		t.Fatalf("verify top-up: %v", err)
	}
	pass, err = st.ApplicantHiringPass(ctx, applicant.ID)
	if err != nil || pass.RemainingHires != 5 {
		t.Fatalf("expected 5 held after a 1-hire top-up, got %+v", pass)
	}
}

// Ending crew who are on the way adds ₱50 each to the employer's next search fee,
// cleared once that fee is verified. Ending crew who haven't left costs nothing.
func TestStore_TransitCancelPenalty(t *testing.T) {
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
		RatePeriod: "hour", City: "Cebu City", Availability: "now", Active: &active, Roles: []string{"cook"}}))
	_, err := raw.Exec(`UPDATE applicant_hiring_passes SET total_hires=10, remaining_hires=10 WHERE applicant_id=$1`, crew.ID)
	must("hires", err)

	penalty := func() int64 {
		t.Helper()
		p, err := st.EmployerSearchPenalty(ctx, employer.ID)
		must("penalty", err)
		return p
	}
	// book searches, gets an acceptance and pays; verify=true also verifies the fee.
	book := func(verify bool) (jobID int64, pay *store.JobPayment) {
		t.Helper()
		j := &store.Job{EmployerID: employer.ID, Category: "kitchen", RoleID: "cook", RoleLabel: "Cook",
			EmploymentType: "full-time", RateCentavos: 10000, RatePeriod: "hour", City: "Cebu City",
			MapLocation: "IT Park", Timing: "now", CrewNeeded: 1}
		must("create job", st.CreateJob(ctx, j))
		must("search", st.StartSearch(ctx, employer.ID, j.ID))
		must("accept", st.AcceptJob(ctx, j.ID, crew.ID))
		must("fee", st.RecordMatchFee(ctx, employer.ID, j.ID, "gcash", "", "/uploads/fee.jpg", 10000, "ref"))
		pay, err := st.LatestJobPayment(ctx, j.ID)
		must("latest payment", err)
		if verify {
			must("verify", st.SetPaymentStatus(ctx, "employer", pay.ID, "verified", "Admin", ""))
		}
		return j.ID, pay
	}
	amount := func(paymentID int64) (total, pen int64) {
		t.Helper()
		must("amount", raw.QueryRow(`SELECT amount_centavos, penalty_centavos FROM payments WHERE id=$1`, paymentID).Scan(&total, &pen))
		return
	}

	// Ending before the crew left: no penalty.
	id, _ := book(true)
	must("end", st.EndJob(ctx, employer.ID, id))
	if p := penalty(); p != 0 {
		t.Fatalf("ending crew who hadn't left must not add a penalty, got %d", p)
	}

	// Ending twice while the crew is on the way: ₱50 each, adding up.
	for i := 0; i < 2; i++ {
		id, _ = book(true)
		must("en route", st.UpdateApplicantTransit(ctx, id, crew.ID, "en_route", "t", "m", "transit"))
		must("end", st.EndJob(ctx, employer.ID, id))
	}
	if p := penalty(); p != 2*store.TransitCancelPenaltyCentavos {
		t.Fatalf("want penalty %d, got %d", 2*store.TransitCancelPenaltyCentavos, p)
	}
	// The first booking's crew hadn't left, so its hire stays spent (10 -> 9); the
	// two ended on the way got theirs back.
	pass, err := st.ApplicantHiringPass(ctx, crew.ID)
	must("hiring pass", err)
	if pass.RemainingHires != 9 || pass.UsedHires != 1 {
		t.Fatalf("want 9 remaining / 1 used after transit refunds, got %d / %d", pass.RemainingHires, pass.UsedHires)
	}

	// The next fee includes the penalty (₱200), and verifying it clears the penalty.
	_, pay := book(false)
	if total, pen := amount(pay.ID); total != 20000 || pen != 10000 {
		t.Fatalf("want fee 20000 incl. 10000 penalty, got %d incl. %d", total, pen)
	}
	must("verify", st.SetPaymentStatus(ctx, "employer", pay.ID, "verified", "Admin", ""))
	if p := penalty(); p != 0 {
		t.Fatalf("verified fee must clear its penalty, got %d", p)
	}
}

// Two accounting staff (or a double-click) verifying the same ₱10 receipt at
// once must add the 5 hires only once.
func TestStore_ConcurrentHiringPassVerifyGrantsOnce(t *testing.T) {
	st, raw := storetest.Open(t)
	ctx := context.Background()
	app := &store.User{Email: "race@test.com", PasswordHash: "x", Role: "applicant", Name: "Race"}
	if err := st.CreateUser(ctx, app); err != nil {
		t.Fatal(err)
	}
	before, err := st.ApplicantHiringPass(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	pass, err := st.AddApplicantHiringPass(ctx, app.ID, "gcash", "/uploads/proof_1_ab.jpg", 0)
	if err != nil || pass.PendingPayment == nil {
		t.Fatalf("AddApplicantHiringPass: %v %+v", err, pass)
	}

	// Hold the payment row so every verify reads it before any of them commits;
	// without that, a fast local database finishes each verify before the next starts.
	hold, err := raw.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hold.Exec(`UPDATE applicant_hiring_pass_payments SET method=method WHERE id=$1`, pass.PendingPayment.ID); err != nil {
		t.Fatal(err)
	}
	const n = 4
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			errs <- st.SetPaymentStatus(ctx, "applicant", pass.PendingPayment.ID, "verified", "Accounting", "")
		}()
	}
	time.Sleep(300 * time.Millisecond)
	if err := hold.Commit(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("SetPaymentStatus: %v", err)
		}
	}

	after, err := st.ApplicantHiringPass(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := after.RemainingHires - before.RemainingHires; got != 5 {
		t.Fatalf("concurrent verifies added %d hires, want 5", got)
	}
}

// A crew member whose employer submits no receipt within PaymentDeadline is
// released and the search reopens; a submitted receipt keeps them booked.
func TestStore_ReleaseUnpaidMatches(t *testing.T) {
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
	must("employer", st.CreateUser(ctx, employer))
	accepted := func(email string) int64 {
		t.Helper()
		crew := &store.User{Email: email, PasswordHash: "x", Role: "applicant", Name: "Crew"}
		must("crew", st.CreateUser(ctx, crew))
		active := true
		must("profile", st.SaveApplicantProfile(ctx, crew.ID, store.ApplicantProfile{EmploymentType: "full-time", ExpectedRate: 100,
			RatePeriod: "hour", City: "Cebu City", Availability: "now", Active: &active, Roles: []string{"cook"}}))
		_, err := raw.Exec(`UPDATE applicant_hiring_passes SET total_hires=5, remaining_hires=5 WHERE applicant_id=$1`, crew.ID)
		must("hires", err)
		j := &store.Job{EmployerID: employer.ID, Category: "kitchen", RoleID: "cook", RoleLabel: "Cook",
			EmploymentType: "full-time", RateCentavos: 10000, RatePeriod: "hour", City: "Cebu City",
			MapLocation: "IT Park", Timing: "now", CrewNeeded: 1}
		must("create job", st.CreateJob(ctx, j))
		must("search", st.StartSearch(ctx, employer.ID, j.ID))
		must("accept", st.AcceptJob(ctx, j.ID, crew.ID))
		return j.ID
	}
	unpaid, paid, fresh := accepted("a@test.com"), accepted("b@test.com"), accepted("c@test.com")
	must("fee", st.RecordMatchFee(ctx, employer.ID, paid, "gcash", "", "/uploads/fee.jpg", 10000, "ref"))
	_, err := raw.Exec(`UPDATE job_assignments SET accepted_at=$1 WHERE job_id IN ($2,$3)`,
		time.Now().Add(-store.PaymentDeadline-time.Minute).Unix(), unpaid, paid)
	must("backdate", err)

	n, err := st.ReleaseUnpaidMatches(ctx)
	must("release", err)
	if n != 1 {
		t.Fatalf("want 1 released job, got %d", n)
	}
	for id, want := range map[int64]string{unpaid: "matching", paid: "accepted", fresh: "accepted"} {
		j, err := st.GetJob(ctx, employer.ID, id)
		must("get job", err)
		if j.Status != want {
			t.Fatalf("job %d: want %s, got %s", id, want, j.Status)
		}
	}
	var open int
	must("assignments", raw.QueryRow(`SELECT COUNT(*) FROM job_assignments WHERE job_id=$1 AND ended_at IS NULL`, unpaid).Scan(&open))
	if open != 0 {
		t.Fatalf("released crew member must be unassigned, %d still assigned", open)
	}
}

// After a contract ends both sides are asked to rate each other once. The
// employer's stars move the crew member's rating; the crew's stars average into
// the employer's rating; a skip is never asked again.
func TestStore_Reviews(t *testing.T) {
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
		RatePeriod: "hour", City: "Cebu City", Availability: "now", Active: &active, Roles: []string{"cook"}}))
	_, err := raw.Exec(`UPDATE applicant_hiring_passes SET total_hires=5, remaining_hires=5 WHERE applicant_id=$1`, crew.ID)
	must("hires", err)
	finished := func() int64 {
		t.Helper()
		j := &store.Job{EmployerID: employer.ID, Category: "kitchen", RoleID: "cook", RoleLabel: "Cook",
			EmploymentType: "full-time", RateCentavos: 10000, RatePeriod: "hour", City: "Cebu City",
			MapLocation: "IT Park", Timing: "now", CrewNeeded: 1}
		must("create job", st.CreateJob(ctx, j))
		must("search", st.StartSearch(ctx, employer.ID, j.ID))
		must("accept", st.AcceptJob(ctx, j.ID, crew.ID))
		must("fee", st.RecordMatchFee(ctx, employer.ID, j.ID, "gcash", "", "/uploads/fee.jpg", 10000, "ref"))
		pay, err := st.LatestJobPayment(ctx, j.ID)
		must("latest payment", err)
		must("verify", st.SetPaymentStatus(ctx, "employer", pay.ID, "verified", "Admin", ""))
		return j.ID
	}
	pending := func(userID int64) int {
		t.Helper()
		p, err := st.PendingReviews(ctx, userID)
		must("pending", err)
		return len(p)
	}
	rating := func() float64 {
		t.Helper()
		p, err := st.ApplicantProfile(ctx, crew.ID)
		must("profile", err)
		return p.Rating
	}

	id := finished()
	if pending(employer.ID) != 0 || pending(crew.ID) != 0 {
		t.Fatal("a contract still running must not ask for a rating")
	}
	if err := st.SubmitReview(ctx, employer.ID, id, 5); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("rating a running contract: want ErrNotFound, got %v", err)
	}
	must("end", st.EndJob(ctx, employer.ID, id))
	if pending(employer.ID) != 1 || pending(crew.ID) != 1 {
		t.Fatalf("both sides must be asked once, got employer %d crew %d", pending(employer.ID), pending(crew.ID))
	}

	before := rating()
	must("employer rates", st.SubmitReview(ctx, employer.ID, id, 5))
	must("employer rates again", st.SubmitReview(ctx, employer.ID, id, 1))
	if got := rating(); got < before+0.099 || got > before+0.101 {
		t.Fatalf("5 stars once should add 0.10 to %v, got %v", before, got)
	}
	must("crew skips", st.SubmitReview(ctx, crew.ID, id, 0))
	if pending(employer.ID) != 0 || pending(crew.ID) != 0 {
		t.Fatal("rated and skipped contracts must not be asked again")
	}

	id = finished()
	must("end", st.EndJob(ctx, employer.ID, id))
	must("crew rates", st.SubmitReview(ctx, crew.ID, id, 4))
	m, err := st.EmployerRatings(ctx, []int64{employer.ID})
	must("employer ratings", err)
	if r := m[employer.ID]; r.Reviews != 1 || r.Rating != 4 {
		t.Fatalf("skip must not count: want 4.0 from 1 review, got %+v", r)
	}
}
