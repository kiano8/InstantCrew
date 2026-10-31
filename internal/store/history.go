package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// InitialFreeHires is how many hires a new applicant gets before paying.
const InitialFreeHires = 0

// grantInitialHires creates the applicant's empty hiring pass. Paid hires are
// added only after accounting verifies the signup payment.
// ON CONFLICT DO NOTHING keeps it idempotent: an existing pass is never reset.
func (s *Store) grantInitialHires(ctx context.Context, applicantID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO applicant_hiring_passes
			(applicant_id, total_hires, remaining_hires, used_hires, amount_centavos, batches_paid, updated_at)
		VALUES ($1, $2, $3, 0, 0, 0, $4)
		ON CONFLICT (applicant_id) DO NOTHING`,
		applicantID, InitialFreeHires, InitialFreeHires, time.Now().Unix())
	return err
}

// afterUserCreated runs for every signup path. If the hiring-pass row cannot
// be created, remove the half-created account.
func (s *Store) afterUserCreated(ctx context.Context, u *User) error {
	if u.Role != "applicant" {
		return nil
	}
	if err := s.grantInitialHires(ctx, u.ID); err != nil {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, u.ID)
		return err
	}
	return nil
}

// CreateUser creates a user. Applicants start with InitialFreeHires hires.
func (s *Store) CreateUser(ctx context.Context, u *User) error {
	if err := s.createUserRow(ctx, u); err != nil {
		return err
	}
	return s.afterUserCreated(ctx, u)
}

// EmailInOtherRole reports whether email already has an employer or applicant
// account under a role other than role.
func (s *Store) EmailInOtherRole(ctx context.Context, email, role string) (bool, error) {
	var taken bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE email = $1 AND role <> $2 AND role IN ('employer', 'applicant'))`,
		email, role).Scan(&taken)
	return taken, err
}

// CreateUserCapped is CreateUser for signups: the beta cap on distinct emails,
// and an email can only hold one of the employer and applicant roles.
// shortcut: check then insert, so two simultaneous signups could still get both roles; upgrade to a unique index once existing dual-role emails are resolved.
func (s *Store) CreateUserCapped(ctx context.Context, u *User, max int) error {
	taken, err := s.EmailInOtherRole(ctx, u.Email, u.Role)
	if err != nil {
		return err
	}
	if taken {
		return ErrEmailTaken
	}
	if err := s.createUserCappedRow(ctx, u, max); err != nil {
		return err
	}
	return s.afterUserCreated(ctx, u)
}

