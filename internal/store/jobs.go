package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TransitCancelPenaltyCentavos (₱50) is added to the employer's next search fee
// for each crew member they end while that crew member is on the way.
const TransitCancelPenaltyCentavos = 5000

// PaymentDeadline is how long an employer has, after a crew member accepts, to
// submit the search fee receipt before that crew member is released.
const PaymentDeadline = 3 * time.Minute

var (
	ErrBadState        = errors.New("job is not in a valid state for this action")
	ErrAlreadyAccepted = errors.New("already accepted")
)

type Job struct {
	ID, EmployerID              int64
	Category, RoleID, RoleLabel string
	EmploymentType              string
	RateCentavos                int64
	RatePeriod                  string
	City, MapLocation, MapsURL  string
	Lat, Lng                    *float64
	Timing                      string
	ScheduledAt                 *time.Time
	CrewNeeded, AcceptedCount   int
	Status                      string
	SearchAttempts              int // number of searches started; the current one's fee has search_attempt = SearchAttempts
	CreatedAt, UpdatedAt        time.Time
}

type Assignment struct {
	ApplicantID int64
	Name        string
	AcceptedAt  time.Time
	EndedAt     *time.Time
	Transit     string
	Rating      float64 // the crew member's rating, 1-5
	// ArrivalDeadlineAt is set once the employer confirms; nil before that.
	ArrivalDeadlineAt *time.Time
}

// eligibleCrew excludes applicants who are suspended or permanently deactivated.
const eligibleCrew = `NOT p.is_permanently_deactivated
	AND (p.deactivated_until IS NULL OR p.deactivated_until <= extract(epoch from now())::bigint)`

// MatchedApplicant contains only the profile details employers need to identify
// a crew member while a matching search is open. Contact information is omitted.
type MatchedApplicant struct {
	Name             string  `json:"name"`
	ExpectedRate     int64   `json:"expected_rate"`
	RatePeriod       string  `json:"rate_period"`
	City             string  `json:"city"`
	AvailabilityMode string  `json:"availability_mode"`
	Rating           float64 `json:"rating"`
}

