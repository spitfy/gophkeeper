CREATE TABLE IF NOT EXISTS record_versions (
    id SERIAL PRIMARY KEY,
    record_id INTEGER NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    encrypted_data BYTEA NOT NULL,
    meta JSONB NOT NULL DEFAULT '{}',
    checksum VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (record_id, version)
);

CREATE INDEX IF NOT EXISTS idx_record_versions_record_id ON record_versions(record_id);
CREATE INDEX IF NOT EXISTS idx_record_versions_created_at ON record_versions(created_at);