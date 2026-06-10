-- PostgreSQL supports altering CHECK constraints in place, so we redefine the
-- inline column constraint (auto-named attendees_rsvp_status_check) to add
-- 'waitlisted' rather than rebuilding the table (which would fail due to
-- incoming foreign keys).
ALTER TABLE attendees DROP CONSTRAINT attendees_rsvp_status_check;
ALTER TABLE attendees ADD CONSTRAINT attendees_rsvp_status_check
    CHECK (rsvp_status IN ('pending','attending','maybe','declined','waitlisted'));