type JobPayment struct {
	ID              int64  `json:"id"`
	Status          string `json:"status"` // pending | verified | rejected
	Method          string `json:"method"`
	ProofImage      string `json:"proof_image,omitempty"`
	SearchAttempt   int    `json:"search_attempt"`
	VerifiedAt      *int64 `json:"verified_at,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

const jobCols = `id, employer_id, category, role_id, role_label, employment_type, rate_centavos,
	rate_period, city, map_location, maps_url, lat, lng, timing, scheduled_at, crew_needed,
	accepted_count, status, search_attempts, created_at, updated_at`

type rowScanner interface{ Scan(dest ...any) error }

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanJob(r rowScanner) (*Job, error) {
	var j Job
	var lat, lng sql.NullFloat64
	var sched sql.NullInt64
	var created, updated int64
	err := r.Scan(&j.ID, &j.EmployerID, &j.Category, &j.RoleID, &j.RoleLabel, &j.EmploymentType,
		&j.RateCentavos, &j.RatePeriod, &j.City, &j.MapLocation, &j.MapsURL, &lat, &lng, &j.Timing,
		&sched, &j.CrewNeeded, &j.AcceptedCount, &j.Status, &j.SearchAttempts, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lat.Valid && lng.Valid {
		j.Lat, j.Lng = &lat.Float64, &lng.Float64
	}
	if sched.Valid {
		t := time.Unix(sched.Int64, 0)
		j.ScheduledAt = &t
	}
	j.CreatedAt, j.UpdatedAt = time.Unix(created, 0), time.Unix(updated, 0)
	return &j, nil
}

// placeholders returns "$start,$start+1,..." with n numbered parameters.
func placeholders(start, n int) string {
	ps := make([]string, n)
	for i := range ps {
		ps[i] = fmt.Sprintf("$%d", start+i)
	}
	return strings.Join(ps, ",")
}

func nullF(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
}

func nullT(p *time.Time) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: p.Unix(), Valid: true}
}

// missOrBadState explains why a guarded UPDATE touched 0 rows.
func missOrBadState(ctx context.Context, q queryer, employerID, id int64) error {
	var one int
	err := q.QueryRowContext(ctx, `SELECT 1 FROM jobs WHERE id = $1 AND employer_id = $2`, id, employerID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return ErrBadState
}

func (s *Store) CreateJob(ctx context.Context, j *Job) error {
	now := time.Now().Unix()
	return s.db.QueryRowContext(ctx, `
		INSERT INTO jobs (employer_id, category, role_id, role_label, employment_type, rate_centavos,
			rate_period, city, map_location, maps_url, lat, lng, timing, scheduled_at, crew_needed,
			status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'draft',$16,$17)
		RETURNING id`,
		j.EmployerID, j.Category, j.RoleID, j.RoleLabel, j.EmploymentType, j.RateCentavos,
		j.RatePeriod, j.City, j.MapLocation, j.MapsURL, nullF(j.Lat), nullF(j.Lng), j.Timing,
		nullT(j.ScheduledAt), j.CrewNeeded, now, now).Scan(&j.ID)
}

// UpdateJob edits details; only allowed before/after a search (draft, unfulfilled).
func (s *Store) UpdateJob(ctx context.Context, j *Job) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE jobs SET category=$1, role_id=$2, role_label=$3, employment_type=$4, rate_centavos=$5,
			rate_period=$6, city=$7, map_location=$8, maps_url=$9, lat=$10, lng=$11, timing=$12, scheduled_at=$13,
			crew_needed=$14, updated_at=$15
		WHERE id=$16 AND employer_id=$17 AND status IN ('draft','unfulfilled')`,
		j.Category, j.RoleID, j.RoleLabel, j.EmploymentType, j.RateCentavos, j.RatePeriod, j.City,
		j.MapLocation, j.MapsURL, nullF(j.Lat), nullF(j.Lng), j.Timing, nullT(j.ScheduledAt),
		j.CrewNeeded, time.Now().Unix(), j.ID, j.EmployerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, s.db, j.EmployerID, j.ID)
	}
	return nil
}

// GetJob is scoped to the employer, so one employer can never read another's job.
// JobOwnerID returns the employer ID of a job, or an error if not found.
func (s *Store) JobOwnerID(ctx context.Context, jobID int64) (int64, error) {
	var empID int64
	err := s.db.QueryRowContext(ctx, `SELECT employer_id FROM jobs WHERE id = $1`, jobID).Scan(&empID)
	return empID, err
}

func (s *Store) GetJob(ctx context.Context, employerID, id int64) (*Job, error) {
	if employerID <= 0 {
		return scanJob(s.db.QueryRowContext(ctx,
			`SELECT `+jobCols+` FROM jobs WHERE id = $1`, id))
	}
	return scanJob(s.db.QueryRowContext(ctx,
		`SELECT `+jobCols+` FROM jobs WHERE id = $1 AND employer_id = $2`, id, employerID))
}

func (s *Store) ListJobs(ctx context.Context, employerID int64, statuses []string, limit, offset int) ([]*Job, error) {
	q := `SELECT ` + jobCols + ` FROM jobs WHERE employer_id = $1`
	args := []any{employerID}
	if len(statuses) > 0 {
		q += ` AND status IN (` + placeholders(len(args)+1, len(statuses)) + `)`
		for _, st := range statuses {
			args = append(args, st)
		}
	}
	q += fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// StartSearch opens a search that stays open until a crew member accepts or the
// employer cancels. The fee is paid later, once a crew member has accepted.
// The status guard makes double-clicks and concurrent requests safe: only one wins.
func (s *Store) StartSearch(ctx context.Context, employerID, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	res, err := tx.ExecContext(ctx, `
		UPDATE jobs SET status='matching', search_attempts = search_attempts + 1, updated_at = $1
		WHERE id = $2 AND employer_id = $3 AND status IN ('draft','unfulfilled')`,
		now.Unix(), id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, tx, employerID, id)
	}
	s.notifyMatchingCrew(ctx, tx, id, now)
	return tx.Commit()
}

