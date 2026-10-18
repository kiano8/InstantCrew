// Package store is the SQLite persistence layer for the GCash verifier.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	mrand "math/rand/v2"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var (
	ErrNotFound  = errors.New("not found")
	ErrNoMatch   = errors.New("no matching pending payment")
	ErrDuplicate = errors.New("duplicate")
)

type Store struct{ db *sql.DB }

type Payment struct {
	ID           string     `json:"id"`
	Label        string     `json:"label"`
	BaseCentavos int64      `json:"baseCentavos"`
	PayCentavos  int64      `json:"payCentavos"`
	PayAmount    string     `json:"payAmount"` // exact amount to send, e.g. "100.37"
	Status       string     `json:"status"`    // pending | paid | expired
	GcashRef     string     `json:"gcashRef,omitempty"`
	PayerNumber  string     `json:"payerNumber,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	PaidAt       *time.Time `json:"paidAt,omitempty"`
}

type SMSLog struct {
	ID             int64     `json:"id"`
	MessageID      string    `json:"messageId,omitempty"`
	Sender         string    `json:"sender"`
	Body           string    `json:"body"`
	Raw            string    `json:"-"`
	SigOK          bool      `json:"sigOk"`
	AmountCentavos int64     `json:"amountCentavos,omitempty"`
	Ref            string    `json:"ref,omitempty"`
	Result         string    `json:"result"`
	Note           string    `json:"note,omitempty"`
	PaymentID      string    `json:"paymentId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

func FormatPesos(centavos int64) string {
	return fmt.Sprintf("%d.%02d", centavos/100, centavos%100)
}

func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: one writer; keeps things simple and safe
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY, applied_at INTEGER NOT NULL)`); err != nil {
		return err
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, e := range entries { // ReadDir returns entries sorted by filename
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, e.Name()).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		sqlText, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return err
		}
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(sqlText)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", e.Name(), err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)`, e.Name(), time.Now().Unix()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

// ---- payments ----

const paymentCols = `id, label, base_centavos, pay_centavos, status, gcash_ref, payer_number, created_at, expires_at, paid_at`

type scanner interface{ Scan(dest ...any) error }

