package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AdminPaymentRecord struct {
	ID              string `json:"id"`
	RawID           int64  `json:"raw_id"`
	Type            string `json:"type"`       // "employer" or "applicant"
	TypeLabel       string `json:"type_label"` // "Employer Search Fee" or "Applicant Hiring Pass"
	UserID          int64  `json:"user_id"`
	UserName        string `json:"user_name"`
	UserEmail       string `json:"user_email"`
	UserRole        string `json:"user_role"`
	Company         string `json:"company"`
	AmountCentavos  int64  `json:"amount_centavos"`
	AmountPesos     string `json:"amount_pesos"`
	Method          string `json:"method"`
	Details         string `json:"details"`
	ProofImage      string `json:"proof_image"`
	Status          string `json:"status"` // "pending", "verified", or "rejected"
	IsVerified      bool   `json:"is_verified"`
	VerifiedAt      *int64 `json:"verified_at"`
	VerifiedBy      string `json:"verified_by"`
	RejectionReason string `json:"rejection_reason"`
	CreatedAt       int64  `json:"created_at"`
	Ref             string `json:"ref"`
	JobID           *int64 `json:"job_id,omitempty"`
	JobRole         string `json:"job_role,omitempty"`
}

func (s *Store) ListAllPayments(ctx context.Context) ([]AdminPaymentRecord, error) {
	records := make([]AdminPaymentRecord, 0)

	// 1. Fetch employer payments
	empRows, err := s.db.QueryContext(ctx, `
		SELECT 
			p.id, p.job_id, p.user_id, p.amount_centavos, p.method, 
			COALESCE(p.details, ''), COALESCE(p.proof_image, ''), 
			COALESCE(p.status, 'paid'), p.verified_at, COALESCE(p.verified_by, ''),
			COALESCE(p.rejection_reason, ''),
			p.provider_ref, p.created_at,
			COALESCE(u.name, ''), COALESCE(u.email, ''), COALESCE(u.company, ''),
			COALESCE(j.role_label, '')
		FROM payments p
		LEFT JOIN users u ON u.id = p.user_id
		LEFT JOIN jobs j ON j.id = p.job_id
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list employer payments: %w", err)
	}
	defer empRows.Close()

	for empRows.Next() {
		var (
			rawID, jobID, userID, amountCentavos, createdAt                    int64
			method, details, proofImage, status, verifiedBy, rejectReason, ref string
			userName, userEmail, company, jobRole                              string
			verifiedAt                                                         sql.NullInt64
		)
		if err := empRows.Scan(
			&rawID, &jobID, &userID, &amountCentavos, &method,
			&details, &proofImage, &status, &verifiedAt, &verifiedBy,
			&rejectReason, &ref, &createdAt,
			&userName, &userEmail, &company, &jobRole,
		); err != nil {
			return nil, err
		}

		isVerified := verifiedAt.Valid && verifiedAt.Int64 > 0
		var vAt *int64
		if isVerified {
			v := verifiedAt.Int64
			vAt = &v
		}

		displayStatus := status
		if isVerified {
			displayStatus = "verified"
		} else if displayStatus == "paid" || displayStatus == "" {
			displayStatus = "pending"
		}

		jid := jobID
		records = append(records, AdminPaymentRecord{
			ID:              fmt.Sprintf("emp_%d", rawID),
			RawID:           rawID,
			Type:            "employer",
			TypeLabel:       "Employer Search Fee",
			UserID:          userID,
			UserName:        userName,
			UserEmail:       userEmail,
			UserRole:        "employer",
			Company:         company,
			AmountCentavos:  amountCentavos,
			AmountPesos:     fmt.Sprintf("₱%.2f", float64(amountCentavos)/100),
			Method:          method,
			Details:         details,
			ProofImage:      proofImage,
			Status:          displayStatus,
			IsVerified:      isVerified,
			VerifiedAt:      vAt,
			VerifiedBy:      verifiedBy,
			RejectionReason: rejectReason,
			CreatedAt:       createdAt,
			Ref:             ref,
			JobID:           &jid,
			JobRole:         jobRole,
		})
	}
	if err := empRows.Err(); err != nil {
		return nil, err
	}

	// 2. Fetch applicant hiring pass payments
	appRows, err := s.db.QueryContext(ctx, `
		SELECT 
			ap.id, ap.applicant_id, ap.amount_centavos, ap.method,
			COALESCE(ap.proof_image, ''), COALESCE(ap.status, 'paid'),
			ap.verified_at, COALESCE(ap.verified_by, ''),
			COALESCE(ap.rejection_reason, ''), ap.created_at,
			COALESCE(u.name, ''), COALESCE(u.email, '')
		FROM applicant_hiring_pass_payments ap
		LEFT JOIN users u ON u.id = ap.applicant_id
		ORDER BY ap.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list applicant payments: %w", err)
	}
	defer appRows.Close()

	for appRows.Next() {
		var (
			rawID, applicantID, amountCentavos, createdAt        int64
			method, proofImage, status, verifiedBy, rejectReason string
			userName, userEmail                                  string
			verifiedAt                                           sql.NullInt64
		)
		if err := appRows.Scan(
			&rawID, &applicantID, &amountCentavos, &method,
			&proofImage, &status, &verifiedAt, &verifiedBy, &rejectReason, &createdAt,
			&userName, &userEmail,
		); err != nil {
			return nil, err
		}

		isVerified := verifiedAt.Valid && verifiedAt.Int64 > 0
		var vAt *int64
		if isVerified {
			v := verifiedAt.Int64
			vAt = &v
		}

		displayStatus := status
		if isVerified {
			displayStatus = "verified"
		} else if displayStatus == "paid" || displayStatus == "" {
			displayStatus = "pending"
		}

		records = append(records, AdminPaymentRecord{
			ID:              fmt.Sprintf("app_%d", rawID),
			RawID:           rawID,
			Type:            "applicant",
			TypeLabel:       "Applicant Hiring Pass",
			UserID:          applicantID,
			UserName:        userName,
			UserEmail:       userEmail,
			UserRole:        "applicant",
			Company:         "",
			AmountCentavos:  amountCentavos,
			AmountPesos:     fmt.Sprintf("₱%.2f", float64(amountCentavos)/100),
			Method:          method,
			Details:         hiresText(amountCentavos/HirePriceCentavos),
			ProofImage:      proofImage,
			Status:          displayStatus,
			IsVerified:      isVerified,
			VerifiedAt:      vAt,
			VerifiedBy:      verifiedBy,
			RejectionReason: rejectReason,
			CreatedAt:       createdAt,
			Ref:             fmt.Sprintf("PASS-%d", rawID),
		})
	}
	if err := appRows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})

	return records, nil
}

