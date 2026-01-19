-- Откатываем изменения в обратном порядке

-- Удаляем VIEW
DROP VIEW IF EXISTS sync_status_view;

-- Удаляем триггеры
DROP TRIGGER IF EXISTS update_sync_conflicts_updated_at ON sync_conflicts;
DROP TRIGGER IF EXISTS update_devices_updated_at ON devices;

-- Удаляем таблицы
DROP TABLE IF EXISTS sync_conflicts CASCADE;
DROP TABLE IF EXISTS devices CASCADE;

-- Восстанавливаем старые таблицы из миграции 006
CREATE TABLE IF NOT EXISTS sync_status (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_sync_time TIMESTAMP WITH TIME ZONE,
    total_records INTEGER DEFAULT 0,
    device_count INTEGER DEFAULT 0,
    storage_used BIGINT DEFAULT 0,
    storage_limit BIGINT DEFAULT 104857600,
    sync_version BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'desktop',
    last_sync_time TIMESTAMP WITH TIME ZONE,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sync_stats (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_syncs INTEGER DEFAULT 0,
    last_sync TIMESTAMP WITH TIME ZONE,
    total_uploads BIGINT DEFAULT 0,
    total_downloads BIGINT DEFAULT 0,
    total_conflicts INTEGER DEFAULT 0,
    total_resolved INTEGER DEFAULT 0,
    avg_sync_duration DOUBLE PRECISION DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sync_conflicts (
    id SERIAL PRIMARY KEY,
    record_id INTEGER NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    local_data BYTEA,
    server_data BYTEA,
    conflict_type TEXT NOT NULL,
    resolved BOOLEAN DEFAULT FALSE,
    resolution TEXT,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_last_sync ON devices(last_sync_time DESC);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_resolved ON sync_conflicts(resolved);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_record_id ON sync_conflicts(record_id);

CREATE TRIGGER update_sync_status_updated_at BEFORE UPDATE ON sync_status
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_devices_updated_at BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sync_stats_updated_at BEFORE UPDATE ON sync_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sync_conflicts_updated_at BEFORE UPDATE ON sync_conflicts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
