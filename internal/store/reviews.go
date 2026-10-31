package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// finishedContracts joins each job j to the crew member a who worked it: the one
// confirmed (an arrival deadline is set on confirmation) who didn't no-show.
// Callers add j.status='completed'.
const finishedContracts = `jobs j JOIN job_assignments a ON a.job_id=j.id
	AND a.arrival_deadline_at IS NOT NULL AND a.transit_status <> 'no_show'`

// PendingReview is a finished contract the user hasn't rated or skipped yet.
type PendingReview struct {
	JobID     int64  `json:"job_id"`
	RoleLabel string `json:"role_label"`
	Name      string `json:"name"` // the person to rate: crew member, or employer's company/name
	Rates     string `json:"rates"`
}

// PendingReviews lists the finished contracts userID still has to rate.
func (s *Store) PendingReviews(ctx context.Context, userID int64) ([]PendingReview, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT j.id, j.role_label, u.name, 'crew' FROM `+finishedContracts+`
		JOIN users u ON u.id=a.applicant_id
		WHERE j.status='completed' AND j.employer_id=$1
		AND NOT EXISTS (SELECT 1 FROM job_reviews r WHERE r.job_id=j.id AND r.reviewer_id=$1)
		UNION ALL
		SELECT j.id, j.role_label, COALESCE(NULLIF(e.company,''), e.name), 'employer' FROM `+finishedContracts+`
		JOIN users e ON e.id=j.employer_id
		WHERE j.status='completed' AND a.applicant_id=$1
		AND NOT EXISTS (SELECT 1 FROM job_reviews r WHERE r.job_id=j.id AND r.reviewer_id=$1)
		ORDER BY 1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PendingReview{}
	for rows.Next() {
		var p PendingReview
		if err := rows.Scan(&p.JobID, &p.RoleLabel, &p.Name, &p.Rates); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SubmitReview records reviewerID's rating of the other side of a finished
// contract; stars 0 skips it for good. A crew member's stars from the employer
// move their rating: 5 stars +0.10, 4 +0.05, 3 none, 2 -0.05, 1 -0.10.
// Rating a contract twice changes nothing.
func (s *Store) SubmitReview(ctx context.Context, reviewerID, jobID int64, stars int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var employerID, crewID int64
	err = tx.QueryRowContext(ctx, `SELECT j.employer_id, a.applicant_id FROM `+finishedContracts+`
		WHERE j.status='completed' AND j.id=$1 AND (j.employer_id=$2 OR a.applicant_id=$2)`, jobID, reviewerID).Scan(&employerID, &crewID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	reviewee := crewID
	if reviewerID == crewID {
		reviewee = employerID
	}
	var rated sql.NullInt64
	if stars > 0 {
		rated = sql.NullInt64{Int64: int64(stars), Valid: true}
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO job_reviews(job_id,reviewer_id,reviewee_id,stars,created_at)
		VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, jobID, reviewerID, reviewee, rated, time.Now().Unix())
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 && rated.Valid && reviewee == crewID {
		if _, err := tx.ExecContext(ctx, `UPDATE applicant_profiles
			SET rating_hundredths=LEAST(500, GREATEST(100, rating_hundredths+($1-3)*5)) WHERE user_id=$2`, stars, crewID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// EmployerRating is the average of the stars crew gave an employer.
type EmployerRating struct {
	Rating  float64 `json:"rating"` // 0 when there are no reviews yet
	Reviews int     `json:"reviews"`
}

// EmployerRatings returns the review averages of the given employers.
func (s *Store) EmployerRatings(ctx context.Context, employerIDs []int64) (map[int64]EmployerRating, error) {
	out := map[int64]EmployerRating{}
	rows, err := s.db.QueryContext(ctx, `SELECT reviewee_id, AVG(stars)::float8, COUNT(stars)
		FROM job_reviews WHERE reviewee_id = ANY($1) AND stars IS NOT NULL GROUP BY reviewee_id`, employerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var r EmployerRating
		if err := rows.Scan(&id, &r.Rating, &r.Reviews); err != nil {
			return nil, err
		}
		out[id] = r
	}
	return out, rows.Err()
}
