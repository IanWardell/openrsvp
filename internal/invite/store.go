package invite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yannkr/openrsvp/internal/database"
)

// Store handles database operations for invite cards.
type Store struct {
	db database.DB
}

// NewStore creates a new invite Store.
func NewStore(db database.DB) *Store {
	return &Store{db: db}
}

// FindByEventID retrieves the invite card for a given event.
func (s *Store) FindByEventID(ctx context.Context, eventID string) (*InviteCard, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, event_id, template_id, heading, body, footer, primary_color, secondary_color, font, custom_data, created_at, updated_at
		 FROM invite_cards WHERE event_id = ?`, eventID,
	)
	return scanInviteCard(row)
}

// Upsert inserts a new invite card or updates the existing one for the event.
func (s *Store) Upsert(ctx context.Context, card *InviteCard) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Try to find an existing card for this event.
	existing, err := s.FindByEventID(ctx, card.EventID)
	if err != nil {
		return fmt.Errorf("upsert invite card lookup: %w", err)
	}

	if existing != nil {
		// Update the existing card.
		card.ID = existing.ID
		_, err = s.db.ExecContext(ctx,
			`UPDATE invite_cards SET template_id = ?, heading = ?, body = ?, footer = ?, primary_color = ?, secondary_color = ?, font = ?, custom_data = ?, updated_at = ?
			 WHERE id = ?`,
			card.TemplateID, card.Heading, card.Body, card.Footer,
			card.PrimaryColor, card.SecondaryColor, card.Font, card.CustomData,
			now, card.ID,
		)
		if err != nil {
			return fmt.Errorf("update invite card: %w", err)
		}
		card.CreatedAt = existing.CreatedAt
	} else {
		// Insert a new card.
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO invite_cards (id, event_id, template_id, heading, body, footer, primary_color, secondary_color, font, custom_data, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			card.ID, card.EventID, card.TemplateID, card.Heading, card.Body, card.Footer,
			card.PrimaryColor, card.SecondaryColor, card.Font, card.CustomData,
			now, now,
		)
		if err != nil {
			return fmt.Errorf("insert invite card: %w", err)
		}
		card.CreatedAt, _ = time.Parse(time.RFC3339, now)
	}

	card.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

// scanInviteCard scans a single sql.Row into an InviteCard.
func scanInviteCard(row *sql.Row) (*InviteCard, error) {
	var c InviteCard
	var createdAt, updatedAt string

	err := row.Scan(
		&c.ID, &c.EventID, &c.TemplateID, &c.Heading, &c.Body, &c.Footer,
		&c.PrimaryColor, &c.SecondaryColor, &c.Font, &c.CustomData,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan invite card: %w", err)
	}

	c.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	c.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &c, nil
}

// ListCustomTemplates returns the reusable templates owned by an organizer.
func (s *Store) ListCustomTemplates(ctx context.Context, organizerID string) ([]*CustomTemplate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, organizer_id, name, heading, body, footer, primary_color, secondary_color, font, custom_data,
		        created_at, updated_at
		 FROM custom_invite_templates WHERE organizer_id = ? ORDER BY name`, organizerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list custom invite templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	templates := make([]*CustomTemplate, 0)
	for rows.Next() {
		var tmpl CustomTemplate
		var createdAt, updatedAt string
		if err := rows.Scan(
			&tmpl.ID, &tmpl.OrganizerID, &tmpl.Name, &tmpl.Heading, &tmpl.Body, &tmpl.Footer,
			&tmpl.PrimaryColor, &tmpl.SecondaryColor, &tmpl.Font, &tmpl.CustomData,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan custom invite template: %w", err)
		}
		if tmpl.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, fmt.Errorf("parse custom template created_at: %w", err)
		}
		if tmpl.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
			return nil, fmt.Errorf("parse custom template updated_at: %w", err)
		}
		templates = append(templates, &tmpl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom invite templates: %w", err)
	}
	return templates, nil
}

