package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

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
	SearchAttempts              int
	SearchExpiresAt             *time.Time
	CurrentTry                  int
	MaxTries                    int
	CurrentAttempt              int
	MaxAttempts                 int
	CurrentBatch                int
	CreatedAt, UpdatedAt        time.Time
}

type Assignment struct {
	ApplicantID int64
	Name        string
	AcceptedAt  time.Time
	EndedAt     *time.Time
	Transit     string
	// ArrivalDeadlineAt is set once the employer confirms; nil before that.
	ArrivalDeadlineAt *time.Time
}

// eligibleCrew excludes applicants who are suspended or permanently deactivated.
const eligibleCrew = `NOT p.is_permanently_deactivated
	AND (p.deactivated_until IS NULL OR p.deactivated_until <= extract(epoch from now())::bigint)`

// MatchedApplicant contains only the profile details employers need to identify
// a crew member while a matching search is open. Contact information is omitted.
type MatchedApplicant struct {
	Name             string `json:"name"`
	ExpectedRate     int64  `json:"expected_rate"`
	RatePeriod       string `json:"rate_period"`
	City             string `json:"city"`
	AvailabilityMode string `json:"availability_mode"`
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
	accepted_count, status, search_attempts, search_expires_at, current_try, max_tries, current_attempt, max_attempts, current_batch, created_at, updated_at`

type rowScanner interface{ Scan(dest ...any) error }

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanJob(r rowScanner) (*Job, error) {
	var j Job
	var lat, lng sql.NullFloat64
	var sched, exp sql.NullInt64
	var curTry, maxTries, curAtt, maxAtt, curBatch sql.NullInt64
	var created, updated int64
	err := r.Scan(&j.ID, &j.EmployerID, &j.Category, &j.RoleID, &j.RoleLabel, &j.EmploymentType,
		&j.RateCentavos, &j.RatePeriod, &j.City, &j.MapLocation, &j.MapsURL, &lat, &lng, &j.Timing,
		&sched, &j.CrewNeeded, &j.AcceptedCount, &j.Status, &j.SearchAttempts, &exp,
		&curTry, &maxTries, &curAtt, &maxAtt, &curBatch, &created, &updated)
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
	if exp.Valid {
		t := time.Unix(exp.Int64, 0)
		j.SearchExpiresAt = &t
	}
	j.CurrentTry = 1
	if curTry.Valid && curTry.Int64 > 0 {
		j.CurrentTry = int(curTry.Int64)
	}
	j.MaxTries = 3
	if maxTries.Valid && maxTries.Int64 > 0 {
		j.MaxTries = int(maxTries.Int64)
	}
	j.CurrentAttempt = 1
	if curAtt.Valid && curAtt.Int64 > 0 {
		j.CurrentAttempt = int(curAtt.Int64)
	}
	j.MaxAttempts = 3
	if maxAtt.Valid && maxAtt.Int64 > 0 {
		j.MaxAttempts = int(maxAtt.Int64)
	}
	j.CurrentBatch = 1
	if curBatch.Valid && curBatch.Int64 > 0 {
		j.CurrentBatch = int(curBatch.Int64)
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

// StartSearch opens a search window only after accounting has verified the fee.
// The status guard makes double-clicks and concurrent requests safe: only one wins.
func (s *Store) StartSearch(ctx context.Context, employerID, id int64, window time.Duration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var payStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT p.status FROM jobs j
		JOIN payments p ON p.job_id=j.id AND p.search_attempt=j.search_attempts+1
		WHERE j.id=$1 AND j.employer_id=$2 AND j.status IN ('draft','unfulfilled')`,
		id, employerID).Scan(&payStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPaymentNotVerified
	}
	if err != nil {
		return err
	}
	switch strings.ToLower(payStatus) {
	case "verified":
	case "rejected":
		return ErrPaymentRejected
	default:
		return ErrPaymentNotVerified
	}

	now := time.Now()
	res, err := tx.ExecContext(ctx, `
		UPDATE jobs SET status='matching', search_attempts = search_attempts + 1,
			current_try = 1, max_tries = 3, current_attempt = 1, max_attempts = 3, current_batch = 1,
			search_expires_at = $1, updated_at = $2
		WHERE id = $3 AND employer_id = $4 AND status IN ('draft','unfulfilled')
		AND EXISTS (SELECT 1 FROM payments p WHERE p.job_id=jobs.id
			AND p.search_attempt=jobs.search_attempts+1 AND p.status='verified')`,
		now.Add(window).Unix(), now.Unix(), id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, tx, employerID, id)
	}
	var searchAttempt int
	if err := tx.QueryRowContext(ctx, `SELECT search_attempts FROM jobs WHERE id=$1`, id).Scan(&searchAttempt); err != nil {
		return err
	}
	if err := createMatchBatch(ctx, tx, id, searchAttempt, 1); err != nil {
		return err
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

func (s *Store) notifyMatchingCrew(ctx context.Context, tx *sql.Tx, jobID int64, now time.Time) {
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		SELECT p.user_id,'New shift available',j.role_label || ' in ' || j.city,'match',json_build_object('job_id', j.id)::text,$1
		FROM jobs j JOIN job_match_batch_members b ON b.job_id=j.id AND b.search_attempt=j.search_attempts AND b.batch_number=j.current_batch
		JOIN applicant_profiles p ON p.user_id=b.applicant_id AND p.is_active=true AND lower(trim(p.city))=lower(trim(j.city))
			AND `+eligibleCrew+`
		JOIN applicant_roles ar ON ar.applicant_id=p.user_id AND ar.role_id=j.role_id
		WHERE j.id=$2 AND p.employment_type=j.employment_type
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
			))))`, now.Unix(), jobID)
}

// RecordMatchFee records the fee for the next search attempt before crew are notified.
// Receipts start as pending until an admin verifies them. A rejected receipt can be replaced.
func (s *Store) RecordMatchFee(ctx context.Context, employerID, id int64, method, details, proofImage string, feeCentavos int64, ref string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	var paymentID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO payments (job_id, user_id, amount_centavos, method, details, proof_image, search_attempt, status, provider_ref, created_at)
		SELECT id, employer_id, $1::bigint, $2::text, $3::text, $4::text, search_attempts+1, 'pending', $5::text, $6::bigint FROM jobs
		WHERE id=$7 AND employer_id=$8 AND status IN ('draft','unfulfilled')
		AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.job_id=jobs.id AND p.search_attempt=jobs.search_attempts+1)
		RETURNING id`,
		feeCentavos, method, details, proofImage, ref, now, id, employerID).Scan(&paymentID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		jobID := id
		if err := addPaymentSubmittedNotification(ctx, tx, employerID, paymentID, "employer", &jobID, now); err != nil {
			return err
		}
		return tx.Commit()
	}

	// Replace a rejected receipt for the same attempt.
	res, err := tx.ExecContext(ctx, `
		UPDATE payments SET method=$1, details=$2, proof_image=$3, status='pending',
			provider_ref=$4, created_at=$5, verified_at=NULL, verified_by='', rejection_reason=''
		WHERE job_id=$6 AND search_attempt=(
			SELECT search_attempts+1 FROM jobs WHERE id=$7 AND employer_id=$8 AND status IN ('draft','unfulfilled')
		) AND status='rejected'`,
		method, details, proofImage, ref, now, id, id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		var paymentID int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM payments WHERE job_id=$1 AND search_attempt=(SELECT search_attempts+1 FROM jobs WHERE id=$2)`, id, id).Scan(&paymentID); err != nil {
			return err
		}
		jobID := id
		if err := addPaymentSubmittedNotification(ctx, tx, employerID, paymentID, "employer", &jobID, now); err != nil {
			return err
		}
		return tx.Commit()
	}

	var sameAttempt int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs j JOIN payments p ON p.job_id=j.id
		WHERE j.id=$1 AND j.employer_id=$2
		AND ((j.status IN ('draft','unfulfilled') AND p.search_attempt=j.search_attempts+1)
			OR (j.status='matching' AND p.search_attempt=j.search_attempts))
		AND p.status IN ('paid', 'pending', 'verified') AND p.method=$3`,
		id, employerID, method).Scan(&sameAttempt)
	if err != nil {
		return err
	}
	if sameAttempt > 0 {
		return nil
	}
	return missOrBadState(ctx, tx, employerID, id)
}

