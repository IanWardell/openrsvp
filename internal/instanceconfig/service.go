package instanceconfig

import "context"

// Service provides typed access to instance configuration settings.
type Service struct {
	store *Store
}

// NewService creates a new instance config Service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// IsConfigured reports whether the first-run wizard has been completed.
func (s *Service) IsConfigured(ctx context.Context) (bool, error) {
	return s.store.IsConfigured(ctx)
}

// GetSettings returns the full typed settings, falling back to zero values for
// any key that has not been set yet.
func (s *Service) GetSettings(ctx context.Context) (*Settings, error) {
	all, err := s.store.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return &Settings{
		InstanceName:    all[KeyInstanceName],
		DefaultTimezone: all[KeyDefaultTimezone],
		AllowSignups:    all[KeyAllowSignups] == "true",
		SupportEmail:    all[KeySupportEmail],
		Configured:      all[KeyConfigured] == "true",
	}, nil
}

// SaveSettings persists the editable settings and marks the instance configured.
func (s *Service) SaveSettings(ctx context.Context, in *Settings) error {
	pairs := []struct {
		key, value string
	}{
		{KeyInstanceName, in.InstanceName},
		{KeyDefaultTimezone, in.DefaultTimezone},
		{KeyAllowSignups, boolToString(in.AllowSignups)},
		{KeySupportEmail, in.SupportEmail},
		{KeyConfigured, "true"},
	}
	for _, p := range pairs {
		if err := s.store.Set(ctx, p.key, p.value); err != nil {
			return err
		}
	}
	return nil
}

// GetPublicConfig returns the non-sensitive subset for the frontend.
func (s *Service) GetPublicConfig(ctx context.Context) (*PublicConfig, error) {
	all, err := s.store.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return &PublicConfig{
		InstanceName:    all[KeyInstanceName],
		DefaultTimezone: all[KeyDefaultTimezone],
		AllowSignups:    all[KeyAllowSignups] == "true",
		SupportEmail:    all[KeySupportEmail],
	}, nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
