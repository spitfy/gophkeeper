-- Удаляем избыточные таблицы
DROP TABLE IF EXISTS sync_status CASCADE;
DROP TABLE IF EXISTS sync_stats CASCADE;

-- Пересоздаем таблицу devices с правильной структурой
DROP TABLE IF EXISTS sync_conflicts CASCADE;
DROP TABLE IF EXISTS devices CASCADE;

CREATE TABLE devices (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'desktop',
    last_sync_time TIMESTAMP WITH TIME ZONE,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_devices_user_id ON devices(user_id);
CREATE INDEX idx_devices_last_sync ON devices(last_sync_time DESC);

-- Пересоздаем таблицу sync_conflicts с правильными ссылками
CREATE TABLE sync_conflicts (
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

CREATE INDEX idx_sync_conflicts_resolved ON sync_conflicts(resolved);
CREATE INDEX idx_sync_conflicts_record_id ON sync_conflicts(record_id);
CREATE INDEX idx_sync_conflicts_user_device ON sync_conflicts(user_id, device_id);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_devices_updated_at BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sync_conflicts_updated_at BEFORE UPDATE ON sync_conflicts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Создаем VIEW для замены sync_status (вычисляемые данные)
CREATE OR REPLACE VIEW sync_status_view AS
SELECT 
    u.id as user_id,
    MAX(r.last_modified) as last_sync_time,
    COUNT(r.id) FILTER (WHERE r.deleted_at IS NULL) as total_records,
    COUNT(DISTINCT r.device_id) FILTER (WHERE r.deleted_at IS NULL AND r.device_id IS NOT NULL) as device_count,
    COALESCE(SUM(LENGTH(r.encrypted_data)) FILTER (WHERE r.deleted_at IS NULL), 0) as storage_used,
    104857600 as storage_limit,
    COALESCE(MAX(r.version), 0) as sync_version
FROM users u
LEFT JOIN records r ON u.id = r.user_id
GROUP BY u.id;
