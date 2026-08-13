package admin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yannkr/openrsvp/internal/auth"
	"github.com/yannkr/openrsvp/internal/event"
	"github.com/yannkr/openrsvp/internal/testutil"
)

func newTestService(t *testing.T) (*Service, *auth.Store, *auth.Service, *event.Service) {
	t.Helper()
	db := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	authStore := auth.NewStore(db)
	authService := auth.NewService(authStore, cfg, zerolog.Nop())
	return NewService(NewStore(db), authStore, authService, cfg), authStore, authService, event.NewService(event.NewStore(db), cfg.DefaultRetentionDays)
}

func createTestOrganizer(t *testing.T, store *auth.Store, email, role string) *auth.Organizer {
	t.Helper()
	o, err := store.CreateOrganizer(context.Background(), email)
	require.NoError(t, err)
	if role != auth.RoleOrganizer {
		require.NoError(t, store.SetRole(context.Background(), o.ID, role))
		o, err = store.FindOrganizerByID(context.Background(), o.ID)
		require.NoError(t, err)
	}
	return o
}

func TestPaginationUsesAPICasing(t *testing.T) {
	b, err := json.Marshal(pagination(2, 25, 51))
	require.NoError(t, err)
	assert.JSONEq(t, `{"page":2,"pageSize":25,"total":51,"totalPages":3}`, string(b))
}

func TestGuestMasking(t *testing.T) {
	assert.Equal(t, "I**", maskName("Ian"))
	assert.Equal(t, "i**@example.com", maskEmail("ian@example.com"))
	assert.Equal(t, "********4567", maskPhone("+15551234567"))
}

func TestSuspendUserRevokesAccessAndHidesOwnedEvents(t *testing.T) {
	svc, authStore, authService, eventService := newTestService(t)
	ctx := context.Background()
	actor := createTestOrganizer(t, authStore, "root@example.com", auth.RoleSuperAdmin)
	authService.ApplyEffectiveRole(actor)
	target := createTestOrganizer(t, authStore, "owner@example.com", auth.RoleOrganizer)
	ev, err := eventService.Create(ctx, target.ID, event.CreateEventRequest{Title: "Hidden event", EventDate: "2026-09-01T18:00:00Z"})
	require.NoError(t, err)

	err = svc.SuspendUser(ctx, actor, target.ID, "abuse report", true, AuditEntry{ActorID: actor.ID, ActorRole: actor.Role, Metadata: "{}"})
	require.NoError(t, err)

	_, err = eventService.GetByShareToken(ctx, ev.ShareToken)
	assert.ErrorContains(t, err, "not found")
	newTitle := "Changed"
	_, err = eventService.Update(ctx, ev.ID, target.ID, event.UpdateEventRequest{Title: &newTitle})
	assert.ErrorContains(t, err, "suspended")

	target, err = authStore.FindOrganizerByID(ctx, target.ID)
	require.NoError(t, err)
	require.NotNil(t, target.SuspendedAt)
	auditPage, err := svc.ListAudit(ctx, 1, 25)
	require.NoError(t, err)
	require.Len(t, auditPage.Data, 1)
	assert.Equal(t, "user.suspended", auditPage.Data[0].Action)
	assert.Equal(t, actor.Email, auditPage.Data[0].ActorEmail)
	assert.Equal(t, target.Email, auditPage.Data[0].TargetLabel)
}

func TestEventSummarySeparatesResponsesFromAttendingHeadcount(t *testing.T) {
	svc, authStore, _, eventService := newTestService(t)
	ctx := context.Background()
	owner := createTestOrganizer(t, authStore, "owner@example.com", auth.RoleOrganizer)
	ev, err := eventService.Create(ctx, owner.ID, event.CreateEventRequest{Title: "Garden supper", EventDate: "2026-10-17T22:30:00Z"})
	require.NoError(t, err)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = svc.store.db.ExecContext(ctx, `INSERT INTO attendees(id,event_id,name,email,rsvp_status,rsvp_token,contact_method,dietary_notes,plus_ones,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		"guest-1", ev.ID, "Jamie", "jamie@example.com", "maybe", "token-1", "email", "", 0, now, now)
	require.NoError(t, err)

	page, err := svc.ListEvents(ctx, "Garden", "", 1, 25)
	require.NoError(t, err)
	require.Len(t, page.Data, 1)
	assert.Equal(t, 1, page.Data[0].Responses)
	assert.Equal(t, 1, page.Data[0].Maybe)
	assert.Equal(t, 0, page.Data[0].Headcount)
}

func TestEnvironmentSuperAdminCannotBeSuspended(t *testing.T) {
	svc, authStore, authService, _ := newTestService(t)
	ctx := context.Background()
	actor := createTestOrganizer(t, authStore, "root@example.com", auth.RoleSuperAdmin)
	authService.ApplyEffectiveRole(actor)
	target := createTestOrganizer(t, authStore, "protected@example.com", auth.RoleOrganizer)
	svc.cfg.SuperAdminEmails = []string{target.Email}

	err := svc.SuspendUser(ctx, actor, target.ID, "test", true, AuditEntry{ActorID: actor.ID, ActorRole: actor.Role, Metadata: "{}"})
	assert.ErrorIs(t, err, ErrProtected)
}

func TestAdminCannotSuspendSuperAdminAccount(t *testing.T) {
	svc, authStore, authService, _ := newTestService(t)
	ctx := context.Background()
	actor := createTestOrganizer(t, authStore, "admin@example.com", auth.RoleAdmin)
	authService.ApplyEffectiveRole(actor)
	target := createTestOrganizer(t, authStore, "root@example.com", auth.RoleSuperAdmin)

	err := svc.SuspendUser(ctx, actor, target.ID, "test", true, AuditEntry{ActorID: actor.ID, ActorRole: actor.Role, Metadata: "{}"})
	assert.ErrorIs(t, err, ErrForbidden)
}