func (s *Store) LatestJobPayment(ctx context.Context, jobID int64) (*JobPayment, error) {
	m, err := s.LatestJobPayments(ctx, []int64{jobID})
	return m[jobID], err
}

// LatestJobPayments returns each job's newest payment in one query; jobs without one are absent.
func (s *Store) LatestJobPayments(ctx context.Context, jobIDs []int64) (map[int64]*JobPayment, error) {
	out := map[int64]*JobPayment{}
	if len(jobIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT ON (job_id) job_id, id, COALESCE(status,''), COALESCE(method,''), COALESCE(proof_image,''), search_attempt, verified_at, COALESCE(rejection_reason,'')
		FROM payments WHERE job_id = ANY($1) ORDER BY job_id, search_attempt DESC, id DESC`, jobIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p JobPayment
		var jobID int64
		var verifiedAt sql.NullInt64
		if err := rows.Scan(&jobID, &p.ID, &p.Status, &p.Method, &p.ProofImage, &p.SearchAttempt, &verifiedAt, &p.RejectionReason); err != nil {
			return nil, err
		}
		if verifiedAt.Valid && verifiedAt.Int64 > 0 {
			v := verifiedAt.Int64
			p.VerifiedAt = &v
			if p.Status == "" || p.Status == "paid" {
				p.Status = "verified"
			}
		} else if p.Status == "paid" || p.Status == "" {
			p.Status = "pending"
		}
		out[jobID] = &p
	}
	return out, rows.Err()
}

// matchingCrew selects the applicant profiles p who can take job j.id=$1.
const matchingCrew = `FROM jobs j
		JOIN applicant_profiles p ON p.is_active=true AND lower(trim(p.city))=lower(trim(j.city))
			AND p.employment_type=j.employment_type AND ` + eligibleCrew + `
		JOIN applicant_roles ar ON ar.applicant_id=p.user_id AND ar.role_id=j.role_id
		WHERE j.id=$1
		AND NOT EXISTS (
			SELECT 1 FROM job_assignments active_assignment
			JOIN jobs assigned_job ON assigned_job.id=active_assignment.job_id
			WHERE active_assignment.applicant_id=p.user_id AND active_assignment.ended_at IS NULL
			AND assigned_job.role_id=j.role_id
		)
		AND NOT EXISTS (
			SELECT 1 FROM job_applications ja
			WHERE ja.job_id=j.id AND ja.applicant_id=p.user_id AND ja.status='rejected'
		)
		AND ((j.timing='now' AND p.availability_mode='now') OR
			(j.timing='later' AND (p.availability_mode='now' OR EXISTS (
				SELECT 1 FROM applicant_availability av
				WHERE av.applicant_id=p.user_id AND av.status='available'
				AND av.starts_at<=j.scheduled_at AND av.ends_at>j.scheduled_at
			))))`

func (s *Store) notifyMatchingCrew(ctx context.Context, tx *sql.Tx, jobID int64, now time.Time) {
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		SELECT p.user_id,'New shift available',j.role_label || ' in ' || j.city,'match',json_build_object('job_id', j.id)::text,$2
		`+matchingCrew, jobID, now.Unix())
}

// EmployerSearchPenalty is the transit-cancel penalty the employer still owes.
func (s *Store) EmployerSearchPenalty(ctx context.Context, employerID int64) (int64, error) {
	var p int64
	err := s.db.QueryRowContext(ctx, `SELECT search_penalty_centavos FROM users WHERE id=$1`, employerID).Scan(&p)
	return p, err
}

