-- Web Push subscriptions: one row per browser/device that allowed notifications.
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user ON push_subscriptions(user_id);

-- pushed_at marks notifications the push dispatcher has handled. Existing rows are
-- marked so turning push on does not replay old notifications.
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS pushed_at BIGINT;
UPDATE notifications SET pushed_at = created_at WHERE pushed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_unpushed ON notifications(id) WHERE pushed_at IS NULL;
