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
	accepted_count, status, search_attempts, search_expires_at, created_at, updated_at`

type rowScanner interface{ Scan(dest ...any) error }

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanJob(r rowScanner) (*Job, error) {
	var j Job
	var lat, lng sql.NullFloat64
	var sched, exp sql.NullInt64
	var created, updated int64
	err := r.Scan(&j.ID, &j.EmployerID, &j.Category, &j.RoleID, &j.RoleLabel, &j.EmploymentType,
		&j.RateCentavos, &j.RatePeriod, &j.City, &j.MapLocation, &j.MapsURL, &lat, &lng, &j.Timing,
		&sched, &j.CrewNeeded, &j.AcceptedCount, &j.Status, &j.SearchAttempts, &exp, &created, &updated)
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

// StartSearch records the (mock) match-fee payment and opens a search window.
// The status guard makes double-clicks and concurrent requests safe: only one wins.
func (s *Store) StartSearch(ctx context.Context, employerID, id int64, method string, feeCentavos int64, ref string, window time.Duration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	res, err := tx.ExecContext(ctx, `
		UPDATE jobs SET status='matching', search_attempts = search_attempts + 1,
			search_expires_at = ?, updated_at = ?
		WHERE id = ? AND employer_id = ? AND status IN ('draft','unfulfilled')`,
		now.Add(window).Unix(), now.Unix(), id, employerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return missOrBadState(ctx, tx, employerID, id)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO payments (job_id, user_id, amount_centavos, method, status, provider_ref, created_at)
		VALUES (?, ?, ?, ?, 'paid', ?, ?)`, id, employerID, feeCentavos, method, ref, now.Unix()); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		SELECT p.user_id,'New shift available',j.role_label || ' in ' || j.city,'match',json_object('job_id', j.id),?
		FROM jobs j JOIN applicant_profiles p ON p.is_active=1 AND lower(trim(p.city))=lower(trim(j.city))
		JOIN applicant_roles ar ON ar.applicant_id=p.user_id AND ar.role_id=j.role_id
		WHERE j.id=? AND p.employment_type=j.employment_type
		AND ((j.timing='now' AND p.availability_mode='now') OR
			(j.timing='later' AND EXISTS (
				SELECT 1 FROM applicant_availability av
				WHERE av.applicant_id=p.user_id AND av.status='available'
				AND av.starts_at<=j.scheduled_at AND av.ends_at>=j.scheduled_at+8*3600
			)))`, now.Unix(), id); err != nil {
		return err
	}
	return tx.Commit()
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
			(j.timing='later' AND EXISTS (
				SELECT 1 FROM applicant_availability av
				WHERE av.applicant_id=p.user_id AND av.status='available'
				AND av.starts_at<=j.scheduled_at AND av.ends_at>=j.scheduled_at+8*3600
			)))
		AND NOT EXISTS (
			SELECT 1 FROM job_applications a
			WHERE a.job_id=j.id AND a.applicant_id=p.user_id
			AND a.status IN ('pending','accepted','rejected')
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

// ExpireSearches flips timed-out searches to 'unfulfilled'. Called by a sweeper and on reads.
func (s *Store) ExpireSearches(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `
		UPDATE jobs SET status='unfulfilled', updated_at = ?
		WHERE status='matching' AND search_expires_at IS NOT NULL AND search_expires_at <= ?`, now, now)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) SimulateSearchTimeout(ctx context.Context, employerID, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE jobs SET status='unfulfilled',updated_at=?,search_expires_at=? WHERE id=? AND employer_id=? AND status='matching'`, time.Now().Unix(), time.Now().Unix(), id, employerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return missOrBadState(ctx, s.db, employerID, id)
	}
	return nil
}

// AcceptJob is what a crew member's "Accept" will call (applicant side comes next).
// It is atomic: two workers can't both take the last slot, and nobody can accept after expiry.
func (s *Store) AcceptJob(ctx context.Context, jobID, applicantID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `
		UPDATE jobs SET accepted_count = accepted_count + 1,
			status = CASE WHEN accepted_count + 1 >= crew_needed THEN 'accepted' ELSE status END,
			updated_at = ?
		WHERE id = ? AND status = 'matching' AND search_expires_at > ? AND accepted_count < crew_needed`,
		now, jobID, now)
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