// RecordMatchFee records the current search's fee once a crew member has accepted.
// The fee is baseCentavos plus the employer's owed penalties. Receipts start as
// pending until accounting verifies them; a rejected receipt can be replaced, and
// is re-priced so penalties added since are included.
func (s *Store) RecordMatchFee(ctx context.Context, employerID, id int64, method, details, proofImage string, baseCentavos int64, ref string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var penalty int64
	if err := tx.QueryRowContext(ctx, `SELECT search_penalty_centavos FROM users WHERE id=$1 FOR UPDATE`, employerID).Scan(&penalty); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	var attempt int
	err = tx.QueryRowContext(ctx, `SELECT search_attempts FROM jobs WHERE id=$1 AND employer_id=$2 AND status='accepted' FOR UPDATE`,
		id, employerID).Scan(&attempt)
	if errors.Is(err, sql.ErrNoRows) {
		return missOrBadState(ctx, tx, employerID, id)
	}
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	var paymentID int64
	var status string
	err = tx.QueryRowContext(ctx, `SELECT id, COALESCE(status,'') FROM payments WHERE job_id=$1 AND search_attempt=$2`, id, attempt).Scan(&paymentID, &status)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		err = tx.QueryRowContext(ctx, `
			INSERT INTO payments (job_id, user_id, amount_centavos, penalty_centavos, method, details, proof_image, search_attempt, status, provider_ref, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending',$9,$10) RETURNING id`,
			id, employerID, baseCentavos+penalty, penalty, method, details, proofImage, attempt, ref, now).Scan(&paymentID)
	case err != nil:
		return err
	case status == "rejected":
		_, err = tx.ExecContext(ctx, `
			UPDATE payments SET amount_centavos=$1, penalty_centavos=$2, method=$3, details=$4, proof_image=$5, status='pending',
				provider_ref=$6, created_at=$7, verified_at=NULL, verified_by='', rejection_reason=''
			WHERE id=$8`,
			baseCentavos+penalty, penalty, method, details, proofImage, ref, now, paymentID)
	default:
		return nil // already pending or verified: a repeated submit changes nothing
	}
	if err != nil {
		return err
	}
	jobID := id
	if err := addPaymentSubmittedNotification(ctx, tx, employerID, paymentID, "employer", &jobID, now); err != nil {
		return err
	}
	return tx.Commit()
}

// MatchingApplicants returns the crew who can currently take this open search.
func (s *Store) MatchingApplicants(ctx context.Context, jobID int64) ([]MatchedApplicant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.name,p.expected_rate,p.rate_period,p.city,p.availability_mode,p.rating_hundredths/100.0
		`+matchingCrew+`
		AND j.status='matching'
		ORDER BY u.name`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := make([]MatchedApplicant, 0)
	for rows.Next() {
		var applicant MatchedApplicant
		if err := rows.Scan(&applicant.Name, &applicant.ExpectedRate, &applicant.RatePeriod, &applicant.City, &applicant.AvailabilityMode, &applicant.Rating); err != nil {
			return nil, err
		}
		matches = append(matches, applicant)
	}
	return matches, rows.Err()
}

// DeclineMatch lets the employer turn down the crew member who accepted. The job
// goes back to an open search, and that crew member is not offered it again.
func (s *Store) DeclineMatch(ctx context.Context, employerID, id int64) (*Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `UPDATE jobs SET accepted_count = 0, status = 'matching', updated_at = $1
		WHERE id = $2 AND employer_id = $3 AND status = 'accepted'`, now, id, employerID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, missOrBadState(ctx, tx, employerID, id)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE job_assignments SET ended_at = $1 WHERE job_id = $2 AND ended_at IS NULL`, now, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE job_applications SET status = 'rejected', updated_at = $1 WHERE job_id = $2 AND status = 'accepted'`, now, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetJob(ctx, employerID, id)
}

// ReleaseUnpaidMatches frees crew whose employer submitted no search fee receipt
// within PaymentDeadline of their acceptance: the job goes back to an open
// search and both sides are told. It returns how many jobs it released.
// shortcut: a rejected receipt keeps the crew member held until the employer
// re-uploads or declines; add a rejection timestamp if that gets abused.
func (s *Store) ReleaseUnpaidMatches(ctx context.Context) (int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT j.id FROM jobs j JOIN job_assignments a ON a.job_id=j.id AND a.ended_at IS NULL
		WHERE j.status='accepted' AND a.accepted_at <= $1
		AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.job_id=j.id AND p.search_attempt=j.search_attempts)`,
		time.Now().Add(-PaymentDeadline).Unix())
	if err != nil {
		return 0, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	count := 0
	for _, id := range ids {
		ok, err := s.releaseUnpaidMatch(ctx, id)
		if err != nil {
			return count, err
		}
		if ok {
			count++
		}
	}
	return count, nil
}

