-- Revert the reminders.status CHECK constraint to remove 'processing'. Move any
-- rows out of that status first, then narrow the constraint.
UPDATE reminders SET status = 'scheduled' WHERE status = 'processing';
ALTER TABLE reminders DROP CONSTRAINT reminders_status_check;
ALTER TABLE reminders ADD CONSTRAINT reminders_status_check
    CHECK (status IN ('scheduled','sent','cancelled','failed'));
