package invite

import "time"

// InviteCard represents the visual invite card configuration for an event.
type InviteCard struct {
	ID             string    `json:"id"`
	EventID        string    `json:"eventId"`
	TemplateID     string    `json:"templateId"`
	Heading        string    `json:"heading"`
	Body           string    `json:"body"`
	Footer         string    `json:"footer"`
	PrimaryColor   string    `json:"primaryColor"`
	SecondaryColor string    `json:"secondaryColor"`
	Font           string    `json:"font"`
	CustomData     string    `json:"customData"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Template represents a built-in invite card template.
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SaveInviteRequest is the request body for creating or updating an invite card.
type SaveInviteRequest struct {
	TemplateID     string `json:"templateId"`
	Heading        string `json:"heading"`
	Body           string `json:"body"`
	Footer         string `json:"footer"`
	PrimaryColor   string `json:"primaryColor"`
	SecondaryColor string `json:"secondaryColor"`
	Font           string `json:"font"`
	CustomData     string `json:"customData"`
}

// CustomTemplate is a reusable, organizer-owned custom invite design.
type CustomTemplate struct {
	ID                 string    `json:"id"`
	OrganizerID        string    `json:"-"`
	Name               string    `json:"name"`
	Heading            string    `json:"heading"`
	Body               string    `json:"body"`
	Footer             string    `json:"footer"`
	PrimaryColor       string    `json:"primaryColor"`
	SecondaryColor     string    `json:"secondaryColor"`
	Font               string    `json:"font"`
	CustomData         string    `json:"customData"`
	PhotoData          []byte    `json:"-"`
	PhotoExt           string    `json:"-"`
	CardBackgroundData []byte    `json:"-"`
	CardBackgroundExt  string    `json:"-"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// SaveCustomTemplateRequest snapshots a custom card as a reusable template.
type SaveCustomTemplateRequest struct {
	Name string `json:"name"`
	SaveInviteRequest
}
