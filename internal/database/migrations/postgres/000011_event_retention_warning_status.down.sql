-- Revert to the original CHECK constraint without retention_warning. Move any
-- rows out of the soon-to-be-invalid status first, then narrow the constraint.
UPDATE events SET status = 'published' WHERE status = 'retention_warning';
ALTER TABLE events DROP CONSTRAINT events_status_check;
ALTER TABLE events ADD CONSTRAINT events_status_check
    CHECK (status IN ('draft','published','cancelled','archived'));
