package store_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
)

func TestStore_OpenAndMigrateNew(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open failed on new db: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	if err := st.EnsureAdminUser(ctx, "admin@instantcrew.com", "admin123", "Instant Crew Administrator"); err != nil {
		t.Fatalf("EnsureAdminUser failed: %v", err)
	}
}

func TestStore_MigrateExistingDB(t *testing.T) {
	srcPath := filepath.Join("..", "..", "data", "instantcrew.db")
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		t.Skip("data/instantcrew.db not found, skipping migration test against existing db")
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "existing_copy.db")

	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("open src: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("create dst: %v", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		t.Fatalf("copy db: %v", err)
	}
	dst.Close()

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open failed on existing db copy: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	if err := st.EnsureAdminUser(ctx, "admin@instantcrew.com", "admin123", "Instant Crew Administrator"); err != nil {
		t.Fatalf("EnsureAdminUser failed: %v", err)
	}
}

func TestStore_PaymentVerificationFlows(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_payments.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open failed: %v", err)
	}
	defer st.Close()

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

	// 3. Test Employer Search Fee Flow
	// Try start search before payment -> should fail
	err = st.StartSearch(ctx, employer.ID, job.ID, 3*time.Minute)
	if !errors.Is(err, store.ErrPaymentNotVerified) {
		t.Fatalf("expected ErrPaymentNotVerified, got %v", err)
	}

	// Submit payment (upload receipt)
	err = st.RecordMatchFee(ctx, employer.ID, job.ID, "gcash", "Receipt 123", "/uploads/proof1.jpg", 10000, "pay_ref_1")
	if err != nil {
		t.Fatalf("RecordMatchFee failed: %v", err)
	}

	// Latest payment should be pending
	pay, err := st.LatestJobPayment(ctx, job.ID)
	if err != nil || pay == nil {
		t.Fatalf("LatestJobPayment failed: %v", err)
	}
	if pay.Status != "pending" {
		t.Fatalf("expected status pending, got %s", pay.Status)
	}

	// Start search should fail while pending
	err = st.StartSearch(ctx, employer.ID, job.ID, 3*time.Minute)
	if !errors.Is(err, store.ErrPaymentNotVerified) {
		t.Fatalf("expected ErrPaymentNotVerified, got %v", err)
	}

	// Admin lists payments -> finds employer payment
	allPayments, err := st.ListAllPayments(ctx)
	if err != nil {
		t.Fatalf("ListAllPayments failed: %v", err)
	}
	var empPayRecord *store.AdminPaymentRecord
	for i := range allPayments {
		if allPayments[i].Type == "employer" && allPayments[i].UserID == employer.ID {
			empPayRecord = &allPayments[i]
			break
		}
	}
	if empPayRecord == nil {
		t.Fatalf("employer payment record not found in admin list")
	}
	if empPayRecord.Status != "pending" {
		t.Fatalf("expected admin record pending, got %s", empPayRecord.Status)
	}

	// Admin rejects with reason
	err = st.SetPaymentStatus(ctx, "employer", empPayRecord.RawID, "rejected", "Admin Jane", "Receipt screenshot blurry")
	if err != nil {
		t.Fatalf("SetPaymentStatus reject failed: %v", err)
	}

	pay, err = st.LatestJobPayment(ctx, job.ID)
	if err != nil || pay == nil || pay.Status != "rejected" || pay.RejectionReason != "Receipt screenshot blurry" {
		t.Fatalf("expected rejected with reason, got: %+v", pay)
	}

	err = st.StartSearch(ctx, employer.ID, job.ID, 3*time.Minute)
	if !errors.Is(err, store.ErrPaymentRejected) {
		t.Fatalf("expected ErrPaymentRejected, got %v", err)
	}

	// Employer re-uploads receipt
	err = st.RecordMatchFee(ctx, employer.ID, job.ID, "gcash", "Receipt 456", "/uploads/proof2.jpg", 10000, "pay_ref_2")
	if err != nil {
		t.Fatalf("re-upload RecordMatchFee failed: %v", err)
	}
	pay, err = st.LatestJobPayment(ctx, job.ID)
	if err != nil || pay.Status != "pending" {
		t.Fatalf("expected pending after re-upload, got: %+v", pay)
	}

	// Admin verifies payment
	err = st.SetPaymentStatus(ctx, "employer", empPayRecord.RawID, "verified", "Admin Jane", "")
	if err != nil {
		t.Fatalf("SetPaymentStatus verify failed: %v", err)
	}

	// Now StartSearch should succeed!
	err = st.StartSearch(ctx, employer.ID, job.ID, 3*time.Minute)
	if err != nil {
		t.Fatalf("StartSearch failed after verification: %v", err)
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
	raw, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	if _, err := raw.Exec(`UPDATE applicant_hiring_passes SET total_hires=0, remaining_hires=0, used_hires=0 WHERE applicant_id=?`, applicant.ID); err != nil {
		raw.Close()
		t.Fatalf("exhaust free hires: %v", err)
	}
	raw.Close()

	// Attempting AcceptJob fails with ErrHiringPassEmpty
	err = st.AcceptJob(ctx, job.ID, applicant.ID)
	if !errors.Is(err, store.ErrHiringPassEmpty) {
		t.Fatalf("expected ErrHiringPassEmpty, got %v", err)
	}

	// Applicant pays ₱10 for hiring pass (uploads proof)
	pass, err = st.AddApplicantHiringPass(ctx, applicant.ID, "gcash", "/uploads/pass_proof1.jpg")
	if err != nil {
		t.Fatalf("AddApplicantHiringPass failed: %v", err)
	}
	if pass.PendingPayment == nil || pass.PendingPayment.Status != "pending" {
		t.Fatalf("expected pending payment on hiring pass, got %+v", pass.PendingPayment)
	}
	if pass.RemainingHires != 0 {
		t.Fatalf("expected 0 remaining hires before verification, got %d", pass.RemainingHires)
	}

	// Attempting AcceptJob now returns ErrHiringPassPending
	err = st.AcceptJob(ctx, job.ID, applicant.ID)
	if !errors.Is(err, store.ErrHiringPassPending) {
		t.Fatalf("expected ErrHiringPassPending, got %v", err)
	}

	// Admin finds applicant payment in admin list
	allPayments, err = st.ListAllPayments(ctx)
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
	pass, err = st.AddApplicantHiringPass(ctx, applicant.ID, "gcash", "/uploads/pass_proof2.jpg")
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

	// Applicant accepts job -> succeeds, but NO hire is spent yet (employer has not confirmed).
	err = st.AcceptJob(ctx, job.ID, applicant.ID)
	if err != nil {
		t.Fatalf("AcceptJob failed after hiring pass verified: %v", err)
	}
	pass, err = st.ApplicantHiringPass(ctx, applicant.ID)
	if err != nil || pass.RemainingHires != 5 {
		t.Fatalf("expected 5 remaining hires before employer confirmation, got %d", pass.RemainingHires)
	}

	// Employer confirms the booking -> the hire is spent now.
	if err := st.ConfirmJob(ctx, employer.ID, job.ID); err != nil {
		t.Fatalf("ConfirmJob failed: %v", err)
	}
	pass, err = st.ApplicantHiringPass(ctx, applicant.ID)
	if err != nil || pass.RemainingHires != 4 || pass.UsedHires != 1 {
		t.Fatalf("expected 4 remaining / 1 used after confirmation, got %+v", pass)
	}
}

