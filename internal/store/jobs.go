package store

import (
	"context"
	"database/sql"
	"errors"
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
	CreatedAt, UpdatedAt        time.Time
}

type Assignment struct {
	ApplicantID int64
	Name        string
	AcceptedAt  time.Time
	EndedAt     *time.Time
	Transit     string
}

// MatchedApplicant contains only the profile details employers need to identify
// a crew member while a matching search is open. Contact information is omitted.
type MatchedApplicant struct {
	Name             string `json:"name"`
	ExpectedRate     int64  `json:"expected_rate"`
	RatePeriod       string `json:"rate_period"`
	City             string `json:"city"`
	AvailabilityMode string `json:"availability_mode"`
}

const jobCols = `id, employer_id, category, role_id, role_label, employment_type, rate_centavos,
	rate_period, city, map_location, maps_url, lat, lng, timing, scheduled_at, crew_needed,
	accepted_count, status, search_attempts, search_expires_at, current_try, max_tries, current_attempt, max_attempts, created_at, updated_at`

type rowScanner interface{ Scan(dest ...any) error }

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanJob(r rowScanner) (*Job, error) {
	var j Job
	var lat, lng sql.NullFloat64
	var sched, exp sql.NullInt64
	var curTry, maxTries, curAtt, maxAtt sql.NullInt64
	var created, updated int64
	err := r.Scan(&j.ID, &j.EmployerID, &j.Category, &j.RoleID, &j.RoleLabel, &j.EmploymentType,
		&j.RateCentavos, &j.RatePeriod, &j.City, &j.MapLocation, &j.MapsURL, &lat, &lng, &j.Timing,
		&sched, &j.CrewNeeded, &j.AcceptedCount, &j.Status, &j.SearchAttempts, &exp,
		&curTry, &maxTries, &curAtt, &maxAtt, &created, &updated)
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
	j.CreatedAt, j.UpdatedAt = time.Unix(created, 0), time.Unix(updated, 0)
	return &j, nil
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
	err := q.QueryRowContext(ctx, `SELECT 1 FROM jobs WHERE id = ? AND employer_id = ?`, id, employerID).Scan(&one)
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
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO jobs (employer_id, category, role_id, role_label, employment_type, rate_centavos,
			rate_period, city, map_location, maps_url, lat, lng, timing, scheduled_at, crew_needed,
			status, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'draft',?,?)`,
		j.EmployerID, j.Category, j.RoleID, j.RoleLabel, j.EmploymentType, j.RateCentavos,
		j.RatePeriod, j.City, j.MapLocation, j.MapsURL, nullF(j.Lat), nullF(j.Lng), j.Timing,
		nullT(j.ScheduledAt), j.CrewNeeded, now, now)
	if err != nil {
		return err
	}
	j.ID, _ = res.LastInsertId()
	return nil
}

// UpdateJob edits details; only allowed before/after a search (draft, unfulfilled).
func (s *Store) UpdateJob(ctx context.Context, j *Job) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE jobs SET category=?, role_id=?, role_label=?, employment_type=?, rate_centavos=?,
			rate_period=?, city=?, map_location=?, maps_url=?, lat=?, lng=?, timing=?, scheduled_at=?,
			crew_needed=?, updated_at=?
		WHERE id=? AND employer_id=? AND status IN ('draft','unfulfilled')`,
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
func (s *Store) GetJob(ctx context.Context, employerID, id int64) (*Job, error) {
	return scanJob(s.db.QueryRowContext(ctx,
		`SELECT `+jobCols+` FROM jobs WHERE id = ? AND employer_id = ?`, id, employerID))
}

func (s *Store) ListJobs(ctx context.Context, employerID int64, statuses []string, limit, offset int) ([]*Job, error) {
	q := `SELECT ` + jobCols + ` FROM jobs WHERE employer_id = ?`
	args := []any{employerID}
	if len(statuses) > 0 {
		q += ` AND status IN (` + strings.TrimSuffix(strings.Repeat("?,", len(statuses)), ",") + `)`
		for _, st := range statuses {
			args = append(args, st)
		}
	}
	q += ` ORDER BY id DESC LIMIT ? OFFSET ?`
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

// StartSearch opens a search window only after its fee has been recorded.
// The status guard makes double-clicks and concurrent requests safe: only one wins.
func (s *Store) StartSearch(ctx context.Context, employerID, id int64, window time.Duration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	res, err := tx.ExecContext(ctx, `
		UPDATE jobs SET status='matching', search_attempts = search_attempts + 1,
			current_try = 1, max_tries = 3, current_attempt = 1, max_attempts = 3,
			search_expires_at = ?, updated_at = ?
		WHERE id = ? AND employer_id = ? AND status IN ('draft','unfulfilled')
		AND EXISTS (SELECT 1 FROM payments p WHERE p.job_id=jobs.id
			AND p.search_attempt=jobs.search_attempts+1 AND p.status IN ('paid', 'pending', 'verified'))`,
		now.Add(window).Unix(), now.Unix(), id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, tx, employerID, id)
	}
	s.notifyMatchingCrew(ctx, tx, id, now)
	return tx.Commit()
}

func (s *Store) notifyMatchingCrew(ctx context.Context, tx *sql.Tx, jobID int64, now time.Time) {
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		SELECT p.user_id,'New shift available',j.role_label || ' in ' || j.city,'match',json_object('job_id', j.id),?
		FROM jobs j JOIN applicant_profiles p ON p.is_active=1 AND lower(trim(p.city))=lower(trim(j.city))
		JOIN applicant_roles ar ON ar.applicant_id=p.user_id AND ar.role_id=j.role_id
		WHERE j.id=? AND p.employment_type=j.employment_type
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
func (s *Store) RecordMatchFee(ctx context.Context, employerID, id int64, method, details, proofImage string, feeCentavos int64, ref string) error {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO payments (job_id, user_id, amount_centavos, method, details, proof_image, search_attempt, status, provider_ref, created_at)
		SELECT id, employer_id, ?, ?, ?, ?, search_attempts+1, 'pending', ?, ? FROM jobs
		WHERE id=? AND employer_id=? AND status IN ('draft','unfulfilled')
		AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.job_id=jobs.id AND p.search_attempt=jobs.search_attempts+1)`,
		feeCentavos, method, details, proofImage, ref, time.Now().Unix(), id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var sameAttempt int
		err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs j JOIN payments p ON p.job_id=j.id
			WHERE j.id=? AND j.employer_id=?
			AND ((j.status IN ('draft','unfulfilled') AND p.search_attempt=j.search_attempts+1)
				OR (j.status='matching' AND p.search_attempt=j.search_attempts))
			AND p.status IN ('paid', 'pending', 'verified') AND p.method=?`,
			id, employerID, method).Scan(&sameAttempt)
		if err != nil {
			return err
		}
		if sameAttempt > 0 {
			return nil
		}
		return missOrBadState(ctx, s.db, employerID, id)
	}
	return nil
}

