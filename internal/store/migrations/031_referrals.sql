-- Referral promo: a crew member who signs up with another crew member's code
-- earns that referrer 1 hire. New crew earn ShiftRewardHires after their first
-- ShiftRewardAfter completed shifts.
ALTER TABLE users ADD COLUMN referred_by BIGINT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN shift_reward_granted BOOLEAN NOT NULL DEFAULT false;
-- The shift reward is for new sign-ups only, so existing accounts never get it.
UPDATE users SET shift_reward_granted = true;
