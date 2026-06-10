-- Revert the attendees.rsvp_status CHECK constraint to remove 'waitlisted'. Move
-- any rows out of that status first, then narrow the constraint.
UPDATE attendees SET rsvp_status = 'pending' WHERE rsvp_status = 'waitlisted';
ALTER TABLE attendees DROP CONSTRAINT attendees_rsvp_status_check;
ALTER TABLE attendees ADD CONSTRAINT attendees_rsvp_status_check
    CHECK (rsvp_status IN ('pending','attending','maybe','declined'));