func TestNoShowEscalationAndPenalties(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "noshow_test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	employer := &store.User{
		Email:        "emp@noshow.test",
		PasswordHash: "pass",
		Role:         "employer",
		Name:         "Emp Bob",
		Company:      "Bob Cafe",
	}
	if err := st.CreateUser(ctx, employer); err != nil {
		t.Fatalf("create employer: %v", err)
	}
	applicant := &store.User{
		Email:        "app@noshow.test",
		PasswordHash: "pass",
		Role:         "applicant",
		Name:         "App Alice",
	}
	if err := st.CreateUser(ctx, applicant); err != nil {
		t.Fatalf("create applicant: %v", err)
	}
	isAct := true
	if err := st.SaveApplicantProfile(ctx, applicant.ID, store.ApplicantProfile{
		EmploymentType: "part-time",
		ExpectedRate:   150,
		RatePeriod:     "hour",
		Availability:   "now",
		City:           "Cebu City",
		Active:         &isAct,
		Roles:          []string{"kitchen-helper"},
	}); err != nil {
		t.Fatalf("save applicant profile: %v", err)
	}

	initRaw, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	if _, err := initRaw.Exec("UPDATE applicant_hiring_passes SET total_hires=20, remaining_hires=20, used_hires=0 WHERE applicant_id=?", applicant.ID); err != nil {
		initRaw.Close()
		t.Fatalf("grant test hires: %v", err)
	}
	initRaw.Close()

	createJobAndAssign := func() int64 {
		j := &store.Job{
			EmployerID:     employer.ID,
			Category:       "kitchen",
			RoleID:         "kitchen-helper",
			RoleLabel:      "Kitchen Helper",
			EmploymentType: "part-time",
			RateCentavos:   15000,
			RatePeriod:     "hour",
			City:           "Cebu City",
			MapLocation:    "Bob Cafe",
			Timing:         "now",
			CrewNeeded:     1,
		}
		if err := st.CreateJob(ctx, j); err != nil {
			t.Fatalf("create job: %v", err)
		}
		raw, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)")
		if err != nil {
			t.Fatalf("open raw db: %v", err)
		}
		if _, err := raw.Exec("UPDATE jobs SET status='matching' WHERE id=?", j.ID); err != nil {
			raw.Close()
			t.Fatalf("set matching: %v", err)
		}
		raw.Close()
		if err := st.AcceptJob(ctx, j.ID, applicant.ID); err != nil {
			t.Fatalf("accept job: %v", err)
		}
		if err := st.ConfirmJob(ctx, employer.ID, j.ID); err != nil {
			t.Fatalf("confirm job: %v", err)
		}
		return j.ID
	}

	// Strike 1
	j1 := createJobAndAssign()
	count, isTemp, isPerm, err := st.RecordNoShow(ctx, j1, applicant.ID)
	if err != nil || count != 1 || isTemp || isPerm {
		t.Fatalf("strike 1 failed: count=%d temp=%v perm=%v err=%v", count, isTemp, isPerm, err)
	}
	p, _ := st.ApplicantProfile(ctx, applicant.ID)
	if p.Rating >= 4.90 {
		t.Fatalf("expected rating deduction, got %f", p.Rating)
	}

	// Strike 2
	j2 := createJobAndAssign()
	count, isTemp, isPerm, err = st.RecordNoShow(ctx, j2, applicant.ID)
	if err != nil || count != 2 || isTemp || isPerm {
		t.Fatalf("strike 2 failed: count=%d temp=%v perm=%v err=%v", count, isTemp, isPerm, err)
	}

	// Strike 3
	j3 := createJobAndAssign()
	count, isTemp, isPerm, err = st.RecordNoShow(ctx, j3, applicant.ID)
	if err != nil || count != 3 || isTemp || isPerm {
		t.Fatalf("strike 3 failed: count=%d temp=%v perm=%v err=%v", count, isTemp, isPerm, err)
	}

	// Strike 4 (>3 times) -> Should trigger 5-day temporary deactivation!
	j4 := createJobAndAssign()
	count, isTemp, isPerm, err = st.RecordNoShow(ctx, j4, applicant.ID)
	if err != nil || count != 4 || !isTemp || isPerm {
		t.Fatalf("strike 4 failed: count=%d temp=%v perm=%v err=%v", count, isTemp, isPerm, err)
	}
	p, _ = st.ApplicantProfile(ctx, applicant.ID)
	if p.DeactivatedUntil <= time.Now().Unix() || *p.Active {
		t.Fatalf("expected temporary deactivation active=false, got active=%v deactUntil=%d", *p.Active, p.DeactivatedUntil)
	}
	if !p.ServedTemporaryDeactivation {
		t.Fatalf("expected ServedTemporaryDeactivation=true")
	}

	// Simulate serving the deactivation (simulate deactivation expired in the past)
	raw, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	pastTime := time.Now().Unix() - 100
	if _, err := raw.Exec("UPDATE applicant_profiles SET deactivated_until=? WHERE user_id=?", pastTime, applicant.ID); err != nil {
		raw.Close()
		t.Fatalf("update past deactivation: %v", err)
	}
	raw.Close()

	// Fetch profile -> should auto-reactivate because 5 days completed!
	p, _ = st.ApplicantProfile(ctx, applicant.ID)
	if !*p.Active || p.DeactivatedUntil != 0 {
		t.Fatalf("expected auto-reactivation after serving deactivation, got active=%v deactUntil=%d", *p.Active, p.DeactivatedUntil)
	}
	if !p.ServedTemporaryDeactivation {
		t.Fatalf("expected ServedTemporaryDeactivation to remain true after suspension")
	}

	// Next no-show after serving deactivation -> MUST LEAD TO PERMANENT DEACTIVATION!
	j5 := createJobAndAssign()
	count, isTemp, isPerm, err = st.RecordNoShow(ctx, j5, applicant.ID)
	if err != nil || count != 5 || isTemp || !isPerm {
		t.Fatalf("expected permanent deactivation: count=%d temp=%v perm=%v err=%v", count, isTemp, isPerm, err)
	}
	p, _ = st.ApplicantProfile(ctx, applicant.ID)
	if !p.IsPermanentlyDeactivated || *p.Active {
		t.Fatalf("expected IsPermanentlyDeactivated=true active=false, got perm=%v active=%v", p.IsPermanentlyDeactivated, *p.Active)
	}
}
