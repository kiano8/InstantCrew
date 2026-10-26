package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	// ArrivalLimit is how long confirmed crew have to mark arrival on site.
	ArrivalLimit = time.Hour
	// NoShowRatingPenalty is subtracted from rating_hundredths (0.20 stars).
	NoShowRatingPenalty = 20
	// NoShowSuspendAfter strikes are tolerated; the next one suspends the account.
	NoShowSuspendAfter = 3
	// SuspensionLength is the temporary deactivation after too many no-shows.
	SuspensionLength = 5 * 24 * time.Hour
)

var (
	ErrAccountSuspended   = errors.New("your account is suspended for 5 days after repeated no-shows; you cannot accept shifts until it unlocks")
	ErrAccountDeactivated = errors.New("your account has been permanently deactivated after a no-show following a suspension")
	ErrArrivalExpired     = errors.New("the 1-hour arrival limit for this shift has expired")
)

// manila is used for dates shown in notifications.
var manila = time.FixedZone("PHT", 8*60*60)

// ApplicantStanding is the applicant's no-show record and any penalty in force.
type ApplicantStanding struct {
	NoShowCount                 int    `json:"no_show_count"`
	DeactivatedUntil            *int64 `json:"deactivated_until,omitempty"`
	Suspended                   bool   `json:"suspended"`
	ServedTemporaryDeactivation bool   `json:"served_temporary_deactivation"`
	PermanentlyDeactivated      bool   `json:"is_permanently_deactivated"`
}

func applicantStanding(ctx context.Context, q queryer, applicantID int64, now time.Time) (ApplicantStanding, error) {
	var st ApplicantStanding
	var until sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT no_show_count, deactivated_until, served_temporary_deactivation, is_permanently_deactivated
		FROM applicant_profiles WHERE user_id=$1`, applicantID).
		Scan(&st.NoShowCount, &until, &st.ServedTemporaryDeactivation, &st.PermanentlyDeactivated)
	if errors.Is(err, sql.ErrNoRows) {
		return st, ErrNotFound
	}
	if err != nil {
		return st, err
	}
	if until.Valid {
		v := until.Int64
		st.DeactivatedUntil = &v
		if v > now.Unix() {
			st.Suspended = !st.PermanentlyDeactivated
		} else {
			st.ServedTemporaryDeactivation = true
		}
	}
	return st, nil
}

// ApplicantStanding reports the applicant's no-show strikes and penalties.
func (s *Store) ApplicantStanding(ctx context.Context, applicantID int64) (ApplicantStanding, error) {
	return applicantStanding(ctx, s.db, applicantID, time.Now())
}

// noShowOutcome is the penalty applied by one no-show.
type noShowOutcome struct {
	Strikes        int
	SuspendedUntil *time.Time
	Permanent      bool
}

// recordNoShow adds a strike, lowers the rating by 0.20 stars and escalates:
// more than NoShowSuspendAfter strikes suspends the account for 5 days, and a
// no-show after that suspension has been served deactivates it permanently.
func recordNoShow(ctx context.Context, tx *sql.Tx, applicantID int64, now time.Time) (noShowOutcome, error) {
	var out noShowOutcome
	var until sql.NullInt64
	var served, permanent bool
	err := tx.QueryRowContext(ctx, `UPDATE applicant_profiles
		SET no_show_count=no_show_count+1, rating_hundredths=GREATEST(100, rating_hundredths-$2), updated_at=$3
		WHERE user_id=$1
		RETURNING no_show_count, deactivated_until, served_temporary_deactivation, is_permanently_deactivated`,
		applicantID, NoShowRatingPenalty, now.Unix()).Scan(&out.Strikes, &until, &served, &permanent)
	if err != nil {
		return out, err
	}
	served = served || (until.Valid && until.Int64 <= now.Unix())
	switch {
	case permanent:
		out.Permanent = true
	case served:
		if _, err := tx.ExecContext(ctx, `UPDATE applicant_profiles
			SET is_permanently_deactivated=true, served_temporary_deactivation=true WHERE user_id=$1`, applicantID); err != nil {
			return out, err
		}
		out.Permanent = true
	case !until.Valid && out.Strikes > NoShowSuspendAfter:
		t := now.Add(SuspensionLength)
		if _, err := tx.ExecContext(ctx, `UPDATE applicant_profiles SET deactivated_until=$2 WHERE user_id=$1`, applicantID, t.Unix()); err != nil {
			return out, err
		}
		out.SuspendedUntil = &t
	}
	return out, nil
}

// ExpireArrivals auto-cancels active shifts whose crew did not mark arrival
// within the arrival limit, and marks served suspensions. Safe to call often.
func (s *Store) ExpireArrivals(ctx context.Context) (int, error) {
	now := time.Now()
	if _, err := s.db.ExecContext(ctx, `UPDATE applicant_profiles SET served_temporary_deactivation=true
		WHERE deactivated_until IS NOT NULL AND deactivated_until <= $1 AND NOT served_temporary_deactivation`, now.Unix()); err != nil {
		return 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.job_id, a.applicant_id FROM job_assignments a JOIN jobs j ON j.id=a.job_id
		WHERE a.ended_at IS NULL AND a.transit_status IN ('none','en_route')
		AND a.arrival_deadline_at IS NOT NULL AND a.arrival_deadline_at <= $1 AND j.status='active'`, now.Unix())
	if err != nil {
		return 0, err
	}
	type overdue struct{ jobID, applicantID int64 }
	var list []overdue
	for rows.Next() {
		var o overdue
		if err := rows.Scan(&o.jobID, &o.applicantID); err != nil {
			rows.Close()
			return 0, err
		}
		list = append(list, o)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	count := 0
	for _, o := range list {
		ok, err := s.expireArrival(ctx, o.jobID, o.applicantID, now)
		if err != nil {
			return count, err
		}
		if ok {
			count++
		}
	}
	return count, nil
}

