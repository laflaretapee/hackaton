CREATE TABLE IF NOT EXISTS teams (id text PRIMARY KEY, name text NOT NULL, direction text NOT NULL);
CREATE TABLE IF NOT EXISTS users (
 id text PRIMARY KEY, name text NOT NULL, role text NOT NULL CHECK(role IN ('intern','mentor','manager')),
 team_id text NOT NULL REFERENCES teams(id), mentor_id text REFERENCES users(id) ON DELETE SET NULL,
 telegram_id bigint UNIQUE, is_demo boolean NOT NULL DEFAULT false, notifications boolean NOT NULL DEFAULT true,
 reminder_minutes integer NOT NULL DEFAULT 60 CHECK (reminder_minutes BETWEEN 1 AND 10080), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS users_mentor ON users(mentor_id);
CREATE INDEX IF NOT EXISTS users_team ON users(team_id);
CREATE TABLE IF NOT EXISTS sessions (token_hash text PRIMARY KEY, user_id text REFERENCES users(id) ON DELETE CASCADE, admin boolean NOT NULL DEFAULT false, expires_at timestamptz NOT NULL);
CREATE INDEX IF NOT EXISTS sessions_expiry ON sessions(expires_at);
CREATE TABLE IF NOT EXISTS login_links (token_hash text PRIMARY KEY, user_id text REFERENCES users(id) ON DELETE CASCADE, expires_at timestamptz NOT NULL, consumed boolean NOT NULL DEFAULT false);
CREATE TABLE IF NOT EXISTS catalog (id text PRIMARY KEY, kind text NOT NULL CHECK(kind IN ('article','lesson')), body jsonb NOT NULL);
CREATE INDEX IF NOT EXISTS catalog_search ON catalog USING gin(to_tsvector('russian',body::text));
CREATE TABLE IF NOT EXISTS records (id text PRIMARY KEY, kind text NOT NULL CHECK(kind IN ('task','read','attempt','message','request','meeting')), user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE, body jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX IF NOT EXISTS records_owner_kind ON records(user_id,kind);
CREATE TABLE IF NOT EXISTS settings (id integer PRIMARY KEY CHECK(id=1), body jsonb NOT NULL, telegram_secret text NOT NULL DEFAULT '', routerai_secret text NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS audit (id bigserial PRIMARY KEY, action text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS bot_states (telegram_id bigint PRIMARY KEY, body jsonb NOT NULL, updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS runtime (key text PRIMARY KEY, value text NOT NULL);