func scanPayment(row scanner) (*Payment, error) {
	var p Payment
	var ref, payer sql.NullString
	var created, expires int64
	var paid sql.NullInt64
	err := row.Scan(&p.ID, &p.Label, &p.BaseCentavos, &p.PayCentavos, &p.Status, &ref, &payer, &created, &expires, &paid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p.GcashRef = ref.String
	p.PayerNumber = payer.String
	p.CreatedAt = time.Unix(created, 0)
	p.ExpiresAt = time.Unix(expires, 0)
	if paid.Valid {
		t := time.Unix(paid.Int64, 0)
		p.PaidAt = &t
	}
	p.PayAmount = FormatPesos(p.PayCentavos)
	return &p, nil
}

func (s *Store) expireOld(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE payments SET status = 'expired' WHERE status = 'pending' AND expires_at < ?`, time.Now().Unix())
	return err
}

// CreatePayment creates a pending payment whose exact amount (base + 1..99
// centavos) is unique among all pending payments.
func (s *Store) CreatePayment(ctx context.Context, label string, baseCentavos int64, ttl time.Duration) (*Payment, error) {
	if err := s.expireOld(ctx); err != nil {
		return nil, err
	}
	now := time.Now()
	id := newID("pay_")
	for attempt := 0; attempt < 150; attempt++ {
		pay := baseCentavos + int64(mrand.IntN(99)+1)
		_, err := s.db.ExecContext(ctx, `INSERT INTO payments
			(id, label, base_centavos, pay_centavos, status, created_at, expires_at)
			VALUES (?, ?, ?, ?, 'pending', ?, ?)`,
			id, label, baseCentavos, pay, now.Unix(), now.Add(ttl).Unix())
		if err == nil {
			return s.GetPayment(ctx, id)
		}
		if isUnique(err) {
			continue // that exact amount is taken by another pending payment
		}
		return nil, err
	}
	return nil, errors.New("too many pending payments at this amount; try again shortly")
}

func (s *Store) GetPayment(ctx context.Context, id string) (*Payment, error) {
	if err := s.expireOld(ctx); err != nil {
		return nil, err
	}
	return scanPayment(s.db.QueryRowContext(ctx, `SELECT `+paymentCols+` FROM payments WHERE id = ?`, id))
}

func (s *Store) ListPayments(ctx context.Context, limit int) ([]*Payment, error) {
	if err := s.expireOld(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+paymentCols+` FROM payments ORDER BY created_at DESC, rowid DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Payment{}
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MarkPaid settles the pending, unexpired payment whose exact amount equals
// payCentavos. Returns ErrNoMatch if there is none, ErrDuplicate if the GCash
// reference was already used.
func (s *Store) MarkPaid(ctx context.Context, payCentavos int64, ref, payer string, receivedAt time.Time) (*Payment, error) {
	// An SMS can only settle a payment that already existed when the SMS arrived
	// (5 min clock-skew allowance). This stops old messages being replayed against
	// new payments that happen to reuse the same exact amount. Zero = skip the check.
	createdLimit := int64(1 << 62)
	if !receivedAt.IsZero() {
		createdLimit = receivedAt.Unix() + 300
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	var id string
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM payments WHERE status = 'pending' AND pay_centavos = ? AND expires_at >= ? AND created_at <= ?`,
		payCentavos, now, createdLimit).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoMatch
	}
	if err != nil {
		return nil, err
	}

	var refArg, payerArg any
	if ref != "" {
		refArg = ref
	}
	if payer != "" {
		payerArg = payer
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE payments SET status = 'paid', gcash_ref = ?, payer_number = ?, paid_at = ? WHERE id = ?`,
		refArg, payerArg, now, id); err != nil {
		if isUnique(err) {
			return nil, ErrDuplicate
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetPayment(ctx, id)
}

func (s *Store) RefAlreadyUsed(ctx context.Context, ref string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments WHERE gcash_ref = ?`, ref).Scan(&n)
	return n > 0, err
}

// ---- sms log ----

// InsertSMS records an incoming message. Returns ErrDuplicate if a message
// with the same messageId was already recorded (webhook retry).
func (s *Store) InsertSMS(ctx context.Context, l *SMSLog) (int64, error) {
	var mid any
	if l.MessageID != "" {
		mid = l.MessageID
	}
	sig := 0
	if l.SigOK {
		sig = 1
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO sms_log
		(message_id, sender, body, raw, sig_ok, result, created_at)
		VALUES (?, ?, ?, ?, ?, 'received', ?)`,
		mid, l.Sender, l.Body, l.Raw, sig, time.Now().Unix())
	if err != nil {
		if isUnique(err) {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateSMSResult(ctx context.Context, id int64, result, note, paymentID string, amountCentavos int64, ref string) error {
	var amt, refArg, pid any
	if amountCentavos > 0 {
		amt = amountCentavos
	}
	if ref != "" {
		refArg = ref
	}
	if paymentID != "" {
		pid = paymentID
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE sms_log SET result = ?, note = ?, payment_id = ?, amount_centavos = ?, ref = ? WHERE id = ?`,
		result, note, pid, amt, refArg, id)
	return err
}

func (s *Store) ListSMS(ctx context.Context, limit int) ([]*SMSLog, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, COALESCE(message_id,''), sender, body, sig_ok,
		COALESCE(amount_centavos,0), COALESCE(ref,''), result, note, COALESCE(payment_id,''), created_at
		FROM sms_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*SMSLog{}
	for rows.Next() {
		var l SMSLog
		var sig int
		var created int64
		if err := rows.Scan(&l.ID, &l.MessageID, &l.Sender, &l.Body, &sig, &l.AmountCentavos,
			&l.Ref, &l.Result, &l.Note, &l.PaymentID, &created); err != nil {
			return nil, err
		}
		l.SigOK = sig == 1
		l.CreatedAt = time.Unix(created, 0)
		out = append(out, &l)
	}
	return out, rows.Err()
}
