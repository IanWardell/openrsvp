CREATE TABLE email_suppressions (
    id         TEXT PRIMARY KEY,
    email      TEXT NOT NULL,
    event_id   TEXT REFERENCES events(id) ON DELETE CASCADE,
    reason     TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_email_suppressions_email_event ON email_suppressions(email, event_id);
CREATE INDEX idx_email_suppressions_email ON email_suppressions(email);

CREATE TABLE unsubscribe_tokens (
    id         TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    email      TEXT NOT NULL,
    event_id   TEXT REFERENCES events(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_unsubscribe_tokens_email ON unsubscribe_tokens(email);
