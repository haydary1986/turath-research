-- ══════════════════════════════════════════════════
-- Migration 20: login audit and account activity (admin only).
-- ══════════════════════════════════════════════════

-- Last authenticated request, written by the auth middleware at most once a minute per user.
ALTER TABLE users ADD COLUMN last_seen_at TIMESTAMPTZ;
CREATE INDEX idx_users_last_seen ON users (last_seen_at DESC NULLS LAST);

-- Every sign-in attempt, successful or not. Never holds passwords or tokens.
-- Rows older than 180 days are deleted daily by the API.
CREATE TABLE login_events (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    email      VARCHAR(255) NOT NULL DEFAULT '',
    method     VARCHAR(20) NOT NULL CHECK (method IN ('password', 'google')),
    success    BOOLEAN NOT NULL,
    reason     VARCHAR(40),
    ip         VARCHAR(64) NOT NULL DEFAULT '',
    user_agent VARCHAR(300) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (success = (reason IS NULL))
);

CREATE INDEX idx_login_events_created ON login_events (created_at DESC);
CREATE INDEX idx_login_events_user    ON login_events (user_id, created_at DESC);
CREATE INDEX idx_login_events_email   ON login_events (lower(email), created_at DESC);
