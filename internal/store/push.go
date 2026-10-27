package store

import (
	"context"
	"time"
)

// maxPushSubscriptions caps devices per account; the oldest are dropped first.
const maxPushSubscriptions = 10

type PushSubscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// PendingPush is a notification the dispatcher has claimed for delivery.
type PendingPush struct {
	ID        int64
	UserID    int64
	Role      string
	Title     string
	Message   string
	Payload   string
	CreatedAt int64
}

// SavePushSubscription stores a device for userID. A browser that was subscribed
// under another account is moved to this one.
func (s *Store) SavePushSubscription(ctx context.Context, userID int64, sub PushSubscription) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO push_subscriptions(user_id,endpoint,p256dh,auth,created_at) VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (endpoint) DO UPDATE SET user_id=EXCLUDED.user_id, p256dh=EXCLUDED.p256dh, auth=EXCLUDED.auth, created_at=EXCLUDED.created_at`,
		userID, sub.Endpoint, sub.P256dh, sub.Auth, time.Now().Unix()); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE user_id=$1 AND id NOT IN (
		SELECT id FROM push_subscriptions WHERE user_id=$1 ORDER BY created_at DESC, id DESC LIMIT $2)`,
		userID, maxPushSubscriptions); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeletePushSubscription(ctx context.Context, userID int64, endpoint string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE user_id=$1 AND endpoint=$2`, userID, endpoint)
	return err
}

// DeletePushEndpoint removes a subscription the push service reported as gone.
func (s *Store) DeletePushEndpoint(ctx context.Context, endpoint string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE endpoint=$1`, endpoint)
	return err
}

func (s *Store) PushSubscriptions(ctx context.Context, userID int64) ([]PushSubscription, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT endpoint,p256dh,auth FROM push_subscriptions WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PushSubscription
	for rows.Next() {
		var sub PushSubscription
		if err := rows.Scan(&sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// ClaimPushNotifications marks up to limit undelivered notifications as pushed and
// returns them. Claiming first means a notification is pushed at most once.
func (s *Store) ClaimPushNotifications(ctx context.Context, limit int) ([]PendingPush, error) {
	rows, err := s.db.QueryContext(ctx, `UPDATE notifications n SET pushed_at=$1
		FROM users u
		WHERE u.id=n.user_id AND n.id IN (
			SELECT id FROM notifications WHERE pushed_at IS NULL ORDER BY id LIMIT $2 FOR UPDATE SKIP LOCKED)
		RETURNING n.id, n.user_id, u.role, n.title, n.message, n.payload, n.created_at`,
		time.Now().Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingPush
	for rows.Next() {
		var p PendingPush
		if err := rows.Scan(&p.ID, &p.UserID, &p.Role, &p.Title, &p.Message, &p.Payload, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
