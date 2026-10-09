package store

import (
	"context"
	"database/sql"
	"time"
)

type ApplicantProfile struct {
	EmploymentType string   `json:"employment_type"`
	ExpectedRate   int64    `json:"expected_rate"`
	RatePeriod     string   `json:"rate_period"`
	City           string   `json:"city"`
	Location       string   `json:"location_address"`
	MapsURL        string   `json:"maps_url"`
	Lat            *float64 `json:"lat"`
	Lng            *float64 `json:"lng"`
	Availability   string   `json:"availability_mode"`
	Active         *bool    `json:"is_active"`
	Roles          []string `json:"roles"`
}

type Availability struct {
	ID       int64  `json:"id"`
	StartsAt int64  `json:"starts_at"`
	EndsAt   int64  `json:"ends_at"`
	Status   string `json:"status"`
}

type Notification struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	Payload   string `json:"payload"`
	CreatedAt int64  `json:"created_at"`
	Read      bool   `json:"read"`
}

func (s *Store) Notifications(ctx context.Context, userID int64) ([]Notification, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,message,type,payload,created_at,read_at FROM notifications WHERE user_id=? ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		var n Notification
		var read sql.NullInt64
		if err = rows.Scan(&n.ID, &n.Title, &n.Message, &n.Type, &n.Payload, &n.CreatedAt, &read); err != nil {
			return nil, err
		}
		n.Read = read.Valid
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) AddNotification(ctx context.Context, userID int64, title, message, kind, payload string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at) VALUES(?,?,?,?,?,?)`, userID, title, message, kind, payload, time.Now().Unix())
	return err
}

func (s *Store) NotifyJobEmployer(ctx context.Context, jobID int64, title, message, kind string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at) SELECT employer_id,?,?,?,?,? FROM jobs WHERE id=?`, title, message, kind, "{}", time.Now().Unix(), jobID)
	return err
}

func (s *Store) MarkNotificationsRead(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notifications SET read_at=? WHERE user_id=? AND read_at IS NULL`, time.Now().Unix(), userID)
	return err
}
func (s *Store) ClearNotifications(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE user_id=?`, userID)
	return err
}

