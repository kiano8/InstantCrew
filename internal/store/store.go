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

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrNotFound             = errors.New("not found")
	ErrEmailTaken           = errors.New("email already registered")
	ErrHiringPassEmpty      = errors.New("no hires left; add hires at ₱2 each (up to 5)")
	ErrHiringPassPending    = errors.New("your hiring pass receipt is awaiting verification; hires activate once accounting approves it")
	ErrHiringPassFull       = errors.New("you can hold at most 5 hires at a time")
	ErrBetaFull             = errors.New("beta tester limit reached")
	ErrNotConfirmed         = errors.New("the employer has not confirmed this booking yet")
	ErrBookingPending       = errors.New("the employer is paying the search fee for this booking; you can leave it once it is confirmed or declined")
	ErrPaymentNotVerified   = errors.New("accounting has not verified this payment yet")
	ErrPaymentRejected      = errors.New("this payment receipt was rejected; please upload a new proof of payment")
	ErrPaymentPendingExists = errors.New("a payment is already waiting for accounting verification")
	ErrAlreadyVerified      = errors.New("this payment is already verified and can no longer be changed")
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

func Open(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	// Configure connection pool for PostgreSQL
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)
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
	// Roles dropped from the catalog are deactivated, not deleted: jobs and
	// applicant_roles still reference their IDs.
	if _, err := tx.ExecContext(ctx, `UPDATE job_roles SET is_active=false`); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_roles(id,category,label,is_active) VALUES($1,$2,$3,true) ON CONFLICT(id) DO UPDATE SET category=excluded.category,label=excluded.label,is_active=true`, role.ID, role.Category, role.Label); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListRoles(ctx context.Context) ([]RoleRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,category,label FROM job_roles WHERE is_active=true ORDER BY category,label`)
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
		version TEXT PRIMARY KEY, applied_at BIGINT NOT NULL)`); err != nil {
		return err
	}
	files, err := fs.ReadDir(migrationsFS, "migrations") // sorted by filename
	if err != nil {
		return err
	}
	for _, f := range files {
		var done int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = $1`, f.Name()).Scan(&done); err != nil {
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
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)`,
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
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, role, name, company, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		u.Email, u.PasswordHash, u.Role, u.Name, u.Company, time.Now().Unix()).Scan(&u.ID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrEmailTaken
		}
		return err
	}
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
		 FROM users WHERE email = $1 AND role = $2`, email, role))
}

func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID int64, expires time.Time) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, created_at, last_seen_at) VALUES ($1, $2, $3, $4, $5)`,
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
		`UPDATE sessions SET last_seen_at = $1
		 WHERE token_hash = $2 AND expires_at > $3 AND last_seen_at > $4 AND last_seen_at < $5`,
		now.Unix(), tokenHash, now.Unix(), s.idleCutoff(now), now.Unix()-20)
	return err
}

// CreateUserCapped creates a user unless the beta already has max distinct
// people (distinct emails). An email that already exists is not a new tester,
// so it never counts as full and hits the duplicate check instead. The check
// and the insert are one statement, so concurrent signups cannot exceed the cap.
// max <= 0 means no cap.
func (s *Store) createUserCappedRow(ctx context.Context, u *User, max int) error {
	if max <= 0 {
		return s.createUserRow(ctx, u)
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, role, name, company, created_at)
		 SELECT $1, $2, $3, $4, $5, $6
		 WHERE EXISTS (SELECT 1 FROM users WHERE email = $7)
		    OR (SELECT COUNT(DISTINCT email) FROM users) < $8
		 RETURNING id`,
		u.Email, u.PasswordHash, u.Role, u.Name, u.Company, time.Now().Unix(),
		u.Email, max).Scan(&u.ID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrEmailTaken
		}
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBetaFull
		}
		return err
	}
	return nil
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (*User, error) {
	now := time.Now()
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT u.id, u.email, u.password_hash, u.role, u.name, u.company
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1 AND s.expires_at > $2 AND s.last_seen_at > $3`,
		tokenHash, now.Unix(), s.idleCutoff(now)))
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at <= $1 OR last_seen_at <= $2`,
		now.Unix(), s.idleCutoff(now))
	return err
}
