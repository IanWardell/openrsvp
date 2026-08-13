package instanceconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yannkr/openrsvp/internal/testutil"
)

func setupService(t *testing.T) *Service {
	t.Helper()
	return NewService(NewStore(testutil.NewTestDB(t)))
}

func TestService_GetSettings_Empty(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	settings, err := svc.GetSettings(ctx)
	require.NoError(t, err)
	assert.Equal(t, "", settings.InstanceName)
	assert.False(t, settings.AllowSignups)
	assert.False(t, settings.Configured)
}

func TestService_SaveAndGetSettings(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	in := &Settings{
		InstanceName:        "Neighborhood Events",
		DefaultTimezone:     "Europe/Paris",
		AllowSignups:        true,
		SupportEmail:        "help@example.org",
		NotifyNewOrganizer:  true,
		NotifyNewAdmin:      true,
		NotifyNewSuperAdmin: true,
	}
	require.NoError(t, svc.SaveSettings(ctx, in))

	got, err := svc.GetSettings(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Neighborhood Events", got.InstanceName)
	assert.Equal(t, "Europe/Paris", got.DefaultTimezone)
	assert.True(t, got.AllowSignups)
	assert.Equal(t, "help@example.org", got.SupportEmail)
	assert.True(t, got.NotifyNewOrganizer)
	assert.True(t, got.NotifyNewAdmin)
	assert.True(t, got.NotifyNewSuperAdmin)
	assert.True(t, got.Configured, "SaveSettings must set configured=true")
}

func TestService_OrganizerSignupsAllowedUsesFallbackThenLiveOverride(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	allowed, err := svc.OrganizerSignupsAllowed(ctx, true)
	require.NoError(t, err)
	assert.True(t, allowed)

	in := &Settings{InstanceName: "Live", DefaultTimezone: "UTC", AllowSignups: false}
	require.NoError(t, svc.SaveSettings(ctx, in))
	allowed, err = svc.OrganizerSignupsAllowed(ctx, true)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestService_SaveSettings_SetsConfiguredFlag(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	configured, err := svc.IsConfigured(ctx)
	require.NoError(t, err)
	assert.False(t, configured)

	require.NoError(t, svc.SaveSettings(ctx, &Settings{
		InstanceName:    "Hub",
		DefaultTimezone: "UTC",
	}))

	configured, err = svc.IsConfigured(ctx)
	require.NoError(t, err)
	assert.True(t, configured)
}

func TestService_GetPublicConfig(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	require.NoError(t, svc.SaveSettings(ctx, &Settings{
		InstanceName:    "Public Hub",
		DefaultTimezone: "UTC",
		AllowSignups:    false,
		SupportEmail:    "support@example.com",
	}))

	pub, err := svc.GetPublicConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Public Hub", pub.InstanceName)
	assert.Equal(t, "UTC", pub.DefaultTimezone)
	assert.False(t, pub.AllowSignups)
	assert.Equal(t, "support@example.com", pub.SupportEmail)
}
