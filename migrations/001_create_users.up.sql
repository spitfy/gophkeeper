CREATE TABLE users
(
    id            SERIAL PRIMARY KEY,
    login         TEXT                     NOT NULL UNIQUE,
    password_hash TEXT                     NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_login ON users (login);
