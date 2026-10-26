package store

import (
	"context"
	"database/sql"
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
			TypeLabel:       "Applicant Hiring Pass (5 Hires)",
			UserID:          applicantID,
			UserName:        userName,
			UserEmail:       userEmail,
			UserRole:        "applicant",
			Company:         "",
			AmountCentavos:  amountCentavos,
			AmountPesos:     fmt.Sprintf("₱%.2f", float64(amountCentavos)/100),
			Method:          method,
			Details:         "Hiring Pass Batch (5 hires)",
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

	var userID int64
	var prevStatus string
	var creditsGranted int
	switch strings.ToLower(pType) {
	case "employer":
		err = tx.QueryRowContext(ctx, `SELECT user_id, COALESCE(status,'') FROM payments WHERE id=$1`, id).Scan(&userID, &prevStatus)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if normStatus == "verified" {
			res, err := tx.ExecContext(ctx, `UPDATE payments SET status=$1, verified_at=$2, verified_by=$3, rejection_reason='' WHERE id=$4`, normStatus, now, vBy, id)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return ErrNotFound
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
		err = tx.QueryRowContext(ctx, `SELECT applicant_id, COALESCE(status,''), COALESCE(credits_granted,0) FROM applicant_hiring_pass_payments WHERE id=$1`, id).Scan(&userID, &prevStatus, &creditsGranted)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
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
					VALUES($1,5,5,0,1000,1,$2)
					ON CONFLICT(applicant_id) DO UPDATE SET
						total_hires=applicant_hiring_passes.total_hires+5, remaining_hires=applicant_hiring_passes.remaining_hires+5,
						amount_centavos=applicant_hiring_passes.amount_centavos+1000, batches_paid=applicant_hiring_passes.batches_paid+1, updated_at=excluded.updated_at`,
					userID, now); err != nil {
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
			title, message = "Search fee verified", "Accounting verified your receipt. You can now start crew search."
		} else {
			title, message = "Hiring pass verified", "Accounting verified your ₱10 receipt. Five hiring opportunities are now active."
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
		if _, err = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at) VALUES($1,$2,$3,$4,$5,$6)`,
			userID, title, message, kind, "{}", now); err != nil {
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