// MatchingApplicants reads eligible active profiles for an employer-owned job.
// The caller verifies job ownership before asking for the results.
func (s *Store) MatchingApplicants(ctx context.Context, jobID int64) ([]MatchedApplicant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.name,p.expected_rate,p.rate_period,p.city,p.availability_mode
		FROM jobs j
		JOIN applicant_profiles p ON p.is_active=1
		JOIN applicant_roles ar ON ar.applicant_id=p.user_id AND ar.role_id=j.role_id
		JOIN users u ON u.id=p.user_id
		WHERE j.id=? AND j.status='matching'
		AND (j.search_expires_at IS NULL OR j.search_expires_at>?)
		AND p.employment_type=j.employment_type
		AND lower(trim(p.city))=lower(trim(j.city))
		AND ((j.timing='now' AND p.availability_mode='now') OR
			(j.timing='later' AND (p.availability_mode='now' OR EXISTS (
				SELECT 1 FROM applicant_availability av
				WHERE av.applicant_id=p.user_id AND av.status='available'
				AND av.starts_at<=j.scheduled_at AND av.ends_at>j.scheduled_at
			))))
		AND NOT EXISTS (
			SELECT 1 FROM job_applications a
			WHERE a.job_id=j.id AND a.applicant_id=p.user_id
			AND a.status IN ('pending','accepted','rejected')
		)
		AND NOT EXISTS (
			SELECT 1 FROM job_assignments active_assignment
			JOIN jobs assigned_job ON assigned_job.id=active_assignment.job_id
			WHERE active_assignment.applicant_id=p.user_id AND active_assignment.ended_at IS NULL
			AND assigned_job.role_id=j.role_id
		)
		ORDER BY p.expected_rate ASC,u.name COLLATE NOCASE`, jobID, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]MatchedApplicant, 0)
	for rows.Next() {
		var applicant MatchedApplicant
		if err := rows.Scan(&applicant.Name, &applicant.ExpectedRate, &applicant.RatePeriod, &applicant.City, &applicant.AvailabilityMode); err != nil {
			return nil, err
		}
		matches = append(matches, applicant)
	}
	return matches, rows.Err()
}

// ExpireSearches advances timed-out searches to the next attempt, next try, or unfulfilled.
func (s *Store) ExpireSearches(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, employer_id FROM jobs
		WHERE status='matching' AND search_expires_at IS NOT NULL AND search_expires_at <= ?`, now)
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
		if _, err := s.AdvanceSearchAttempt(ctx, it.empID, it.id, 3*time.Minute); err == nil {
			count++
		}
	}
	return count, nil
}

