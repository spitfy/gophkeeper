-- Таблица для отслеживания синхронизации
CREATE TABLE IF NOT EXISTS sync_metadata (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    last_sync_time TIMESTAMP WITH TIME ZONE,
    sync_version BIGINT DEFAULT 0,
    device_name TEXT,
    client_version TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Таблица для хранения конфликтов
CREATE TABLE IF NOT EXISTS sync_conflicts (
    id TEXT PRIMARY KEY,
    record_id TEXT NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    local_data JSONB,
    server_data JSONB,
    conflict_type TEXT NOT NULL,
    resolved BOOLEAN DEFAULT FALSE,
    resolution TEXT,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_sync_metadata_client_id ON sync_metadata(client_id);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_record_id ON sync_conflicts(record_id);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_resolved ON sync_conflicts(resolved);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_sync_metadata_updated_at BEFORE UPDATE ON sync_metadata
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sync_conflicts_updated_at BEFORE UPDATE ON sync_conflicts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();