// FindCustomTemplateByID retrieves one organizer-owned reusable template.
func (s *Store) FindCustomTemplateByID(ctx context.Context, organizerID, templateID string) (*CustomTemplate, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, organizer_id, name, heading, body, footer, primary_color, secondary_color, font, custom_data,
		        photo_data, photo_ext, card_background_data, card_background_ext, created_at, updated_at
		 FROM custom_invite_templates WHERE id = ? AND organizer_id = ?`, templateID, organizerID,
	)
	return scanCustomTemplate(row)
}

// FindCustomTemplateByName retrieves a reusable template by organizer and name.
func (s *Store) FindCustomTemplateByName(ctx context.Context, organizerID, name string) (*CustomTemplate, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, organizer_id, name, heading, body, footer, primary_color, secondary_color, font, custom_data,
		        photo_data, photo_ext, card_background_data, card_background_ext, created_at, updated_at
		 FROM custom_invite_templates WHERE organizer_id = ? AND name = ?`, organizerID, name,
	)
	return scanCustomTemplate(row)
}

// UpsertCustomTemplate creates or replaces an organizer template with the same name.
func (s *Store) UpsertCustomTemplate(ctx context.Context, tmpl *CustomTemplate) error {
	now := time.Now().UTC().Format(time.RFC3339)
	existing, err := s.FindCustomTemplateByName(ctx, tmpl.OrganizerID, tmpl.Name)
	if err != nil {
		return fmt.Errorf("custom template lookup: %w", err)
	}
	if existing != nil {
		tmpl.ID = existing.ID
		_, err = s.db.ExecContext(ctx,
			`UPDATE custom_invite_templates
			 SET heading = ?, body = ?, footer = ?, primary_color = ?, secondary_color = ?, font = ?, custom_data = ?,
			     photo_data = ?, photo_ext = ?, card_background_data = ?, card_background_ext = ?, updated_at = ?
			 WHERE id = ? AND organizer_id = ?`,
			tmpl.Heading, tmpl.Body, tmpl.Footer, tmpl.PrimaryColor, tmpl.SecondaryColor, tmpl.Font, tmpl.CustomData,
			tmpl.PhotoData, tmpl.PhotoExt, tmpl.CardBackgroundData, tmpl.CardBackgroundExt, now,
			tmpl.ID, tmpl.OrganizerID,
		)
		if err != nil {
			return fmt.Errorf("update custom invite template: %w", err)
		}
		tmpl.CreatedAt = existing.CreatedAt
	} else {
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO custom_invite_templates
			 (id, organizer_id, name, heading, body, footer, primary_color, secondary_color, font, custom_data,
			  photo_data, photo_ext, card_background_data, card_background_ext, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			tmpl.ID, tmpl.OrganizerID, tmpl.Name, tmpl.Heading, tmpl.Body, tmpl.Footer,
			tmpl.PrimaryColor, tmpl.SecondaryColor, tmpl.Font, tmpl.CustomData,
			tmpl.PhotoData, tmpl.PhotoExt, tmpl.CardBackgroundData, tmpl.CardBackgroundExt, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert custom invite template: %w", err)
		}
		tmpl.CreatedAt, _ = time.Parse(time.RFC3339, now)
	}
	tmpl.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

// DeleteCustomTemplate removes an organizer-owned reusable template.
func (s *Store) DeleteCustomTemplate(ctx context.Context, organizerID, templateID string) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM custom_invite_templates WHERE id = ? AND organizer_id = ?`, templateID, organizerID,
	)
	if err != nil {
		return false, fmt.Errorf("delete custom invite template: %w", err)
	}
	count, err := result.RowsAffected()
	return count > 0, err
}

func scanCustomTemplate(row *sql.Row) (*CustomTemplate, error) {
	var tmpl CustomTemplate
	var createdAt, updatedAt string
	err := row.Scan(
		&tmpl.ID, &tmpl.OrganizerID, &tmpl.Name, &tmpl.Heading, &tmpl.Body, &tmpl.Footer,
		&tmpl.PrimaryColor, &tmpl.SecondaryColor, &tmpl.Font, &tmpl.CustomData,
		&tmpl.PhotoData, &tmpl.PhotoExt, &tmpl.CardBackgroundData, &tmpl.CardBackgroundExt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan custom invite template: %w", err)
	}
	if tmpl.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("parse custom template created_at: %w", err)
	}
	if tmpl.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return nil, fmt.Errorf("parse custom template updated_at: %w", err)
	}
	return &tmpl, nil
}
