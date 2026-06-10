package instanceconfig

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/yannkr/openrsvp/internal/database"
)

// Store handles database access for the instance_config key/value table.
type Store struct {
	db database.DB
}

// NewStore creates a new instance config Store.
func NewStore(db database.DB) *Store {
	return &Store{db: db}
}

// GetAll returns every key/value pair in the instance_config table.
func (s *Store) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT key, value FROM instance_config")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, rows.Err()
}

// Get returns the value for a single key. The bool is false if the key is absent.
func (s *Store) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM instance_config WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// Set upserts a key/value pair, refreshing updated_at.
func (s *Store) Set(ctx context.Context, key, value string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO instance_config (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, now,
	)
	return err
}

// IsConfigured returns true if the "configured" flag has been set to "true".
func (s *Store) IsConfigured(ctx context.Context) (bool, error) {
	value, ok, err := s.Get(ctx, KeyConfigured)
	if err != nil {
		return false, err
	}
	return ok && value == "true", nil
}