func (s *Store) UpdateApplicantTransit(ctx context.Context, jobID, applicantID int64, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE job_assignments SET transit_status=? WHERE job_id=? AND applicant_id=? AND ended_at IS NULL`, status, jobID, applicantID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) EndApplicantAssignment(ctx context.Context, jobID, applicantID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `UPDATE job_assignments SET ended_at=? WHERE job_id=? AND applicant_id=? AND ended_at IS NULL`, now, jobID, applicantID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	var left int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM job_assignments WHERE job_id=? AND ended_at IS NULL`, jobID).Scan(&left); err != nil {
		return err
	}
	if left == 0 {
		_, err = tx.ExecContext(ctx, `UPDATE jobs SET status='completed',updated_at=? WHERE id=? AND status='active'`, now, jobID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ApplicantAvailability(ctx context.Context, id int64) ([]Availability, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,starts_at,ends_at,status FROM applicant_availability WHERE applicant_id=? ORDER BY starts_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Availability
	for rows.Next() {
		var a Availability
		if err = rows.Scan(&a.ID, &a.StartsAt, &a.EndsAt, &a.Status); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (s *Store) SaveApplicantAvailability(ctx context.Context, id int64, items []Availability) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM applicant_availability WHERE applicant_id=?`, id); err != nil {
		return err
	}
	for _, a := range items {
		if _, err = tx.ExecContext(ctx, `INSERT INTO applicant_availability(applicant_id,starts_at,ends_at,status,created_at,updated_at) VALUES(?,?,?,?,?,?)`, id, a.StartsAt, a.EndsAt, a.Status, time.Now().Unix(), time.Now().Unix()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ApplicantProfile(ctx context.Context, id int64) (*ApplicantProfile, error) {
	p := &ApplicantProfile{}
	var active int
	err := s.db.QueryRowContext(ctx, `SELECT employment_type, expected_rate, rate_period, city,
		COALESCE(location_address,''), COALESCE(maps_url,''), lat, lng, availability_mode,is_active
		FROM applicant_profiles WHERE user_id=?`, id).Scan(&p.EmploymentType, &p.ExpectedRate,
		&p.RatePeriod, &p.City, &p.Location, &p.MapsURL, &p.Lat, &p.Lng, &p.Availability, &active)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	b := active != 0
	p.Active = &b
	rows, err := s.db.QueryContext(ctx, `SELECT role_id FROM applicant_roles WHERE applicant_id=? ORDER BY is_primary DESC, role_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		if err = rows.Scan(&role); err != nil {
			return nil, err
		}
		p.Roles = append(p.Roles, role)
	}
	return p, rows.Err()
}

func (s *Store) SaveApplicantProfile(ctx context.Context, id int64, p ApplicantProfile) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	active := 1
	if p.Active != nil && !*p.Active {
		active = 0
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO applicant_profiles(user_id,employment_type,expected_rate,rate_period,city,location_address,maps_url,lat,lng,availability_mode,is_active,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET employment_type=excluded.employment_type,expected_rate=excluded.expected_rate,rate_period=excluded.rate_period,city=excluded.city,location_address=excluded.location_address,maps_url=excluded.maps_url,lat=excluded.lat,lng=excluded.lng,availability_mode=excluded.availability_mode,is_active=excluded.is_active,updated_at=excluded.updated_at`,
		id, p.EmploymentType, p.ExpectedRate, p.RatePeriod, p.City, p.Location, p.MapsURL, p.Lat, p.Lng, p.Availability, active, now, now)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM applicant_roles WHERE applicant_id=?`, id); err != nil {
		return err
	}
	for i, role := range p.Roles {
		if _, err = tx.ExecContext(ctx, `INSERT INTO applicant_roles(applicant_id,role_id,is_primary) VALUES(?,?,?)`, id, role, i == 0); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ApplicantJobs(ctx context.Context, applicantID int64) ([]*Job, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+jobCols+` FROM jobs j
		WHERE j.status='matching' AND (j.search_expires_at IS NULL OR j.search_expires_at>?)
		AND EXISTS(SELECT 1 FROM applicant_profiles p JOIN applicant_roles ar ON ar.applicant_id=p.user_id
			WHERE p.user_id=? AND p.is_active=1 AND p.employment_type=j.employment_type AND lower(trim(p.city))=lower(trim(j.city)) AND ar.role_id=j.role_id
			AND ((j.timing='now' AND p.availability_mode='now') OR
				(j.timing='later' AND EXISTS (
					SELECT 1 FROM applicant_availability av
					WHERE av.applicant_id=p.user_id AND av.status='available'
					AND av.starts_at<=j.scheduled_at AND av.ends_at>=j.scheduled_at+8*3600
				))))
		AND NOT EXISTS(SELECT 1 FROM job_applications a WHERE a.job_id=j.id AND a.applicant_id=? AND a.status IN ('pending','accepted','rejected'))
		ORDER BY j.created_at DESC`, time.Now().Unix(), applicantID, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Job
	for rows.Next() {
		j, e := scanJob(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// ApplicantJob returns a single open offer only when the applicant profile matches it.
func (s *Store) ApplicantJob(ctx context.Context, applicantID, jobID int64) (*Job, error) {
	return scanJob(s.db.QueryRowContext(ctx, `SELECT `+jobCols+` FROM jobs j
		WHERE j.id=? AND j.status='matching' AND (j.search_expires_at IS NULL OR j.search_expires_at>?)
		AND EXISTS(SELECT 1 FROM applicant_profiles p JOIN applicant_roles ar ON ar.applicant_id=p.user_id
			WHERE p.user_id=? AND p.is_active=1 AND p.employment_type=j.employment_type
			AND lower(trim(p.city))=lower(trim(j.city)) AND ar.role_id=j.role_id
			AND ((j.timing='now' AND p.availability_mode='now') OR
				(j.timing='later' AND EXISTS (SELECT 1 FROM applicant_availability av
					WHERE av.applicant_id=p.user_id AND av.status='available'
					AND av.starts_at<=j.scheduled_at AND av.ends_at>=j.scheduled_at+8*3600))))`,
		jobID, time.Now().Unix(), applicantID))
}

func (s *Store) ApplyForJob(ctx context.Context, jobID, applicantID int64) error {
	res, err := s.db.ExecContext(ctx, `INSERT INTO job_applications(job_id,applicant_id,status) SELECT id,?,'pending' FROM jobs WHERE id=? AND status='matching' AND (search_expires_at IS NULL OR search_expires_at>?)`, applicantID, jobID, time.Now().Unix())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrBadState
	}
	return nil
}

func (s *Store) RejectJob(ctx context.Context, jobID, applicantID int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO job_applications(job_id,applicant_id,status,applied_at,updated_at) SELECT id,?,'rejected',?,? FROM jobs WHERE id=? AND status='matching' ON CONFLICT(job_id,applicant_id) DO UPDATE SET status='rejected',updated_at=excluded.updated_at`, applicantID, time.Now().Unix(), time.Now().Unix(), jobID)
	if err != nil {
		return err
	}
	var exists int
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM job_applications WHERE job_id=? AND applicant_id=? AND status='rejected'`, jobID, applicantID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return ErrBadState
	}
	return nil
}

func (s *Store) ApplicantAssignments(ctx context.Context, applicantID int64) ([]*Job, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+jobCols+` FROM jobs j JOIN job_assignments a ON a.job_id=j.id WHERE a.applicant_id=? ORDER BY a.accepted_at DESC`, applicantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Job
	for rows.Next() {
		j, e := scanJob(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) ApplicantAssignmentState(ctx context.Context, applicantID, jobID int64) (Assignment, error) {
	var a Assignment
	var accepted int64
	var ended sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT a.applicant_id,u.name,a.accepted_at,a.ended_at,a.transit_status FROM job_assignments a JOIN users u ON u.id=a.applicant_id WHERE a.applicant_id=? AND a.job_id=?`, applicantID, jobID).Scan(&a.ApplicantID, &a.Name, &accepted, &ended, &a.Transit)
	if err == sql.ErrNoRows {
		return a, ErrNotFound
	}
	if err != nil {
		return a, err
	}
	a.AcceptedAt = time.Unix(accepted, 0)
	if ended.Valid {
		t := time.Unix(ended.Int64, 0)
		a.EndedAt = &t
	}
	return a, nil
}
