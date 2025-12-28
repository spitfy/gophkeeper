DROP INDEX IF EXISTS records_user_type_data_unique;
DROP INDEX IF EXISTS idx_records_checksum;
DROP INDEX IF EXISTS idx_records_deleted_at;
DROP INDEX IF EXISTS idx_records_user_device;
DROP INDEX IF EXISTS idx_records_user_type_modified;

ALTER TABLE records
DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS checksum,
    DROP COLUMN IF EXISTS device_id;

ALTER TABLE records
    ADD CONSTRAINT records_user_id_type_encrypted_data_key
        UNIQUE (user_id, type, encrypted_data);