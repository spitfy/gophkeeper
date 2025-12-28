CREATE TABLE IF NOT EXISTS sync_metadata (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL,
    last_sync_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sync_token VARCHAR(255),
    UNIQUE (user_id, device_id)
);

CREATE TABLE IF NOT EXISTS sync_conflicts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    record_id INTEGER REFERENCES records(id) ON DELETE SET NULL,
    device_id VARCHAR(255) NOT NULL,
    conflict_data JSONB NOT NULL,
    resolved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_metadata_user_device ON sync_metadata(user_id, device_id);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_user_resolved ON sync_conflicts(user_id, resolved);