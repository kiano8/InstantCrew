package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound             = errors.New("not found")
	ErrEmailTaken           = errors.New("email already registered")
	ErrHiringPassEmpty      = errors.New("hiring pass exhausted; add ₱10 for 5 hires")
	ErrHiringPassPending    = errors.New("your ₱10 receipt is awaiting verification; hires activate once accounting approves it")
	ErrBetaFull             = errors.New("beta tester limit reached")
	ErrNotConfirmed         = errors.New("the employer has not confirmed this booking yet")
	ErrPaymentNotVerified   = errors.New("accounting has not verified this payment yet")
	ErrPaymentRejected      = errors.New("this payment receipt was rejected; please upload a new proof of payment")
	ErrPaymentPendingExists = errors.New("a payment is already waiting for accounting verification")
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string // "employer" | "applicant"
	Name         string
	Company      string
}

type Store struct {
	db          *sql.DB
	idleTimeout time.Duration // 0 = no idle timeout
}

// SetSessionIdleTimeout must be called once at startup, before serving requests.
func (s *Store) SetSessionIdleTimeout(d time.Duration) { s.idleTimeout = d }

type RoleRow struct{ ID, Category, Label string }

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // simplest way to avoid SQLITE_BUSY with one writer
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) SyncRoles(ctx context.Context, roles []RoleRow) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, role := range roles {
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_roles(id,category,label,is_active) VALUES(?,?,?,1) ON CONFLICT(id) DO UPDATE SET category=excluded.category,label=excluded.label,is_active=1`, role.ID, role.Category, role.Label); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListRoles(ctx context.Context) ([]RoleRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,category,label FROM job_roles WHERE is_active=1 ORDER BY category,label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoleRow
	for rows.Next() {
		var r RoleRow
		if err = rows.Scan(&r.ID, &r.Category, &r.Label); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// migrate applies each migrations/*.sql file once, in filename order.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY, applied_at INTEGER NOT NULL)`); err != nil {
		return err
	}
	files, err := fs.ReadDir(migrationsFS, "migrations") // sorted by filename
	if err != nil {
		return err
	}
	for _, f := range files {
		var done int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, f.Name()).Scan(&done); err != nil {
			return err
		}
		if done > 0 {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + f.Name())
		if err != nil {
			return err
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", f.Name(), err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			f.Name(), time.Now().Unix()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) createUserRow(ctx context.Context, u *User) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, role, name, company, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		u.Email, u.PasswordHash, u.Role, u.Name, u.Company, time.Now().Unix())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrEmailTaken
		}
		return err
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.Company)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UserByEmailRole(ctx context.Context, email, role string) (*User, error) {
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, name, company
		 FROM users WHERE email = ? AND role = ?`, email, role))
}

func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID int64, expires time.Time) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, created_at, last_seen_at) VALUES (?, ?, ?, ?, ?)`,
		tokenHash, userID, expires.Unix(), now, now)
	return err
}

// idleCutoff returns the oldest last_seen_at that still counts as active.
func (s *Store) idleCutoff(now time.Time) int64 {
	if s.idleTimeout <= 0 {
		return 0
	}
	return now.Add(-s.idleTimeout).Unix()
}

// TouchSession records real user activity. It never revives a session that is
// already expired or idle, and writes at most once per 20 seconds per session.
func (s *Store) TouchSession(ctx context.Context, tokenHash string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ?
		 WHERE token_hash = ? AND expires_at > ? AND last_seen_at > ? AND last_seen_at < ?`,
		now.Unix(), tokenHash, now.Unix(), s.idleCutoff(now), now.Unix()-20)
	return err
}

// CreateUserCapped creates a user unless the beta already has max distinct
// people (distinct emails). A person adding a second role with an email that
// already exists is not a new tester, so it is always allowed. The check and
// the insert are one statement, so concurrent signups cannot exceed the cap.
// max <= 0 means no cap.
func (s *Store) createUserCappedRow(ctx context.Context, u *User, max int) error {
	if max <= 0 {
		return s.createUserRow(ctx, u)
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, role, name, company, created_at)
		 SELECT ?, ?, ?, ?, ?, ?
		 WHERE EXISTS (SELECT 1 FROM users WHERE email = ?)
		    OR (SELECT COUNT(DISTINCT email) FROM users) < ?`,
		u.Email, u.PasswordHash, u.Role, u.Name, u.Company, time.Now().Unix(),
		u.Email, max)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrEmailTaken
		}
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrBetaFull
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (*User, error) {
	now := time.Now()
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT u.id, u.email, u.password_hash, u.role, u.name, u.company
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > ? AND s.last_seen_at > ?`,
		tokenHash, now.Unix(), s.idleCutoff(now)))
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at <= ? OR last_seen_at <= ?`,
		now.Unix(), s.idleCutoff(now))
	return err
}
