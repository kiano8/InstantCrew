package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AdminPaymentRecord struct {
	ID             string `json:"id"`
	RawID          int64  `json:"raw_id"`
	Type           string `json:"type"`       // "employer" or "applicant"
	TypeLabel      string `json:"type_label"` // "Employer Search Fee" or "Applicant Hiring Pass"
	UserID         int64  `json:"user_id"`
	UserName       string `json:"user_name"`
	UserEmail      string `json:"user_email"`
	UserRole       string `json:"user_role"`
	Company        string `json:"company"`
	AmountCentavos int64  `json:"amount_centavos"`
	AmountPesos    string `json:"amount_pesos"`
	Method         string `json:"method"`
	Details        string `json:"details"`
	ProofImage     string `json:"proof_image"`
	Status         string `json:"status"` // "paid" (or "pending"), "verified"
	IsVerified     bool   `json:"is_verified"`
	VerifiedAt     *int64 `json:"verified_at"`
	VerifiedBy     string `json:"verified_by"`
	CreatedAt      int64  `json:"created_at"`
	Ref            string `json:"ref"`
	JobID          *int64 `json:"job_id,omitempty"`
	JobRole        string `json:"job_role,omitempty"`
}

func (s *Store) ListAllPayments(ctx context.Context) ([]AdminPaymentRecord, error) {
	var records []AdminPaymentRecord

	// 1. Fetch employer payments
	empRows, err := s.db.QueryContext(ctx, `
		SELECT 
			p.id, p.job_id, p.user_id, p.amount_centavos, p.method, 
			COALESCE(p.details, ''), COALESCE(p.proof_image, ''), 
			COALESCE(p.status, 'paid'), p.verified_at, COALESCE(p.verified_by, ''),
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
			rawID, jobID, userID, amountCentavos, createdAt int64
			method, details, proofImage, status, verifiedBy, ref string
			userName, userEmail, company, jobRole               string
			verifiedAt                                         sql.NullInt64
		)
		if err := empRows.Scan(
			&rawID, &jobID, &userID, &amountCentavos, &method,
			&details, &proofImage, &status, &verifiedAt, &verifiedBy,
			&ref, &createdAt,
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
		} else if displayStatus == "" {
			displayStatus = "pending"
		}

		jid := jobID
		records = append(records, AdminPaymentRecord{
			ID:             fmt.Sprintf("emp_%d", rawID),
			RawID:          rawID,
			Type:           "employer",
			TypeLabel:      "Employer Search Fee",
			UserID:         userID,
			UserName:       userName,
			UserEmail:      userEmail,
			UserRole:       "employer",
			Company:        company,
			AmountCentavos: amountCentavos,
			AmountPesos:    fmt.Sprintf("₱%.2f", float64(amountCentavos)/100),
			Method:         method,
			Details:        details,
			ProofImage:     proofImage,
			Status:         displayStatus,
			IsVerified:     isVerified,
			VerifiedAt:     vAt,
			VerifiedBy:     verifiedBy,
			CreatedAt:      createdAt,
			Ref:            ref,
			JobID:          &jid,
			JobRole:        jobRole,
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
			ap.verified_at, COALESCE(ap.verified_by, ''), ap.created_at,
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
			rawID, applicantID, amountCentavos, createdAt int64
			method, proofImage, status, verifiedBy       string
			userName, userEmail                           string
			verifiedAt                                    sql.NullInt64
		)
		if err := appRows.Scan(
			&rawID, &applicantID, &amountCentavos, &method,
			&proofImage, &status, &verifiedAt, &verifiedBy, &createdAt,
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
		} else if displayStatus == "" {
			displayStatus = "pending"
		}

		records = append(records, AdminPaymentRecord{
			ID:             fmt.Sprintf("app_%d", rawID),
			RawID:          rawID,
			Type:           "applicant",
			TypeLabel:      "Applicant Hiring Pass (5 Hires)",
			UserID:         applicantID,
			UserName:       userName,
			UserEmail:      userEmail,
			UserRole:       "applicant",
			Company:        "",
			AmountCentavos: amountCentavos,
			AmountPesos:    fmt.Sprintf("₱%.2f", float64(amountCentavos)/100),
			Method:         method,
			Details:        "Hiring Pass Batch (5 hires)",
			ProofImage:     proofImage,
			Status:         displayStatus,
			IsVerified:     isVerified,
			VerifiedAt:     vAt,
			VerifiedBy:     verifiedBy,
			CreatedAt:      createdAt,
			Ref:            fmt.Sprintf("PASS-%d", rawID),
		})
	}
	if err := appRows.Err(); err != nil {
		return nil, err
	}

	// Sort combined records by CreatedAt descending
	for i := 0; i < len(records); i++ {
		for j := i + 1; j < len(records); j++ {
			if records[i].CreatedAt < records[j].CreatedAt {
				records[i], records[j] = records[j], records[i]
			}
		}
	}

	return records, nil
}

func (s *Store) SetPaymentStatus(ctx context.Context, pType string, id int64, status string, actor string) error {
	now := time.Now().Unix()
	normStatus := strings.ToLower(strings.TrimSpace(status))
	if normStatus == "" {
		normStatus = "pending"
	}

	var (
		vAt *int64
		vBy string
	)
	if normStatus == "verified" {
		vAt = &now
		if actor == "" {
			vBy = "Admin / Accounting"
		} else {
			vBy = actor
		}
	} else if normStatus == "rejected" {
		vAt = nil
		if actor == "" {
			vBy = "Rejected by Accounting"
		} else {
			vBy = "Rejected by " + actor
		}
	} else {
		normStatus = "pending"
		vAt = nil
		vBy = ""
	}

	switch strings.ToLower(pType) {
	case "employer":
		var err error
		if vAt != nil {
			_, err = s.db.ExecContext(ctx, `UPDATE payments SET status=?, verified_at=?, verified_by=? WHERE id=?`, normStatus, *vAt, vBy, id)
		} else {
			_, err = s.db.ExecContext(ctx, `UPDATE payments SET status=?, verified_at=NULL, verified_by=? WHERE id=?`, normStatus, vBy, id)
		}
		return err

	case "applicant":
		var err error
		if vAt != nil {
			_, err = s.db.ExecContext(ctx, `UPDATE applicant_hiring_pass_payments SET status=?, verified_at=?, verified_by=? WHERE id=?`, normStatus, *vAt, vBy, id)
		} else {
			_, err = s.db.ExecContext(ctx, `UPDATE applicant_hiring_pass_payments SET status=?, verified_at=NULL, verified_by=? WHERE id=?`, normStatus, vBy, id)
		}
		return err

	default:
		return fmt.Errorf("unknown payment type: %s", pType)
	}
}

func (s *Store) VerifyPayment(ctx context.Context, pType string, id int64, verified bool, verifiedBy string) error {
	status := "pending"
	if verified {
		status = "verified"
	}
	return s.SetPaymentStatus(ctx, pType, id, status, verifiedBy)
}

// EnsureAdminUser guarantees that an admin user exists in the database with the provided credentials.
func (s *Store) EnsureAdminUser(ctx context.Context, email, password, name string) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().Unix()

	var id int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ? AND role = 'admin'`, normEmail).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			_, insertErr := s.db.ExecContext(ctx, `
				INSERT INTO users (email, password_hash, role, name, company, created_at)
				VALUES (?, ?, 'admin', ?, 'Instant Crew Accounting', ?)
			`, normEmail, string(hash), name, now)
			return insertErr
		}
		return err
	}

	// Update existing admin password and name to guarantee validity
	_, updateErr := s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = ?, name = ? WHERE id = ?
	`, string(hash), name, id)
	return updateErr
}
