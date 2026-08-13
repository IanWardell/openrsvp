ALTER TABLE organizers ADD COLUMN role TEXT NOT NULL DEFAULT 'organizer' CHECK(role IN ('organizer','admin','super_admin'));
UPDATE organizers SET role = 'super_admin' WHERE is_admin = TRUE;
ALTER TABLE organizers ADD COLUMN invited_at TEXT;
ALTER TABLE organizers ADD COLUMN last_login_at TEXT;
ALTER TABLE organizers ADD COLUMN suspended_at TEXT;
ALTER TABLE organizers ADD COLUMN suspended_by TEXT;
ALTER TABLE organizers ADD COLUMN suspension_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE organizers DROP COLUMN is_admin;

CREATE INDEX idx_organizers_role ON organizers(role);
CREATE INDEX idx_organizers_suspended_at ON organizers(suspended_at);

ALTER TABLE events ADD COLUMN suspended_at TEXT;
ALTER TABLE events ADD COLUMN suspended_by TEXT;
ALTER TABLE events ADD COLUMN suspension_reason TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_events_suspended_at ON events(suspended_at);

CREATE TABLE admin_audit_log (
    id            TEXT PRIMARY KEY,
    actor_id      TEXT NOT NULL,
    actor_role    TEXT NOT NULL,
    action        TEXT NOT NULL,
    target_type   TEXT NOT NULL,
    target_id     TEXT NOT NULL,
    reason        TEXT NOT NULL DEFAULT '',
    metadata      TEXT NOT NULL DEFAULT '{}',
    request_id    TEXT NOT NULL DEFAULT '',
    remote_ip     TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL
);

CREATE INDEX idx_admin_audit_created_at ON admin_audit_log(created_at);
CREATE INDEX idx_admin_audit_actor_id ON admin_audit_log(actor_id);
CREATE INDEX idx_admin_audit_target ON admin_audit_log(target_type, target_id);
CREATE INDEX idx_admin_audit_action ON admin_audit_log(action);