// AdvanceSearchAttempt advances to the next attempt (within the try) or next try (up to 3 tries, 3 attempts each).
func (s *Store) AdvanceSearchAttempt(ctx context.Context, employerID, id int64, window time.Duration) (*Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	j, err := scanJob(tx.QueryRowContext(ctx, `SELECT `+jobCols+` FROM jobs WHERE id=? AND employer_id=?`, id, employerID))
	if err != nil {
		return nil, err
	}
	if j.Status != "matching" {
		return j, nil
	}

	now := time.Now()
	if j.CurrentAttempt < j.MaxAttempts {
		j.CurrentAttempt++
		newExpires := now.Add(window).Unix()
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs SET current_attempt=?, search_expires_at=?, updated_at=?
			WHERE id=? AND employer_id=? AND status='matching'`,
			j.CurrentAttempt, newExpires, now.Unix(), id, employerID); err != nil {
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
		newExpires := now.Add(window).Unix()
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs SET current_try=?, current_attempt=?, search_expires_at=?, updated_at=?
			WHERE id=? AND employer_id=? AND status='matching'`,
			j.CurrentTry, j.CurrentAttempt, newExpires, now.Unix(), id, employerID); err != nil {
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
		UPDATE jobs SET status='unfulfilled', updated_at=?
		WHERE id=? AND employer_id=? AND status='matching'`,
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

	j, err := scanJob(tx.QueryRowContext(ctx, `SELECT `+jobCols+` FROM jobs WHERE id=? AND employer_id=?`, id, employerID))
	if err != nil {
		return nil, err
	}
	if j.Status != "accepted" && j.Status != "matching" {
		return nil, ErrBadState
	}

	now := time.Now()

	// Clear assignments and mark application as rejected so they aren't immediately re-matched
	_, _ = tx.ExecContext(ctx, `UPDATE job_assignments SET ended_at = ? WHERE job_id = ? AND ended_at IS NULL`, now.Unix(), id)
	_, _ = tx.ExecContext(ctx, `UPDATE job_applications SET status = 'rejected', updated_at = ? WHERE job_id = ? AND status = 'accepted'`, now.Unix(), id)

	// Use up the current try
	if j.CurrentTry < j.MaxTries {
		nextTry := j.CurrentTry + 1
		newExpires := now.Add(window).Unix()
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs SET accepted_count = 0, status = 'matching',
				current_try = ?, current_attempt = 1, search_expires_at = ?, updated_at = ?
			WHERE id = ? AND employer_id = ?`,
			nextTry, newExpires, now.Unix(), id, employerID); err != nil {
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
		UPDATE jobs SET accepted_count = 0, status = 'unfulfilled', updated_at = ?
		WHERE id = ? AND employer_id = ?`,
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
	quota, err := tx.ExecContext(ctx, `UPDATE applicant_hiring_passes SET remaining_hires=remaining_hires-1,used_hires=used_hires+1,updated_at=? WHERE applicant_id=? AND remaining_hires>0`, time.Now().Unix(), applicantID)
	if err != nil {
		return err
	}
	if n, _ := quota.RowsAffected(); n == 0 {
		return ErrHiringPassEmpty
	}

	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `
		UPDATE jobs SET accepted_count = 1,
			status = 'accepted',
			updated_at = ?
		WHERE id = ? AND status = 'matching' AND (search_expires_at IS NULL OR search_expires_at > ?) AND accepted_count = 0
		AND NOT EXISTS (
			SELECT 1 FROM job_assignments active_assignment
			JOIN jobs assigned_job ON assigned_job.id=active_assignment.job_id
			WHERE active_assignment.applicant_id=? AND active_assignment.ended_at IS NULL
			AND assigned_job.role_id=jobs.role_id
		)`, now, jobID, now, applicantID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrBadState // not found, expired, or already full
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO job_assignments (job_id, applicant_id, accepted_at) VALUES (?, ?, ?)`,
		jobID, applicantID, now); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyAccepted
		}
		return err
	}
	_, _ = tx.ExecContext(ctx, `INSERT INTO job_applications(job_id,applicant_id,status,applied_at,updated_at) VALUES(?,?,'accepted',?,?) ON CONFLICT(job_id,applicant_id) DO UPDATE SET status='accepted',updated_at=excluded.updated_at`, jobID, applicantID, now, now)
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO applicant_rating_events(job_id,applicant_id,outcome) VALUES(?,?,'accepted')`, jobID, applicantID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE applicant_profiles SET rating_hundredths=MIN(500,rating_hundredths+1) WHERE user_id=? AND EXISTS(SELECT 1 FROM applicant_rating_events WHERE job_id=? AND applicant_id=? AND outcome='accepted')`, applicantID, jobID, applicantID); err != nil {
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
		`UPDATE jobs SET status = ?, updated_at = ? WHERE id = ? AND employer_id = ? AND status IN (`+
			strings.TrimSuffix(strings.Repeat("?,", len(from)), ",")+`)`, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, s.db, employerID, id)
	}
	return nil
}

func (s *Store) ConfirmJob(ctx context.Context, employerID, id int64) error {
	return s.transition(ctx, employerID, id, "active", "accepted")
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
		`UPDATE jobs SET status='completed', updated_at=? WHERE id=? AND employer_id=? AND status='active'`,
		now, id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, tx, employerID, id)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE job_assignments SET ended_at = ? WHERE job_id = ? AND ended_at IS NULL`, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

// Assignments: call only after the caller has proven they own the job.
func (s *Store) Assignments(ctx context.Context, jobID int64) ([]Assignment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.applicant_id, u.name, a.accepted_at, a.ended_at, a.transit_status
		FROM job_assignments a JOIN users u ON u.id = a.applicant_id
		WHERE a.job_id = ? ORDER BY a.accepted_at`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Assignment
	for rows.Next() {
		var a Assignment
		var acc int64
		var ended sql.NullInt64
		if err := rows.Scan(&a.ApplicantID, &a.Name, &acc, &ended, &a.Transit); err != nil {
			return nil, err
		}
		a.AcceptedAt = time.Unix(acc, 0)
		if ended.Valid {
			t := time.Unix(ended.Int64, 0)
			a.EndedAt = &t
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
