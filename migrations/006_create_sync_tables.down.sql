DROP TRIGGER IF EXISTS update_sync_conflicts_updated_at ON sync_conflicts;
DROP TRIGGER IF EXISTS update_sync_stats_updated_at ON sync_stats;
DROP TRIGGER IF EXISTS update_devices_updated_at ON devices;
DROP TRIGGER IF EXISTS update_sync_status_updated_at ON sync_status;

DROP TABLE IF EXISTS sync_conflicts;
DROP TABLE IF EXISTS sync_stats;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS sync_status;