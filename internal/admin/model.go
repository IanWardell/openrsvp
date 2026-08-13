package admin

import "time"

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}
type Page[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type UserSummary struct {
	ID                       string     `json:"id"`
	Email                    string     `json:"email"`
	Name                     string     `json:"name"`
	Role                     string     `json:"role"`
	StoredRole               string     `json:"storedRole"`
	MinimumRole              string     `json:"minimumRole"`
	RoleManagedByEnvironment bool       `json:"roleManagedByEnvironment"`
	SuspendedAt              *time.Time `json:"suspendedAt,omitempty"`
	SuspensionReason         string     `json:"suspensionReason,omitempty"`
	CreatedAt                time.Time  `json:"createdAt"`
	LastLoginAt              *time.Time `json:"lastLoginAt,omitempty"`
	OwnedEvents              int        `json:"ownedEvents"`
	CohostedEvents           int        `json:"cohostedEvents"`
}

type EventSummary struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	OrganizerID        string     `json:"organizerId"`
	OrganizerEmail     string     `json:"organizerEmail"`
	OrganizerName      string     `json:"organizerName"`
	Status             string     `json:"status"`
	EventDate          time.Time  `json:"eventDate"`
	Timezone           string     `json:"timezone"`
	SuspendedAt        *time.Time `json:"suspendedAt,omitempty"`
	SuspensionReason   string     `json:"suspensionReason,omitempty"`
	OwnerSuspended     bool       `json:"ownerSuspended"`
	EffectiveSuspended bool       `json:"effectiveSuspended"`
	TemplateID         string     `json:"templateId"`
	InviteUpdatedAt    *time.Time `json:"inviteUpdatedAt,omitempty"`
	Attending          int        `json:"attending"`
	Maybe              int        `json:"maybe"`
	Declined           int        `json:"declined"`
	Pending            int        `json:"pending"`
	Waitlisted         int        `json:"waitlisted"`
	Responses          int        `json:"responses"`
	Headcount          int        `json:"headcount"`
	PublicURL          string     `json:"publicUrl,omitempty"`
}

type GuestSummary struct {
	ID           string    `json:"id"`
	EventID      string    `json:"eventId"`
	EventTitle   string    `json:"eventTitle"`
	Name         string    `json:"name"`
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	RSVPStatus   string    `json:"rsvpStatus"`
	PlusOnes     int       `json:"plusOnes"`
	ImportSource string    `json:"importSource,omitempty"`
	DietaryNotes string    `json:"dietaryNotes,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type AuditEntry struct {
	ID          string    `json:"id"`
	ActorID     string    `json:"actorId"`
	ActorEmail  string    `json:"actorEmail,omitempty"`
	ActorRole   string    `json:"actorRole"`
	Action      string    `json:"action"`
	TargetType  string    `json:"targetType"`
	TargetID    string    `json:"targetId"`
	TargetLabel string    `json:"targetLabel,omitempty"`
	Reason      string    `json:"reason"`
	Metadata    string    `json:"metadata"`
	RequestID   string    `json:"requestId"`
	RemoteIP    string    `json:"remoteIp"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ActionRequest struct {
	Reason string `json:"reason"`
}
type RoleRequest struct {
	Role   string `json:"role"`
	Reason string `json:"reason"`
}
type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
type DeleteRequest struct {
	Confirmation string `json:"confirmation"`
	Reason       string `json:"reason"`
}
