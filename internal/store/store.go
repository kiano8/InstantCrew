package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email not registered")
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string
	Name         string
	Company      string
}

type Store struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	email         TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	role          TEXT NOT NULL CHECK (role IN ('employer','applicant')),
	name          TEXT NOT NULL DEFAULT '',
	company       TEXT NOT NULL DEFAULT '',
	created_at    INTEGER NOT NULL,
	UNIQUE (email, role)
);
CREATE TABLE IF NOT EXISTS sessions (
	token_hash TEXT PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at INTEGER NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
`

func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // simplest way to avoid SQLITE_BUSY with one writer
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) CreateUser(ctx context.Context, u *User) error {
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
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		tokenHash, userID, expires.Unix(), time.Now().Unix())
	return err
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (*User, error) {
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT u.id, u.email, u.password_hash, u.role, u.name, u.company
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > ?`, tokenHash, time.Now().Unix()))
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, time.Now().Unix())
	return err
}