func (s *Store) releaseUnpaidMatch(ctx context.Context, jobID int64) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Lock first, then re-check in a fresh statement: a receipt submitted while we
	// waited for the lock must keep the crew member.
	var employerID int64
	var roleLabel string
	err = tx.QueryRowContext(ctx, `SELECT employer_id, role_label FROM jobs WHERE id=$1 AND status='accepted' FOR UPDATE`, jobID).Scan(&employerID, &roleLabel)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var paid bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM payments p JOIN jobs j ON j.id=p.job_id
		WHERE j.id=$1 AND p.search_attempt=j.search_attempts)`, jobID).Scan(&paid); err != nil || paid {
		return false, err
	}

	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		SELECT applicant_id,'Booking released',$1,'end','{}',$2 FROM job_assignments WHERE job_id=$3 AND ended_at IS NULL`,
		"The employer didn't pay the search fee for the "+roleLabel+" shift in time, so you're free to take other shifts.", now, jobID); err != nil {
		return false, err
	}
	// The crew member can't accept this same job again (one assignment per job and crew member).
	if _, err := tx.ExecContext(ctx, `UPDATE job_assignments SET ended_at=$1 WHERE job_id=$2 AND ended_at IS NULL`, now, jobID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status='matching', accepted_count=0, updated_at=$1 WHERE id=$2`, now, jobID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		VALUES($1,'Crew released',$2,'match',json_build_object('job_id',$3::bigint)::text,$4)`,
		employerID, "The search fee wasn't paid within 3 minutes, so your crew member was released. Your "+roleLabel+" search is open again.",
		jobID, now); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// AcceptJob atomically gives the offer to the first applicant who accepts.
// No later applicant can accept after the job leaves the matching state.
func (s *Store) AcceptJob(ctx context.Context, jobID, applicantID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	standing, err := applicantStanding(ctx, tx, applicantID, time.Now())
	if err != nil {
		return err
	}
	if standing.PermanentlyDeactivated {
		return ErrAccountDeactivated
	}
	if standing.Suspended {
		return ErrAccountSuspended
	}
	// A hire is only spent when the employer confirms the booking (see ConfirmJob).
	// Here we only require a hire that is not already reserved by another
	// accepted-but-unconfirmed shift, so confirmation can always spend one.
	// Lock the pass first, in its own statement: a second concurrent accept by the
	// same applicant waits here, and the count below then sees the shift this one
	// reserved, so one hire can't cover two shifts.
	var locked int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM applicant_hiring_passes WHERE applicant_id=$1 FOR UPDATE`, applicantID).Scan(&locked); err != nil && err != sql.ErrNoRows {
		return err
	}
	var available int
	err = tx.QueryRowContext(ctx, `SELECT p.remaining_hires - (
			SELECT COUNT(*) FROM job_assignments a JOIN jobs j ON j.id=a.job_id
			WHERE a.applicant_id=p.applicant_id AND a.ended_at IS NULL AND j.status='accepted')
		FROM applicant_hiring_passes p WHERE p.applicant_id=$1`, applicantID).Scan(&available)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == sql.ErrNoRows || available <= 0 {
		// Distinguish "never paid / exhausted" from "receipt waiting on accounting".
		var payStatus string
		payErr := tx.QueryRowContext(ctx, `SELECT COALESCE(status,'') FROM applicant_hiring_pass_payments
			WHERE applicant_id=$1 ORDER BY id DESC LIMIT 1`, applicantID).Scan(&payStatus)
		if payErr == nil {
			switch strings.ToLower(payStatus) {
			case "pending":
				return ErrHiringPassPending
			case "rejected":
				return ErrPaymentRejected
			}
		}
		return ErrHiringPassEmpty
	}

	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `
		UPDATE jobs SET accepted_count = 1,
			status = 'accepted',
			updated_at = $1
		WHERE id = $2 AND status = 'matching' AND accepted_count = 0
		AND NOT EXISTS (
			SELECT 1 FROM job_assignments active_assignment
			JOIN jobs assigned_job ON assigned_job.id=active_assignment.job_id
			WHERE active_assignment.applicant_id=$3 AND active_assignment.ended_at IS NULL
			AND assigned_job.role_id=jobs.role_id
		)`, now, jobID, applicantID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrBadState // not found, expired, or already full
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO job_assignments (job_id, applicant_id, accepted_at) VALUES ($1, $2, $3)`,
		jobID, applicantID, now); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrAlreadyAccepted
		}
		return err
	}
	_, _ = tx.ExecContext(ctx, `INSERT INTO job_applications(job_id,applicant_id,status,applied_at,updated_at) VALUES($1,$2,'accepted',$3,$4) ON CONFLICT(job_id,applicant_id) DO UPDATE SET status='accepted',updated_at=excluded.updated_at`, jobID, applicantID, now, now)
	if _, err := tx.ExecContext(ctx, `INSERT INTO applicant_rating_events(job_id,applicant_id,outcome) VALUES($1,$2,'accepted') ON CONFLICT DO NOTHING`, jobID, applicantID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE applicant_profiles SET rating_hundredths=LEAST(500,rating_hundredths+1) WHERE user_id=$1 AND EXISTS(SELECT 1 FROM applicant_rating_events WHERE job_id=$2 AND applicant_id=$3 AND outcome='accepted')`, applicantID, jobID, applicantID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) transition(ctx context.Context, employerID, id int64, to string, from ...string) error {
	args := []any{to, time.Now().Unix(), id, employerID}
	for _, f := range from {
		args = append(args, f)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = $1, updated_at = $2 WHERE id = $3 AND employer_id = $4 AND status IN (`+
			placeholders(5, len(from))+`)`, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, s.db, employerID, id)
	}
	return nil
}