// MatchingApplicants returns the stable randomized batch selected for the current
// three-minute window. Polling this endpoint never changes the batch.
func (s *Store) MatchingApplicants(ctx context.Context, jobID int64) ([]MatchedApplicant, error) {
	// Backfill a batch for searches that were already active when this version deployed.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	var attempt, batch int
	if err := tx.QueryRowContext(ctx, `SELECT search_attempts,current_batch FROM jobs WHERE id=$1 AND status='matching'`, jobID).Scan(&attempt, &batch); err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return []MatchedApplicant{}, nil
		}
		return nil, err
	}
	var existing int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM job_match_batch_members WHERE job_id=$1 AND search_attempt=$2 AND batch_number=$3`, jobID, attempt, batch).Scan(&existing); err != nil {
		tx.Rollback()
		return nil, err
	}
	if existing == 0 {
		if err := createMatchBatch(ctx, tx, jobID, attempt, batch); err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT u.name,p.expected_rate,p.rate_period,p.city,p.availability_mode
		FROM job_match_batch_members b
		JOIN jobs j ON j.id=b.job_id AND j.search_attempts=b.search_attempt AND j.current_batch=b.batch_number
		JOIN applicant_profiles p ON p.user_id=b.applicant_id
		JOIN users u ON u.id=p.user_id
		WHERE b.job_id=$1 AND j.status='matching' AND (j.search_expires_at IS NULL OR j.search_expires_at>$2)
		AND NOT EXISTS (SELECT 1 FROM job_applications a WHERE a.job_id=b.job_id AND a.applicant_id=b.applicant_id AND a.status='rejected')
		ORDER BY b.slot`, jobID, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := make([]MatchedApplicant, 0, 5)
	for rows.Next() {
		var applicant MatchedApplicant
		if err := rows.Scan(&applicant.Name, &applicant.ExpectedRate, &applicant.RatePeriod, &applicant.City, &applicant.AvailabilityMode); err != nil {
			return nil, err
		}
		matches = append(matches, applicant)
	}
	return matches, rows.Err()
}

type matchBatchExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func createMatchBatch(ctx context.Context, q matchBatchExecer, jobID int64, searchAttempt, batch int) error {
	rows, err := q.QueryContext(ctx, `
		SELECT p.user_id FROM jobs j
		JOIN applicant_profiles p ON p.is_active=true AND `+eligibleCrew+`
		JOIN applicant_roles ar ON ar.applicant_id=p.user_id AND ar.role_id=j.role_id
		WHERE j.id=$1 AND j.status='matching' AND p.employment_type=j.employment_type
		AND lower(trim(p.city))=lower(trim(j.city))
		AND ((j.timing='now' AND p.availability_mode='now') OR (j.timing='later' AND (p.availability_mode='now' OR EXISTS (
			SELECT 1 FROM applicant_availability av WHERE av.applicant_id=p.user_id AND av.status='available'
			AND av.starts_at<=j.scheduled_at AND av.ends_at>j.scheduled_at))))
		AND NOT EXISTS (SELECT 1 FROM job_applications a WHERE a.job_id=j.id AND a.applicant_id=p.user_id AND a.status IN ('pending','accepted','rejected'))
		AND NOT EXISTS (SELECT 1 FROM job_assignments a JOIN jobs aj ON aj.id=a.job_id
			WHERE a.applicant_id=p.user_id AND a.ended_at IS NULL AND aj.role_id=j.role_id)`, jobID)
	if err != nil {
		return err
	}
	var pool []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		pool = append(pool, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	seen := map[int64]bool{}
	rows, err = q.QueryContext(ctx, `SELECT DISTINCT applicant_id FROM job_match_batch_members WHERE job_id=$1 AND search_attempt=$2`, jobID, searchAttempt)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		seen[id] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(pool) == 0 {
		return nil
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	var fresh, chosen []int64
	for _, id := range pool {
		if !seen[id] {
			fresh = append(fresh, id)
		}
	}
	rand.Shuffle(len(fresh), func(i, j int) { fresh[i], fresh[j] = fresh[j], fresh[i] })
	limit := 5
	if len(pool) < limit {
		limit = len(pool)
	}
	if len(fresh) > limit {
		fresh = fresh[:limit]
	}
	chosen = append(chosen, fresh...)
	if len(chosen) < limit {
		selected := map[int64]bool{}
		for _, id := range chosen {
			selected[id] = true
		}
		var repeat []int64
		for _, id := range pool {
			if !selected[id] {
				repeat = append(repeat, id)
			}
		}
		rand.Shuffle(len(repeat), func(i, j int) { repeat[i], repeat[j] = repeat[j], repeat[i] })
		chosen = append(chosen, repeat[:limit-len(chosen)]...)
	}
	for slot, id := range chosen {
		if _, err := q.ExecContext(ctx, `INSERT INTO job_match_batch_members(job_id,search_attempt,batch_number,slot,applicant_id) VALUES($1,$2,$3,$4,$5)`, jobID, searchAttempt, batch, slot, id); err != nil {
			return err
		}
	}
	return nil
}

// ExpireSearches advances timed-out searches to the next attempt, next try, or unfulfilled.
func (s *Store) ExpireSearches(ctx context.Context, window time.Duration) (int64, error) {
	now := time.Now().Unix()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, employer_id FROM jobs
		WHERE status='matching' AND search_expires_at IS NOT NULL AND search_expires_at <= $1`, now)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type refItem struct {
		id, empID int64
	}
	var expired []refItem
	for rows.Next() {
		var it refItem
		if err := rows.Scan(&it.id, &it.empID); err == nil {
			expired = append(expired, it)
		}
	}
	rows.Close()

	var count int64
	for _, it := range expired {
		if _, err := s.AdvanceSearchAttempt(ctx, it.empID, it.id, window); err == nil {
			count++
		}
	}
	return count, nil
}

// AdvanceSearchAttempt advances to the next attempt (within the try) or next try (up to 3 tries, 3 attempts each).
// It only acts on a window that has really run out, so the background sweep and the
// employer's own timeout call can race without skipping a batch.
func (s *Store) AdvanceSearchAttempt(ctx context.Context, employerID, id int64, window time.Duration) (*Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	j, err := scanJob(tx.QueryRowContext(ctx, `SELECT `+jobCols+` FROM jobs WHERE id=$1 AND employer_id=$2 FOR UPDATE`, id, employerID))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if j.Status != "matching" || (j.SearchExpiresAt != nil && j.SearchExpiresAt.After(now)) {
		return j, nil
	}

	if j.CurrentAttempt < j.MaxAttempts {
		j.CurrentAttempt++
		j.CurrentBatch++
		newExpires := now.Add(window).Unix()
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs SET current_attempt=$1, current_batch=$2, search_expires_at=$3, updated_at=$4
			WHERE id=$5 AND employer_id=$6 AND status='matching'`,
			j.CurrentAttempt, j.CurrentBatch, newExpires, now.Unix(), id, employerID); err != nil {
			return nil, err
		}
		if err := createMatchBatch(ctx, tx, id, j.SearchAttempts, j.CurrentBatch); err != nil {
			return nil, err
		}
		s.notifyMatchingCrew(ctx, tx, id, now)
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.GetJob(ctx, employerID, id)
	}

	// Current try's 3 attempts (9 mins max) have elapsed!
	if j.CurrentTry < j.MaxTries {
		j.CurrentTry++
		j.CurrentAttempt = 1
		j.CurrentBatch++
		newExpires := now.Add(window).Unix()
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs SET current_try=$1, current_attempt=$2, current_batch=$3, search_expires_at=$4, updated_at=$5
			WHERE id=$6 AND employer_id=$7 AND status='matching'`,
			j.CurrentTry, j.CurrentAttempt, j.CurrentBatch, newExpires, now.Unix(), id, employerID); err != nil {
			return nil, err
		}
		if err := createMatchBatch(ctx, tx, id, j.SearchAttempts, j.CurrentBatch); err != nil {
			return nil, err
		}
		s.notifyMatchingCrew(ctx, tx, id, now)
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.GetJob(ctx, employerID, id)
	}

	// All 3 tries expired!
	if _, err := tx.ExecContext(ctx, `
		UPDATE jobs SET status='unfulfilled', updated_at=$1
		WHERE id=$2 AND employer_id=$3 AND status='matching'`,
		now.Unix(), id, employerID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetJob(ctx, employerID, id)
}

// DeclineMatch handles employer declining the matched applicant.
// The current try is used up, and the next try begins (if tries remain).
func (s *Store) DeclineMatch(ctx context.Context, employerID, id int64, window time.Duration) (*Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	j, err := scanJob(tx.QueryRowContext(ctx, `SELECT `+jobCols+` FROM jobs WHERE id=$1 AND employer_id=$2 FOR UPDATE`, id, employerID))
	if err != nil {
		return nil, err
	}
	if j.Status != "accepted" && j.Status != "matching" {
		return nil, ErrBadState
	}

	now := time.Now()

	// Clear assignments and mark application as rejected so they aren't immediately re-matched
	_, _ = tx.ExecContext(ctx, `UPDATE job_assignments SET ended_at = $1 WHERE job_id = $2 AND ended_at IS NULL`, now.Unix(), id)
	_, _ = tx.ExecContext(ctx, `UPDATE job_applications SET status = 'rejected', updated_at = $1 WHERE job_id = $2 AND status = 'accepted'`, now.Unix(), id)

	// Use up the current try
	if j.CurrentTry < j.MaxTries {
		nextTry := j.CurrentTry + 1
		nextBatch := j.CurrentBatch + 1
		newExpires := now.Add(window).Unix()
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs SET accepted_count = 0, status = 'matching',
				current_try = $1, current_attempt = 1, current_batch = $2, search_expires_at = $3, updated_at = $4
			WHERE id = $5 AND employer_id = $6`,
			nextTry, nextBatch, newExpires, now.Unix(), id, employerID); err != nil {
			return nil, err
		}
		if err := createMatchBatch(ctx, tx, id, j.SearchAttempts, nextBatch); err != nil {
			return nil, err
		}
		s.notifyMatchingCrew(ctx, tx, id, now)
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.GetJob(ctx, employerID, id)
	}

	// All 3 tries used up!
	if _, err := tx.ExecContext(ctx, `
		UPDATE jobs SET accepted_count = 0, status = 'unfulfilled', updated_at = $1
		WHERE id = $2 AND employer_id = $3`,
		now.Unix(), id, employerID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetJob(ctx, employerID, id)
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
		WHERE id = $2 AND status = 'matching' AND (search_expires_at IS NULL OR search_expires_at > $3) AND accepted_count = 0
		AND NOT EXISTS (
			SELECT 1 FROM job_assignments active_assignment
			JOIN jobs assigned_job ON assigned_job.id=active_assignment.job_id
			WHERE active_assignment.applicant_id=$4 AND active_assignment.ended_at IS NULL
			AND assigned_job.role_id=jobs.role_id
		)`, now, jobID, now, applicantID)
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

// ConfirmJob is the moment an employer secures the crew member. The job moves
// accepted -> active and one hire is spent for each assigned crew member, in the
// same transaction, so a hire is never charged for an unconfirmed or declined match.
func (s *Store) ConfirmJob(ctx context.Context, employerID, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx,
		`UPDATE jobs SET status='active', updated_at=$1 WHERE id=$2 AND employer_id=$3 AND status='accepted'`,
		now, id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, tx, employerID, id)
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
	return tx.Commit()
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
		SELECT a.job_id, a.applicant_id, u.name, a.accepted_at, a.ended_at, a.transit_status, a.arrival_deadline_at
		FROM job_assignments a JOIN users u ON u.id = a.applicant_id
		WHERE a.job_id = ANY($1) ORDER BY a.accepted_at`, jobIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Assignment
		var jobID, acc int64
		var ended, deadline sql.NullInt64
		if err := rows.Scan(&jobID, &a.ApplicantID, &a.Name, &acc, &ended, &a.Transit, &deadline); err != nil {
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
