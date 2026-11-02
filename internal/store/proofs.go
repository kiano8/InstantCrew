package store

import (
	"context"
	"errors"
	"time"
)

// ErrProofInvalid: the receipt isn't this user's, or a payment already used it.
var ErrProofInvalid = errors.New("this receipt can't be used; please upload your screenshot again")

// SaveProofUpload records who uploaded a receipt.
func (s *Store) SaveProofUpload(ctx context.Context, path string, userID int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO proof_uploads(path,user_id,created_at) VALUES($1,$2,$3)`, path, userID, time.Now().Unix())
	return err
}

// RecentProofUploads counts this user's uploads since the given time.
func (s *Store) RecentProofUploads(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM proof_uploads WHERE user_id=$1 AND created_at>=$2`, userID, since.Unix()).Scan(&n)
	return n, err
}

// ClaimProof marks the user's own upload as used, atomically, so one receipt can't
// back two payments. A refused payment keeps its claim: the pages upload afresh on each try.
func (s *Store) ClaimProof(ctx context.Context, path string, userID int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE proof_uploads SET used_at=$1 WHERE path=$2 AND user_id=$3 AND used_at IS NULL`, time.Now().Unix(), path, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrProofInvalid
	}
	return nil
}

// CanViewProof reports whether the user uploaded or paid with this receipt. The
// payment tables cover receipts from before proof_uploads existed.
func (s *Store) CanViewProof(ctx context.Context, path string, userID int64) (bool, error) {
	var ok bool
	err := s.db.QueryRowContext(ctx, `SELECT
		EXISTS(SELECT 1 FROM proof_uploads WHERE path=$1 AND user_id=$2)
		OR EXISTS(SELECT 1 FROM payments WHERE proof_image=$1 AND user_id=$2)
		OR EXISTS(SELECT 1 FROM applicant_hiring_pass_payments WHERE proof_image=$1 AND applicant_id=$2)`, path, userID).Scan(&ok)
	return ok, err
}
