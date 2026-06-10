-- PostgreSQL supports altering CHECK constraints in place, so we redefine the
-- inline column constraint (auto-named events_status_check) rather than
-- rebuilding the table (which would fail due to incoming foreign keys).
ALTER TABLE events DROP CONSTRAINT events_status_check;
ALTER TABLE events ADD CONSTRAINT events_status_check
    CHECK (status IN ('draft','published','cancelled','archived','retention_warning'));
