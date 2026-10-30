-- Baseline schema for the SQLite dialect.
--
-- Published migrations are immutable: internal/database/migrate compares the
-- SHA-256 of every applied file and refuses to start when one changes, so append
-- a new file instead of editing this one.
--
-- Storage contract shared with every dialect: timestamps are Unix seconds in
-- UTC, booleans are 0/1 integers.

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL COLLATE NOCASE,
    email TEXT COLLATE NOCASE,
    avatar_path TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('owner', 'user')),
    security_version INTEGER NOT NULL DEFAULT 1,
    disabled_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX users_username_unique_idx ON users (username);

CREATE UNIQUE INDEX users_email_unique_idx ON users (email)
WHERE email IS NOT NULL AND email <> '';

-- At most one owner may exist.
CREATE UNIQUE INDEX users_owner_unique_idx ON users (role)
WHERE role = 'owner';

CREATE INDEX users_status_idx ON users (status);

CREATE TABLE credentials (
    user_id INTEGER PRIMARY KEY,
    password_hash TEXT NOT NULL,
    password_changed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    refresh_token_hash TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    last_used_at INTEGER NOT NULL,
    last_rotated_at INTEGER,
    expires_at INTEGER NOT NULL,
    idle_expires_at INTEGER NOT NULL,
    revoked_at INTEGER,
    revoke_reason TEXT,
    ip TEXT,
    user_agent TEXT,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX auth_sessions_user_idx ON auth_sessions (user_id);

CREATE INDEX auth_sessions_active_user_idx ON auth_sessions (user_id)
WHERE revoked_at IS NULL;

-- Singleton row holding installation state.
CREATE TABLE install_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    setup_completed INTEGER NOT NULL CHECK (setup_completed IN (0, 1)),
    owner_user_id INTEGER,
    setup_completed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (owner_user_id) REFERENCES users (id) ON DELETE SET NULL
);

-- Singleton row holding account policies.
CREATE TABLE account_policies (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    public_registration_enabled INTEGER NOT NULL CHECK (public_registration_enabled IN (0, 1)),
    self_service_account_deletion_enabled INTEGER NOT NULL CHECK (self_service_account_deletion_enabled IN (0, 1))
);

INSERT INTO install_state (id, setup_completed, created_at, updated_at)
VALUES (1, 0, CAST(strftime('%s', 'now') AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER));

INSERT INTO account_policies (id, public_registration_enabled, self_service_account_deletion_enabled)
VALUES (1, 0, 0);