func (s *Store) expireArrival(ctx context.Context, jobID, applicantID int64, now time.Time) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// The guards make concurrent sweeps safe: only one can conclude the assignment.
	res, err := tx.ExecContext(ctx, `UPDATE job_assignments SET transit_status='no_show', ended_at=$1
		WHERE job_id=$2 AND applicant_id=$3 AND ended_at IS NULL AND transit_status IN ('none','en_route')
		AND arrival_deadline_at <= $1`, now.Unix(), jobID, applicantID)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}
	var employerID int64
	var roleLabel string
	err = tx.QueryRowContext(ctx, `UPDATE jobs SET status='cancelled', updated_at=$1 WHERE id=$2 AND status='active'
		RETURNING employer_id, role_label`, now.Unix(), jobID).Scan(&employerID, &roleLabel)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	outcome, err := recordNoShow(ctx, tx, applicantID, now)
	if err != nil {
		return false, err
	}
	var crewName string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM users WHERE id=$1`, applicantID).Scan(&crewName); err != nil {
		return false, err
	}

	crewMsg := fmt.Sprintf("You did not mark arrival within the 1-hour limit for %s. The shift was auto-cancelled and your rating was lowered by 0.20 stars (no-show strike %d).", roleLabel, outcome.Strikes)
	switch {
	case outcome.Permanent:
		crewMsg += " Because this happened after a 5-day suspension, your account has been permanently deactivated."
	case outcome.SuspendedUntil != nil:
		crewMsg += fmt.Sprintf(" You have more than %d no-shows, so your account is suspended until %s.", NoShowSuspendAfter, outcome.SuspendedUntil.In(manila).Format("Jan 2, 2006 3:04 PM"))
	}
	empMsg := fmt.Sprintf("%s did not mark arrival within the 1-hour limit for %s, so the shift was auto-cancelled and a no-show penalty was applied to the crew member.", crewName, roleLabel)

	for _, n := range []struct {
		userID         int64
		title, message string
	}{
		{applicantID, "Shift auto-cancelled: no-show penalty", crewMsg},
		{employerID, "Shift auto-cancelled: crew did not arrive", empMsg},
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
			VALUES($1,$2,$3,'no_show',json_build_object('job_id',$4::bigint)::text,$5)`,
			n.userID, n.title, n.message, jobID, now.Unix()); err != nil {
			return false, err
		}
	}
	return true, tx.Commit()
}
