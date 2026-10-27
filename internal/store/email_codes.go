package store

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrCodeTooSoon = errors.New("please wait a minute before requesting another code")
	ErrCodeInvalid = errors.New("that code is incorrect or has expired; request a new one")
)

// maxCodeAttempts wrong guesses burn a code, so a 6-digit code can't be brute-forced.
const maxCodeAttempts = 5

// SaveEmailCode stores codeHash for (email, purpose), replacing any earlier code,
// unless the earlier one was issued less than minInterval ago.
func (s *Store) SaveEmailCode(ctx context.Context, email, purpose, codeHash string, ttl, minInterval time.Duration) error {
	now := time.Now()
	res, err := s.db.ExecContext(ctx, `INSERT INTO email_codes(email,purpose,code_hash,attempts,expires_at,created_at) VALUES($1,$2,$3,0,$4,$5)
		ON CONFLICT (email,purpose) DO UPDATE SET code_hash=EXCLUDED.code_hash, attempts=0, expires_at=EXCLUDED.expires_at, created_at=EXCLUDED.created_at
		WHERE email_codes.created_at <= $6`,
		email, purpose, codeHash, now.Add(ttl).Unix(), now.Unix(), now.Add(-minInterval).Unix())
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCodeTooSoon
	}
	return nil
}

// DeleteEmailCode drops a code, e.g. when the email carrying it could not be sent.
func (s *Store) DeleteEmailCode(ctx context.Context, email, purpose string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM email_codes WHERE email=$1 AND purpose=$2`, email, purpose)
	return err
}

// UseEmailCode consumes the code if codeHash matches; otherwise it counts a failed attempt.
func (s *Store) UseEmailCode(ctx context.Context, email, purpose, codeHash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var stored string
	var attempts int
	var expires int64
	err = tx.QueryRowContext(ctx, `SELECT code_hash,attempts,expires_at FROM email_codes WHERE email=$1 AND purpose=$2 FOR UPDATE`,
		email, purpose).Scan(&stored, &attempts, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrCodeInvalid
	}
	if err != nil {
		return err
	}
	if expires <= time.Now().Unix() || attempts >= maxCodeAttempts {
		if _, err := tx.ExecContext(ctx, `DELETE FROM email_codes WHERE email=$1 AND purpose=$2`, email, purpose); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrCodeInvalid
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(codeHash)) != 1 {
		if _, err := tx.ExecContext(ctx, `UPDATE email_codes SET attempts=attempts+1 WHERE email=$1 AND purpose=$2`, email, purpose); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrCodeInvalid
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM email_codes WHERE email=$1 AND purpose=$2`, email, purpose); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteExpiredEmailCodes(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM email_codes WHERE expires_at <= $1`, time.Now().Unix())
	return err
}

// ResetPassword sets a new password hash and signs the user out everywhere.
func (s *Store) ResetPassword(ctx context.Context, userID int64, passwordHash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=$1 WHERE id=$2`, passwordHash, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1`, userID); err != nil {
		return err
	}
	return tx.Commit()
}
