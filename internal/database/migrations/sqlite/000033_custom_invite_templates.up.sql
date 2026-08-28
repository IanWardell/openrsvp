CREATE TABLE custom_invite_templates (
    id                    TEXT PRIMARY KEY,
    organizer_id          TEXT NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    heading               TEXT NOT NULL DEFAULT '',
    body                  TEXT NOT NULL DEFAULT '',
    footer                TEXT NOT NULL DEFAULT '',
    primary_color         TEXT NOT NULL DEFAULT '#6366f1',
    secondary_color       TEXT NOT NULL DEFAULT '#f0abfc',
    font                  TEXT NOT NULL DEFAULT 'Inter',
    custom_data           TEXT NOT NULL DEFAULT '{}',
    photo_data            BLOB,
    photo_ext             TEXT NOT NULL DEFAULT '',
    card_background_data  BLOB,
    card_background_ext   TEXT NOT NULL DEFAULT '',
    created_at            TEXT NOT NULL,
    updated_at            TEXT NOT NULL,
    UNIQUE(organizer_id, name)
);

CREATE INDEX idx_custom_invite_templates_organizer_id
    ON custom_invite_templates(organizer_id);
