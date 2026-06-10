package suppression

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Service orchestrates email suppression and unsubscribe-token handling.
type Service struct {
	store *Store
}

// NewService creates a new suppression Service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// normalizeEmail lowercases and trims an email so suppression checks are
// case-insensitive and consistent with how addresses are stored.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// normalizeEventID converts an event id into the nullable form used by the
// store: an empty string means a global (instance-wide) scope.
func normalizeEventID(eventID string) *string {
	if strings.TrimSpace(eventID) == "" {
		return nil
	}
	v := eventID
	return &v
}

// IsSuppressed reports whether the email should be excluded from sends for the
// given event. An empty eventID checks only global suppression. A global
// suppression always wins over an event scope.
func (s *Service) IsSuppressed(ctx context.Context, email, eventID string) bool {
	suppressed, err := s.store.IsSuppressed(ctx, normalizeEmail(email), normalizeEventID(eventID))
	if err != nil {
		// Fail closed would block all mail on a transient DB error; fail open
		// so a lookup failure does not silently drop every message. Callers
		// log at the call site if needed.
		return false
	}
	return suppressed
}

// Suppress records that the email opted out. An empty eventID creates a global
// suppression for this instance; otherwise the suppression is scoped to the
// event. The operation is idempotent.
func (s *Service) Suppress(ctx context.Context, email, eventID, reason string) error {
	if !ValidReasons[reason] {
		return fmt.Errorf("invalid suppression reason: %q", reason)
	}
	return s.store.Suppress(ctx, normalizeEmail(email), normalizeEventID(eventID), reason)
}

// hashToken returns the hex-encoded SHA-256 of a raw token. Only the hash is
// persisted, so a database leak does not expose working unsubscribe links.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateUnsubscribeToken creates a one-way unsubscribe token for (email,
// eventID), persists its hash, and returns the raw token for embedding in an
// unsubscribe link. An empty eventID produces a global-scope token.
func (s *Service) GenerateUnsubscribeToken(ctx context.Context, email, eventID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate unsubscribe token: %w", err)
	}
	token := hex.EncodeToString(raw)

	if err := s.store.CreateToken(ctx, hashToken(token), normalizeEmail(email), normalizeEventID(eventID)); err != nil {
		return "", err
	}
	return token, nil
}

// VerifyUnsubscribeToken resolves a raw token to its email and event scope.
// ok is false when the token is unknown. A nil eventID means a global token.
func (s *Service) VerifyUnsubscribeToken(ctx context.Context, token string) (email string, eventID *string, ok bool, err error) {
	if strings.TrimSpace(token) == "" {
		return "", nil, false, nil
	}
	return s.store.FindTokenByHash(ctx, hashToken(token))
}
