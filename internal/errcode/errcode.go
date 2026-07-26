package errcode

import (
	"crypto/rand"
	"errors"
	"fmt"
)

// ErrValidation marks an error as a caller input problem rather than a server
// fault. Handlers map it to HTTP 400 and return its message verbatim.
//
// Match on this sentinel with IsValidation, never on the message text. Handlers
// used to classify errors with hardcoded allowlists of message prefixes, which
// fail open to HTTP 500 for every message not on the list: real guests were
// blocked from RSVPing by an unactionable "an internal error occurred (ref:
// ...)", and an "eventDate is required" / "event_date is required" mismatch did
// the same to event creation.
var ErrValidation = errors.New("validation error")

// validationError carries a client-safe message while matching ErrValidation
// under errors.Is.
type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }
func (e *validationError) Unwrap() error { return ErrValidation }

// Validationf builds a client-safe validation error.
func Validationf(format string, args ...any) error {
	return &validationError{msg: fmt.Sprintf(format, args...)}
}

// IsValidation reports whether err is a client input problem whose message is
// safe to return verbatim.
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}

// Ref generates a short, unique error reference code suitable for
// correlating client-facing error messages with server-side log entries.
// The format is "ERR-" followed by 8 uppercase hex characters
// (e.g., "ERR-A1B2C3D4"), giving ~4 billion unique codes.
func Ref() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("ERR-%X", b)
}
