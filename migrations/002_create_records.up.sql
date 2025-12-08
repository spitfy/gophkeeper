CREATE TABLE records
(
    id             SERIAL PRIMARY KEY,
    user_id        INTEGER                  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type           TEXT                     NOT NULL CHECK (type IN ('login', 'text', 'binary', 'card')),
    encrypted_data BYTEA                    NOT NULL,
    meta           JSONB                    NOT NULL DEFAULT '{}',
    version        INTEGER                  NOT NULL DEFAULT 1,
    last_modified  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, type, encrypted_data) -- предотвращает дубликаты
);

CREATE INDEX idx_records_user_id ON records (user_id);
CREATE INDEX idx_records_user_type ON records (user_id, type);
CREATE INDEX idx_records_last_modified ON records (last_modified);