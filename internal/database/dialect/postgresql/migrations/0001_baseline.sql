-- Baseline schema for the PostgreSQL dialect.
--
-- Published migrations are immutable: internal/database/migrate compares the
-- SHA-256 of every applied file and refuses to start when one changes, so append
-- a new file instead of editing this one.
--
-- Storage contract shared with every dialect: timestamps are Unix seconds in
-- UTC, booleans are native BOOLEAN values.

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT,
    avatar_path TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('owner', 'user')),
    security_version BIGINT NOT NULL DEFAULT 1,
    disabled_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE UNIQUE INDEX users_username_unique_idx ON users (LOWER(username));

CREATE UNIQUE INDEX users_email_unique_idx ON users (LOWER(email))
WHERE email IS NOT NULL AND email <> '';

-- At most one owner may exist.
CREATE UNIQUE INDEX users_owner_unique_idx ON users (role)
WHERE role = 'owner';

CREATE INDEX users_status_idx ON users (status);

CREATE TABLE credentials (
    user_id BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    password_changed_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    last_used_at BIGINT NOT NULL,
    last_rotated_at BIGINT,
    expires_at BIGINT NOT NULL,
    idle_expires_at BIGINT NOT NULL,
    revoked_at BIGINT,
    revoke_reason TEXT,
    ip TEXT,
    user_agent TEXT
);

CREATE INDEX auth_sessions_user_idx ON auth_sessions (user_id);

CREATE INDEX auth_sessions_active_user_idx ON auth_sessions (user_id)
WHERE revoked_at IS NULL;

-- Singleton row holding installation state.
CREATE TABLE install_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    setup_completed BOOLEAN NOT NULL,
    owner_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    setup_completed_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

-- Singleton row holding account policies.
CREATE TABLE account_policies (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    public_registration_enabled BOOLEAN NOT NULL,
    self_service_account_deletion_enabled BOOLEAN NOT NULL
);

INSERT INTO install_state (id, setup_completed, created_at, updated_at)
VALUES (1, FALSE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT);

INSERT INTO account_policies (id, public_registration_enabled, self_service_account_deletion_enabled)
VALUES (1, FALSE, FALSE);