// ConfirmJob is the moment an employer secures the crew member, once accounting has
// verified the current search's fee. Verifying the fee also confirms the job (see
// SetPaymentStatus); this covers a fee verified before the current crew member accepted.
func (s *Store) ConfirmJob(ctx context.Context, employerID, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := confirmJob(ctx, tx, employerID, id, time.Now().Unix()); err != nil {
		return err
	}
	return tx.Commit()
}

// confirmJob moves accepted -> active and spends one hire per assigned crew member
// in the caller's transaction, so a hire is never charged for an unconfirmed or
// declined match. It needs the current search's fee to be verified.
func confirmJob(ctx context.Context, tx *sql.Tx, employerID, id, now int64) error {
	var payStatus string
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(p.status,'') FROM jobs j
		JOIN payments p ON p.job_id=j.id AND p.search_attempt=j.search_attempts
		WHERE j.id=$1 AND j.employer_id=$2`, id, employerID).Scan(&payStatus)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	switch payStatus {
	case "verified":
	case "rejected":
		return ErrPaymentRejected
	default:
		if errors.Is(missOrBadState(ctx, tx, employerID, id), ErrNotFound) {
			return ErrNotFound
		}
		return ErrPaymentNotVerified
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE jobs SET status='active', updated_at=$1 WHERE id=$2 AND employer_id=$3 AND status='accepted'`,
		now, id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, tx, employerID, id)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		SELECT applicant_id,'Booking confirmed','The employer confirmed your shift booking.','accept','{}',$1
		FROM job_assignments WHERE job_id=$2 AND ended_at IS NULL`, now, id); err != nil {
		return err
	}
	// The arrival clock starts now, or at the scheduled start for later shifts.
	if _, err := tx.ExecContext(ctx, `UPDATE job_assignments
		SET arrival_deadline_at = GREATEST($1, COALESCE((SELECT scheduled_at FROM jobs WHERE id=$2), $1)) + $3
		WHERE job_id=$2 AND ended_at IS NULL`, now, id, int64(ArrivalLimit/time.Second)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE applicant_hiring_passes
		SET remaining_hires=remaining_hires-1, used_hires=used_hires+1, updated_at=$1
		WHERE remaining_hires>0 AND applicant_id IN (
			SELECT applicant_id FROM job_assignments WHERE job_id=$2 AND ended_at IS NULL)`, now, id); err != nil {
		return err
	}
	return nil
}

