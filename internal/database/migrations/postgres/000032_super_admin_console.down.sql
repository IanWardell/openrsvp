DROP TABLE IF EXISTS admin_audit_log;
DROP INDEX IF EXISTS idx_events_suspended_at;
ALTER TABLE events DROP COLUMN suspension_reason;
ALTER TABLE events DROP COLUMN suspended_by;
ALTER TABLE events DROP COLUMN suspended_at;

DROP INDEX IF EXISTS idx_organizers_suspended_at;
DROP INDEX IF EXISTS idx_organizers_role;
ALTER TABLE organizers ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE organizers SET is_admin = TRUE WHERE role IN ('admin','super_admin');
ALTER TABLE organizers DROP COLUMN suspension_reason;
ALTER TABLE organizers DROP COLUMN suspended_by;
ALTER TABLE organizers DROP COLUMN suspended_at;
ALTER TABLE organizers DROP COLUMN last_login_at;
ALTER TABLE organizers DROP COLUMN invited_at;
ALTER TABLE organizers DROP COLUMN role;
