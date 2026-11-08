-- 站内通知中心：面向单个用户的站内通知事件表。
--
-- 承载由系统事件产生的、只发给某个具体用户的通知（例如余额低额提醒、账户限额告警、
-- 支付/充值到账等）。与"面向群体广播"的公告(announcements)区分：
--   - announcements: 管理员面向全用户/分组/余额条件的广播，共享一张读状态表。
--   - user_notifications: 由后端事件按 user_id 落库的个性化通知，自带 read_at。
--
-- type 取值约定（可扩展，无需改表）：
--   balance_low      余额低额提醒
--   account_quota    账号限额告警
--   payment_received 充值/支付到账
--   key_rotated      API Key 轮换
-- 前端按 type 决定图标与跳转目标。

CREATE TABLE IF NOT EXISTS user_notifications (
    id         BIGSERIAL PRIMARY KEY,

    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    type       VARCHAR(32)  NOT NULL DEFAULT 'system',
    title      VARCHAR(200) NOT NULL,
    body       TEXT         NOT NULL DEFAULT '',
    metadata   JSONB        NOT NULL DEFAULT '{}'::jsonb,

    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_notifications_user_created
    ON user_notifications (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_notifications_user_read
    ON user_notifications (user_id, read_at);

COMMENT ON TABLE user_notifications IS
    'Per-user in-app notification events (balance_low, account_quota, payment_received, key_rotated, etc.)';
COMMENT ON COLUMN user_notifications.type IS
    'Notification category; drives frontend icon and navigation';