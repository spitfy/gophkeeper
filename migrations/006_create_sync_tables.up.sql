-- Таблица устройств
CREATE TABLE IF NOT EXISTS devices (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'desktop',
    last_sync_time TIMESTAMP WITH TIME ZONE,
    ip_address VARCHAR(45), -- IPv6 support
    user_agent VARCHAR(1000),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Таблица конфликтов синхронизации
CREATE TABLE IF NOT EXISTS sync_conflicts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    record_id INTEGER NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    local_data BYTEA,
    server_data BYTEA,
    conflict_type VARCHAR(50) NOT NULL,
    resolved BOOLEAN DEFAULT FALSE,
    resolution VARCHAR(1000),
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_last_sync ON devices(last_sync_time DESC);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_resolved ON sync_conflicts(resolved);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_record_id ON sync_conflicts(record_id);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггеры
CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sync_conflicts_updated_at
    BEFORE UPDATE ON sync_conflicts
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
