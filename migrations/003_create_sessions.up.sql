CREATE TABLE IF NOT EXISTS sessions
(
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER                  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash BYTEA                    NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions (expires_at); -- простой индекс без WHERE
CREATE INDEX IF NOT EXISTS idx_sessions_user_expires ON sessions (user_id, expires_at);