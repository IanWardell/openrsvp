-- PostgreSQL supports altering CHECK constraints in place, so we redefine the
-- inline column constraint (auto-named reminders_status_check) to add
-- 'processing'. The target_group constraint is left untouched.
ALTER TABLE reminders DROP CONSTRAINT reminders_status_check;
ALTER TABLE reminders ADD CONSTRAINT reminders_status_check
    CHECK (status IN ('scheduled','processing','sent','cancelled','failed'));