// SetPaymentStatus lets an admin verify or reject a receipt. Rejecting stores an
// optional reason that is shown back to the payer and in the admin portal.
func (s *Store) SetPaymentStatus(ctx context.Context, pType string, id int64, status string, actor string, reason string) error {
	now := time.Now().Unix()
	normStatus := strings.ToLower(strings.TrimSpace(status))
	switch normStatus {
	case "verified", "rejected", "pending":
	default:
		normStatus = "pending"
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 500 {
		reason = reason[:500]
	}
	if normStatus != "rejected" {
		reason = ""
	}

	vBy := ""
	switch normStatus {
	case "verified":
		if actor == "" {
			vBy = "Admin / Accounting"
		} else {
			vBy = actor
		}
	case "rejected":
		if actor == "" {
			vBy = "Rejected by Accounting"
		} else {
			vBy = "Rejected by " + actor
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// FOR UPDATE below makes a second concurrent verify wait and then see
	// credits_granted=1, so hires and notifications are granted only once.
	var userID, penalty int64
	var jobID sql.NullInt64
	var prevStatus string
	var creditsGranted int
	var passAmount int64 // a hiring pass receipt buys passAmount / HirePriceCentavos hires
	confirmed := false
	switch strings.ToLower(pType) {
	case "employer":
		err = tx.QueryRowContext(ctx, `SELECT user_id, job_id, COALESCE(status,''), penalty_centavos FROM payments WHERE id=$1 FOR UPDATE`, id).Scan(&userID, &jobID, &prevStatus, &penalty)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if prevStatus == "verified" && normStatus != "verified" {
			return ErrAlreadyVerified
		}
		if normStatus == "verified" {
			res, err := tx.ExecContext(ctx, `UPDATE payments SET status=$1, verified_at=$2, verified_by=$3, rejection_reason='' WHERE id=$4`, normStatus, now, vBy, id)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return ErrNotFound
			}
			if prevStatus != "verified" && jobID.Valid {
				// Penalties added after this receipt was priced stay owed.
				if _, err := tx.ExecContext(ctx, `UPDATE users SET search_penalty_centavos=GREATEST(0, search_penalty_centavos-$1) WHERE id=$2`, penalty, userID); err != nil {
					return err
				}
				// The fee is what the waiting booking needed: confirm it now.
				switch err := confirmJob(ctx, tx, userID, jobID.Int64, now); {
				case err == nil:
					confirmed = true
				case !errors.Is(err, ErrBadState) && !errors.Is(err, ErrPaymentNotVerified) &&
					!errors.Is(err, ErrPaymentRejected) && !errors.Is(err, ErrNotFound):
					return err
				}
			}
		} else {
			res, err := tx.ExecContext(ctx, `UPDATE payments SET status=$1, verified_at=NULL, verified_by=$2, rejection_reason=$3 WHERE id=$4`, normStatus, vBy, reason, id)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return ErrNotFound
			}
		}
	case "applicant":
		err = tx.QueryRowContext(ctx, `SELECT applicant_id, COALESCE(status,''), COALESCE(credits_granted,0), amount_centavos FROM applicant_hiring_pass_payments WHERE id=$1 FOR UPDATE`, id).Scan(&userID, &prevStatus, &creditsGranted, &passAmount)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		// Hires granted on verify are not taken back, so a verified receipt stays verified.
		if prevStatus == "verified" && normStatus != "verified" {
			return ErrAlreadyVerified
		}
		if normStatus == "verified" {
			res, err := tx.ExecContext(ctx, `UPDATE applicant_hiring_pass_payments SET status=$1, verified_at=$2, verified_by=$3, rejection_reason='' WHERE id=$4`, normStatus, now, vBy, id)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return ErrNotFound
			}
			if creditsGranted == 0 && prevStatus != "verified" {
				if _, err = tx.ExecContext(ctx, `
					INSERT INTO applicant_hiring_passes(applicant_id,total_hires,remaining_hires,used_hires,amount_centavos,batches_paid,updated_at)
					VALUES($1,$3,$3,0,$4,1,$2)
					ON CONFLICT(applicant_id) DO UPDATE SET
						total_hires=applicant_hiring_passes.total_hires+$3, remaining_hires=applicant_hiring_passes.remaining_hires+$3,
						amount_centavos=applicant_hiring_passes.amount_centavos+$4, batches_paid=applicant_hiring_passes.batches_paid+1, updated_at=excluded.updated_at`,
					userID, now, passAmount/HirePriceCentavos, passAmount); err != nil {
					return err
				}
				if _, err = tx.ExecContext(ctx, `UPDATE applicant_hiring_pass_payments SET credits_granted=1 WHERE id=$1`, id); err != nil {
					return err
				}
			}
		} else {
			res, err := tx.ExecContext(ctx, `UPDATE applicant_hiring_pass_payments SET status=$1, verified_at=NULL, verified_by=$2, rejection_reason=$3 WHERE id=$4`, normStatus, vBy, reason, id)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return ErrNotFound
			}
		}
	default:
		return fmt.Errorf("unknown payment type: %s", pType)
	}

	title, message, kind := "", "", "payment"
	switch normStatus {
	case "verified":
		if strings.ToLower(pType) == "employer" {
			title, message = "Search fee verified", "Accounting verified your receipt."
			if confirmed {
				message += " Your crew booking is confirmed."
			}
		} else {
			title, message = "Hiring pass verified", fmt.Sprintf("Accounting verified your %s receipt. %s now active.", pesos(passAmount), hiresText(passAmount/HirePriceCentavos))
		}
	case "rejected":
		if strings.ToLower(pType) == "employer" {
			title, message = "Search fee rejected", "Accounting could not verify your receipt. Please upload a new proof of payment."
		} else {
			title, message = "Hiring pass rejected", "Accounting could not verify your receipt. Please upload a new proof of payment."
		}
		if reason != "" {
			message += " Reason: " + reason
		}
	}
	if title != "" && prevStatus != normStatus {
		// The job_id lets the employer's notification open that job's search.
		if _, err = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
			VALUES($1,$2,$3,$4,json_build_object('payment_type',$5::text,'payment_id',$6::bigint,'job_id',$7::bigint)::text,$8)`,
			userID, title, message, kind, strings.ToLower(pType), id, jobID, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) VerifyPayment(ctx context.Context, pType string, id int64, verified bool, verifiedBy string) error {
	status := "pending"
	if verified {
		status = "verified"
	}
	return s.SetPaymentStatus(ctx, pType, id, status, verifiedBy, "")
}

// EnsureAdminUser inserts an admin account when missing. Existing passwords are left unchanged.
func (s *Store) EnsureAdminUser(ctx context.Context, email, password, name string) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" || password == "" {
		return fmt.Errorf("admin email and password are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().Unix()

	var id int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1 AND role = 'admin'`, normEmail).Scan(&id)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	_, insertErr := s.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, role, name, company, created_at)
		VALUES ($1, $2, 'admin', $3, 'Instant Crew Accounting', $4)
	`, normEmail, string(hash), name, now)
	return insertErr
}

// CountUsersByRole returns the total number of registered employers and applicants.
func (s *Store) CountUsersByRole(ctx context.Context) (employers int64, applicants int64, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN role = 'employer' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN role = 'applicant' THEN 1 ELSE 0 END), 0)
		FROM users
	`).Scan(&employers, &applicants)
	if err != nil {
		return 0, 0, fmt.Errorf("count users by role: %w", err)
	}
	return employers, applicants, nil
}
