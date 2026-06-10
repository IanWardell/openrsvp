package suppression

import "time"

// Suppression reasons.
const (
	ReasonUnsubscribe = "unsubscribe"
	ReasonBounce      = "bounce"
	ReasonComplaint   = "complaint"
)

// ValidReasons enumerates the accepted suppression reasons.
var ValidReasons = map[string]bool{
	ReasonUnsubscribe: true,
	ReasonBounce:      true,
	ReasonComplaint:   true,
}

// Suppression represents a single email-suppression entry. A nil EventID means
// the email is suppressed globally for this instance; otherwise the suppression
// is scoped to a single event.
type Suppression struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	EventID   *string   `json:"eventId"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}
