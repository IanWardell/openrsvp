package instanceconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yannkr/openrsvp/internal/testutil"
)

func setupStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(testutil.NewTestDB(t))
}

func TestStore_GetAbsentKey(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	_, ok, err := store.Get(ctx, KeyInstanceName)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestStore_SetAndGet(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	require.NoError(t, store.Set(ctx, KeyInstanceName, "My Party Hub"))

	value, ok, err := store.Get(ctx, KeyInstanceName)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "My Party Hub", value)
}

func TestStore_SetUpsert(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	require.NoError(t, store.Set(ctx, KeyInstanceName, "First"))
	require.NoError(t, store.Set(ctx, KeyInstanceName, "Second"))

	value, ok, err := store.Get(ctx, KeyInstanceName)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Second", value)

	all, err := store.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1, "upsert must not create duplicate rows")
}

func TestStore_GetAll(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	require.NoError(t, store.Set(ctx, KeyInstanceName, "Hub"))
	require.NoError(t, store.Set(ctx, KeyDefaultTimezone, "America/New_York"))

	all, err := store.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		KeyInstanceName:    "Hub",
		KeyDefaultTimezone: "America/New_York",
	}, all)
}

func TestStore_IsConfigured(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	configured, err := store.IsConfigured(ctx)
	require.NoError(t, err)
	assert.False(t, configured)

	require.NoError(t, store.Set(ctx, KeyConfigured, "true"))

	configured, err = store.IsConfigured(ctx)
	require.NoError(t, err)
	assert.True(t, configured)
}
