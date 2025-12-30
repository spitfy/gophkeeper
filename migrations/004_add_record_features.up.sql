ALTER TABLE records
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS checksum VARCHAR(64),
    ADD COLUMN IF NOT EXISTS device_id VARCHAR(255);

ALTER TABLE records DROP CONSTRAINT IF EXISTS records_user_id_type_encrypted_data_key;

CREATE UNIQUE INDEX IF NOT EXISTS records_user_type_data_unique
    ON records(user_id, type, encrypted_data)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_records_checksum ON records(checksum);
CREATE INDEX IF NOT EXISTS idx_records_deleted_at ON records(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_records_user_device ON records(user_id, device_id) WHERE device_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_records_user_type_modified ON records(user_id, type, last_modified DESC);