func (s *Store) CancelJob(ctx context.Context, employerID, id int64) error {
	return s.transition(ctx, employerID, id, "cancelled", "draft", "matching", "unfulfilled")
}

func (s *Store) EndJob(ctx context.Context, employerID, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx,
		`UPDATE jobs SET status='completed', updated_at=$1 WHERE id=$2 AND employer_id=$3 AND status='active'`,
		now, id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, tx, employerID, id)
	}
	// Ending crew who are already on the way adds a penalty per crew member to the next search fee.
	res, err = tx.ExecContext(ctx, `UPDATE users SET search_penalty_centavos = search_penalty_centavos + $1 * (
			SELECT COUNT(*) FROM job_assignments WHERE job_id=$2 AND ended_at IS NULL AND transit_status='en_route')
		WHERE id=$3 AND EXISTS (SELECT 1 FROM job_assignments WHERE job_id=$2 AND ended_at IS NULL AND transit_status='en_route')`,
		TransitCancelPenaltyCentavos, id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
			VALUES($1,'Cancellation penalty',$2,'payment',json_build_object('job_id',$3::bigint)::text,$4)`,
			employerID, "You ended the contract while your crew was on the way. "+pesos(TransitCancelPenaltyCentavos)+" per crew member is added to your next crew search fee.",
			id, now); err != nil {
			return err
		}
	}
	// Crew ended while on the way didn't cause it, so they get their hire back.
	if _, err := tx.ExecContext(ctx, `UPDATE job_assignments SET hire_refunded=true
		WHERE job_id=$1 AND ended_at IS NULL AND transit_status='en_route'`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE applicant_hiring_passes
		SET remaining_hires=remaining_hires+1, used_hires=used_hires-1, updated_at=$1
		WHERE used_hires>0 AND applicant_id IN (
			SELECT applicant_id FROM job_assignments WHERE job_id=$2 AND ended_at IS NULL AND hire_refunded)`, now, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		SELECT applicant_id,'Hire returned','The employer ended the shift while you were on the way, so your hiring opportunity was returned.','end','{}',$1
		FROM job_assignments WHERE job_id=$2 AND ended_at IS NULL AND hire_refunded`, now, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE job_assignments SET ended_at = $1 WHERE job_id = $2 AND ended_at IS NULL`, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

// Assignments: call only after the caller has proven they own the job.
func (s *Store) Assignments(ctx context.Context, jobID int64) ([]Assignment, error) {
	m, err := s.AssignmentsForJobs(ctx, []int64{jobID})
	return m[jobID], err
}

// AssignmentsForJobs loads the crew of many jobs in one query, keyed by job ID.
// Call only with jobs the caller may see.
func (s *Store) AssignmentsForJobs(ctx context.Context, jobIDs []int64) (map[int64][]Assignment, error) {
	out := map[int64][]Assignment{}
	if len(jobIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.job_id, a.applicant_id, u.name, a.accepted_at, a.ended_at, a.transit_status, a.arrival_deadline_at,
			COALESCE(p.rating_hundredths, 0)/100.0
		FROM job_assignments a JOIN users u ON u.id = a.applicant_id
		LEFT JOIN applicant_profiles p ON p.user_id = a.applicant_id
		WHERE a.job_id = ANY($1) ORDER BY a.accepted_at`, jobIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Assignment
		var jobID, acc int64
		var ended, deadline sql.NullInt64
		if err := rows.Scan(&jobID, &a.ApplicantID, &a.Name, &acc, &ended, &a.Transit, &deadline, &a.Rating); err != nil {
			return nil, err
		}
		a.AcceptedAt = time.Unix(acc, 0)
		if ended.Valid {
			t := time.Unix(ended.Int64, 0)
			a.EndedAt = &t
		}
		if deadline.Valid {
			t := time.Unix(deadline.Int64, 0)
			a.ArrivalDeadlineAt = &t
		}
		out[jobID] = append(out[jobID], a)
	}
	return out, rows.Err()
}
