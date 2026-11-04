package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// ShiftRewardAfter is how many completed shifts a new crew member needs for the reward.
	ShiftRewardAfter = 5
	// ShiftRewardHires is how many free hires that reward gives.
	ShiftRewardHires = 5
)

// ErrReferralCode is returned for a referral code that matches no crew member.
var ErrReferralCode = errors.New("that referral code is not valid")

// ReferralCode is a crew member's shareable promo code, derived from their ID.
// shortcut: codes are guessable (IC + base36 ID); move to random stored codes if referral rewards get farmed.
func ReferralCode(userID int64) string {
	return "IC" + strings.ToUpper(strconv.FormatInt(userID, 36))
}

// ReferrerByCode returns the crew member who owns code, or ErrReferralCode.
func (s *Store) ReferrerByCode(ctx context.Context, code string) (int64, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	id, err := strconv.ParseInt(strings.TrimPrefix(code, "IC"), 36, 64)
	if !strings.HasPrefix(code, "IC") || err != nil || id <= 0 {
		return 0, ErrReferralCode
	}
	var ok bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id=$1 AND role='applicant')`, id).Scan(&ok); err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrReferralCode
	}
	return id, nil
}

// CreditReferral links a new crew member to their referrer and gives the
// referrer 1 hire. It does nothing if the new user was already referred.
func (s *Store) CreditReferral(ctx context.Context, newUserID, referrerID int64, newUserName string) error {
	if newUserID == referrerID {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `UPDATE users SET referred_by=$1 WHERE id=$2 AND referred_by IS NULL`, referrerID, newUserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	if err := addRewardHires(ctx, tx, referrerID, 1, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		VALUES($1,'Referral reward',$2,'payment','{}',$3)`,
		referrerID, fmt.Sprintf("%s signed up with your referral code. You earned 1 free hire.", newUserName), now); err != nil {
		return err
	}
	return tx.Commit()
}

// addRewardHires adds free hires. Rewards may take a crew member past MaxHeldHires.
func addRewardHires(ctx context.Context, tx *sql.Tx, applicantID int64, hires int, now int64) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO applicant_hiring_passes(applicant_id,total_hires,remaining_hires,used_hires,amount_centavos,batches_paid,updated_at)
		VALUES($1,$2,$2,0,0,0,$3)
		ON CONFLICT (applicant_id) DO UPDATE SET
			total_hires=applicant_hiring_passes.total_hires+$2,
			remaining_hires=applicant_hiring_passes.remaining_hires+$2, updated_at=$3`,
		applicantID, hires, now)
	return err
}

// grantShiftRewards gives ShiftRewardHires once to each crew member on jobID
// who has now completed ShiftRewardAfter shifts. Call it in the EndJob
// transaction after the job is marked completed. A completed shift is a
// confirmed booking on a completed job that was not a no-show or refunded.
func grantShiftRewards(ctx context.Context, tx *sql.Tx, jobID, now int64) error {
	_, err := tx.ExecContext(ctx, `
		WITH rewarded AS (
			UPDATE users u SET shift_reward_granted=true
			WHERE NOT u.shift_reward_granted
			  AND u.id IN (SELECT applicant_id FROM job_assignments WHERE job_id=$1)
			  AND (SELECT COUNT(*) FROM job_assignments a JOIN jobs j ON j.id=a.job_id
			       WHERE a.applicant_id=u.id AND j.status='completed' AND a.arrival_deadline_at IS NOT NULL
			         AND a.transit_status<>'no_show' AND NOT a.hire_refunded) >= $2
			RETURNING u.id
		), pass AS (
			UPDATE applicant_hiring_passes SET total_hires=total_hires+$3, remaining_hires=remaining_hires+$3, updated_at=$4
			WHERE applicant_id IN (SELECT id FROM rewarded)
		)
		INSERT INTO notifications(user_id,title,message,type,payload,created_at)
		SELECT id,'Reward unlocked',$5,'payment','{}',$4 FROM rewarded`,
		jobID, ShiftRewardAfter, ShiftRewardHires, now,
		fmt.Sprintf("You completed your first %d shifts! You earned %d free hires.", ShiftRewardAfter, ShiftRewardHires))
	return err
}
