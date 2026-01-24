CREATE TABLE users
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    login         VARCHAR(100)             NOT NULL UNIQUE,
    password_hash VARCHAR(255)             NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_login ON users (login);