// PaymentLogEntry is one line of a user's payment history / activity log.
type PaymentLogEntry struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"` // search_fee | hiring_pass | free_hires | hire_used
	Title           string `json:"title"`
	Description     string `json:"description"`
	AmountCentavos  int64  `json:"amount_centavos"`
	AmountPesos     string `json:"amount_pesos"`
	Method          string `json:"method,omitempty"`
	Status          string `json:"status"` // pending | verified | rejected | granted | used
	ProofImage      string `json:"proof_image,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	Ref             string `json:"ref,omitempty"`
	JobID           *int64 `json:"job_id,omitempty"`
	CreatedAt       int64  `json:"created_at"`
	VerifiedAt      *int64 `json:"verified_at,omitempty"`
}

func pesos(centavos int64) string { return fmt.Sprintf("₱%.2f", float64(centavos)/100) }

// hiresText is "1 hire" or "n hires".
func hiresText(n int64) string {
	if n == 1 {
		return "1 hire"
	}
	return fmt.Sprintf("%d hires", n)
}

// EmployerPaymentHistory lists every search fee the employer has paid or submitted, newest first.
func (s *Store) EmployerPaymentHistory(ctx context.Context, employerID int64) ([]PaymentLogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.job_id, p.amount_centavos, p.method,
			COALESCE(p.proof_image,''), COALESCE(p.status,''), p.verified_at,
			COALESCE(p.rejection_reason,''), p.provider_ref, p.created_at,
			COALESCE(j.role_label,'')
		FROM payments p LEFT JOIN jobs j ON j.id = p.job_id
		WHERE ($1 <= 0 OR p.user_id = $2) AND COALESCE(p.status, '') != 'pending'
		ORDER BY p.created_at DESC, p.id DESC`, employerID, employerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PaymentLogEntry, 0)
	for rows.Next() {
		var (
			id, jobID, amount, created             int64
			method, proof, status, reason, ref, rl string
			verifiedAt                             sql.NullInt64
		)
		if err := rows.Scan(&id, &jobID, &amount, &method, &proof, &status, &verifiedAt,
			&reason, &ref, &created, &rl); err != nil {
			return nil, err
		}
		e := PaymentLogEntry{
			ID: fmt.Sprintf("emp_%d", id), Kind: "search_fee",
			Title:          "Crew search fee",
			AmountCentavos: amount, AmountPesos: pesos(amount),
			Method: method, ProofImage: proof, RejectionReason: reason, Ref: ref,
			CreatedAt: created,
		}
		jid := jobID
		e.JobID = &jid
		e.Description = rl
		switch {
		case verifiedAt.Valid && verifiedAt.Int64 > 0:
			v := verifiedAt.Int64
			e.VerifiedAt = &v
			e.Status = "verified"
		case status == "rejected":
			e.Status = "rejected"
		default:
			e.Status = "pending"
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ApplicantPaymentHistory is the applicant's log: hiring-pass payments and
// each hire used on a shift. Newest first.
func (s *Store) ApplicantPaymentHistory(ctx context.Context, applicantID int64) ([]PaymentLogEntry, error) {
	out := make([]PaymentLogEntry, 0)

	// Include a starter entry only when free starter hires are enabled.
	var created int64
	var hasPass int
	err := s.db.QueryRowContext(ctx, `SELECT u.created_at, (SELECT COUNT(*) FROM applicant_hiring_passes WHERE applicant_id = u.id)
		FROM users u WHERE u.id = $1`, applicantID).Scan(&created, &hasPass)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && hasPass > 0 && InitialFreeHires > 0 {
		out = append(out, PaymentLogEntry{
			ID: "free_start", Kind: "free_hires", Title: "Starter hires",
			Description: fmt.Sprintf("%d free hires included with your account", InitialFreeHires),
			AmountPesos: pesos(0), Status: "granted", CreatedAt: created,
		})
	}

	// Paid top-ups.
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, amount_centavos, method, COALESCE(proof_image,''), COALESCE(status,''),
			verified_at, COALESCE(rejection_reason,''), COALESCE(credits_granted,0), created_at
		FROM applicant_hiring_pass_payments WHERE applicant_id = $1`, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, amount, createdAt int64
		var credits int
		var method, proof, status, reason string
		var verifiedAt sql.NullInt64
		if err := rows.Scan(&id, &amount, &method, &proof, &status, &verifiedAt, &reason, &credits, &createdAt); err != nil {
			return nil, err
		}
		e := PaymentLogEntry{
			ID: fmt.Sprintf("app_%d", id), Kind: "hiring_pass", Title: "Hiring pass top-up",
			Description:    hiresText(amount / HirePriceCentavos),
			AmountCentavos: amount, AmountPesos: pesos(amount),
			Method: method, ProofImage: proof, RejectionReason: reason,
			Ref: fmt.Sprintf("PASS-%d", id), CreatedAt: createdAt,
		}
		switch {
		case verifiedAt.Valid && verifiedAt.Int64 > 0:
			v := verifiedAt.Int64
			e.VerifiedAt = &v
			e.Status = "verified"
		case status == "rejected":
			e.Status = "rejected"
		case credits > 0: // legacy rows that credited hires before verification existed
			e.Status = "verified"
		default:
			e.Status = "pending"
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	// Hires used: one per shift the employer confirmed (that is when a hire is spent),
	// except shifts whose hire was returned.
	used, err := s.db.QueryContext(ctx, `
		SELECT a.job_id, a.accepted_at, COALESCE(j.role_label,'')
		FROM job_assignments a JOIN jobs j ON j.id = a.job_id
		WHERE a.applicant_id = $1 AND j.status IN ('active','completed') AND NOT a.hire_refunded`, applicantID)
	if err != nil {
		return nil, err
	}
	defer used.Close()
	for used.Next() {
		var jobID, at int64
		var role string
		if err := used.Scan(&jobID, &at, &role); err != nil {
			return nil, err
		}
		jid := jobID
		desc := fmt.Sprintf("Accepted job #%d", jobID)
		if role != "" {
			desc = fmt.Sprintf("Accepted %s (job #%d)", role, jobID)
		}
		out = append(out, PaymentLogEntry{
			ID: fmt.Sprintf("use_%d", jobID), Kind: "hire_used", Title: "1 hire used",
			Description: desc, AmountPesos: pesos(0), Status: "used", JobID: &jid, CreatedAt: at,
		})
	}
	if err := used.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}
