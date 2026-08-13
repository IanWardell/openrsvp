package admin

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"

	"github.com/yannkr/openrsvp/internal/auth"
	"github.com/yannkr/openrsvp/internal/config"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrProtected = errors.New("protected account")
	ErrConflict  = errors.New("conflict")
)

type Service struct {
	store         *Store
	authStore     *auth.Store
	authService   *auth.Service
	cfg           *config.Config
	roleMu        sync.Mutex
	onDeleteEvent func(eventID string)
}

// SetOnDeleteEvent registers cleanup for non-database event assets, such as
// uploaded invitation images.
func (s *Service) SetOnDeleteEvent(fn func(eventID string)) { s.onDeleteEvent = fn }

func NewService(store *Store, authStore *auth.Store, authService *auth.Service, cfg *config.Config) *Service {
	return &Service{store: store, authStore: authStore, authService: authService, cfg: cfg}
}
func pagination(page, size, total int) Pagination {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 25
	}
	if size > 100 {
		size = 100
	}
	pages := 0
	if total > 0 {
		pages = (total + size - 1) / size
	}
	return Pagination{page, size, total, pages}
}
func (s *Service) ListUsers(ctx context.Context, search string, page, size int) (Page[UserSummary], error) {
	rows, total, err := s.store.ListUsers(ctx, search, page, size)
	if err != nil {
		return Page[UserSummary]{}, err
	}
	for i := range rows {
		o, err := s.authStore.FindOrganizerByID(ctx, rows[i].ID)
		if err == nil && o != nil {
			s.authService.ApplyEffectiveRole(o)
			rows[i].Role = o.Role
			rows[i].MinimumRole = o.MinimumRole
			rows[i].RoleManagedByEnvironment = o.RoleManagedByEnvironment
		}
	}
	return Page[UserSummary]{rows, pagination(page, size, total)}, nil
}
func (s *Service) ListEvents(ctx context.Context, search, status string, page, size int) (Page[EventSummary], error) {
	rows, total, err := s.store.ListEvents(ctx, search, status, page, size)
	if err == nil {
		for i := range rows {
			if rows[i].PublicURL != "" {
				rows[i].PublicURL = strings.TrimRight(s.cfg.BaseURL, "/") + "/i/" + rows[i].PublicURL
			}
		}
	}
	return Page[EventSummary]{rows, pagination(page, size, total)}, err
}
func (s *Service) ListGuests(ctx context.Context, search string, page, size int, reveal bool) (Page[GuestSummary], error) {
	rows, total, err := s.store.ListGuests(ctx, search, page, size, reveal)
	return Page[GuestSummary]{rows, pagination(page, size, total)}, err
}
func (s *Service) ListAudit(ctx context.Context, page, size int) (Page[AuditEntry], error) {
	rows, total, err := s.store.ListAudit(ctx, page, size)
	return Page[AuditEntry]{rows, pagination(page, size, total)}, err
}

func (s *Service) SuspendUser(ctx context.Context, actor *auth.Organizer, id, reason string, suspend bool, audit AuditEntry) error {
	target, err := s.authStore.FindOrganizerByID(ctx, id)
	if err != nil || target == nil {
		return sqlNotFound(err)
	}
	s.authService.ApplyEffectiveRole(target)
	if actor.ID == id {
		return ErrProtected
	}
	if target.IsSuperAdmin && !actor.IsSuperAdmin {
		return ErrForbidden
	}
	if target.IsSuperAdmin && s.cfg.IsSuperAdminEmail(target.Email) {
		return ErrProtected
	}
	if suspend && target.IsSuperAdmin {
		n, countErr := s.store.CountStoredSuperAdmins(ctx)
		if countErr != nil {
			return countErr
		}
		if n <= 1 && len(s.cfg.SuperAdminEmails) == 0 {
			return ErrProtected
		}
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("reason is required")
	}
	if err = s.store.SetUserSuspension(ctx, id, actor.ID, reason, suspend); err != nil {
		return err
	}
	if suspend {
		_ = s.authStore.DeleteSessionsByOrganizer(ctx, id)
		_ = s.authStore.InvalidateUnusedMagicLinks(ctx, id)
	}
	audit.Action = map[bool]string{true: "user.suspended", false: "user.restored"}[suspend]
	audit.TargetType = "user"
	audit.TargetID = id
	audit.Reason = reason
	return s.store.AddAudit(ctx, audit)
}
func (s *Service) SuspendEvent(ctx context.Context, actor *auth.Organizer, id, reason string, suspend bool, audit AuditEntry) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("reason is required")
	}
	title, _, err := s.store.GetEventIdentity(ctx, id)
	_ = title
	if err != nil {
		return sqlNotFound(err)
	}
	if err = s.store.SetEventSuspension(ctx, id, actor.ID, reason, suspend); err != nil {
		return err
	}
	audit.Action = map[bool]string{true: "event.suspended", false: "event.restored"}[suspend]
	audit.TargetType = "event"
	audit.TargetID = id
	audit.Reason = reason
	return s.store.AddAudit(ctx, audit)
}

func (s *Service) CreateUser(ctx context.Context, actor *auth.Organizer, in CreateUserRequest, audit AuditEntry) (*auth.Organizer, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, auth.ErrInvalidEmail
	}
	existing, err := s.authStore.FindOrganizerByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrConflict
	}
	o, err := s.authStore.CreateOrganizer(ctx, email)
	if err != nil {
		return nil, err
	}
	o.Name = strings.TrimSpace(in.Name)
	if err = s.authStore.UpdateOrganizer(ctx, o); err != nil {
		return nil, err
	}
	_ = s.authStore.MarkInvited(ctx, o.ID)
	if err = s.authService.SendMagicLinkToExisting(ctx, o); err != nil {
		return nil, err
	}
	audit.Action = "user.invited"
	audit.TargetType = "user"
	audit.TargetID = o.ID
	_ = s.store.AddAudit(ctx, audit)
	s.authService.ApplyEffectiveRole(o)
	return o, nil
}
func (s *Service) SendMagicLink(ctx context.Context, actor *auth.Organizer, id string, audit AuditEntry) error {
	o, err := s.authStore.FindOrganizerByID(ctx, id)
	if err != nil || o == nil {
		return sqlNotFound(err)
	}
	if err = s.authService.SendMagicLinkToExisting(ctx, o); err != nil {
		return err
	}
	audit.Action = "user.magic_link_sent"
	audit.TargetType = "user"
	audit.TargetID = id
	return s.store.AddAudit(ctx, audit)
}
func (s *Service) RevokeSessions(ctx context.Context, id string, audit AuditEntry) error {
	if err := s.authStore.DeleteSessionsByOrganizer(ctx, id); err != nil {
		return err
	}
	audit.Action = "user.sessions_revoked"
	audit.TargetType = "user"
	audit.TargetID = id
	return s.store.AddAudit(ctx, audit)
}
func (s *Service) ChangeRole(ctx context.Context, actor *auth.Organizer, id string, in RoleRequest, audit AuditEntry) error {
	s.roleMu.Lock()
	defer s.roleMu.Unlock()
	if in.Role != auth.RoleOrganizer && in.Role != auth.RoleAdmin && in.Role != auth.RoleSuperAdmin {
		return fmt.Errorf("invalid role")
	}
	if strings.TrimSpace(in.Reason) == "" {
		return fmt.Errorf("reason is required")
	}
	target, err := s.authStore.FindOrganizerByID(ctx, id)
	if err != nil || target == nil {
		return sqlNotFound(err)
	}
	s.authService.ApplyEffectiveRole(target)
	if actor.ID == id {
		return ErrProtected
	}
	minimum := auth.RoleOrganizer
	if s.cfg.IsAdminEmail(target.Email) {
		minimum = auth.RoleAdmin
	}
	if s.cfg.IsSuperAdminEmail(target.Email) {
		minimum = auth.RoleSuperAdmin
	}
	if rank(in.Role) < rank(minimum) {
		return ErrProtected
	}
	if target.Role == auth.RoleSuperAdmin && in.Role != auth.RoleSuperAdmin {
		n, e := s.store.CountStoredSuperAdmins(ctx)
		if e != nil {
			return e
		}
		if n <= 1 && len(s.cfg.SuperAdminEmails) == 0 {
			return ErrProtected
		}
	}
	if err = s.authStore.SetRole(ctx, id, in.Role); err != nil {
		return err
	}
	_ = s.authStore.DeleteSessionsByOrganizer(ctx, id)
	audit.Action = "user.role_changed"
	audit.TargetType = "user"
	audit.TargetID = id
	audit.Reason = in.Reason
	audit.Metadata = fmt.Sprintf(`{"role":%q}`, in.Role)
	return s.store.AddAudit(ctx, audit)
}
func rank(role string) int {
	switch role {
	case auth.RoleSuperAdmin:
		return 2
	case auth.RoleAdmin:
		return 1
	default:
		return 0
	}
}
func (s *Service) RevealGuest(ctx context.Context, id string, audit AuditEntry) ([]GuestSummary, error) {
	if strings.TrimSpace(audit.Reason) == "" {
		return nil, fmt.Errorf("reason is required")
	}
	g, err := s.store.GetGuest(ctx, id)
	if err != nil {
		return nil, sqlNotFound(err)
	}
	rows, err := s.store.FindGuestParticipation(ctx, g)
	if err != nil {
		return nil, err
	}
	audit.Action = "guest.revealed"
	audit.TargetType = "guest"
	audit.TargetID = id
	audit.Metadata = fmt.Sprintf(`{"records":%d}`, len(rows))
	if err = s.store.AddAudit(ctx, audit); err != nil {
		return nil, err
	}
	return rows, nil
}
func (s *Service) DeleteEvent(ctx context.Context, actor *auth.Organizer, id string, in DeleteRequest, audit AuditEntry) error {
	title, _, err := s.store.GetEventIdentity(ctx, id)
	if err != nil {
		return sqlNotFound(err)
	}
	if in.Confirmation != title || strings.TrimSpace(in.Reason) == "" {
		return fmt.Errorf("confirmation and reason are required")
	}
	if err = s.store.DeleteEvent(ctx, id); err != nil {
		return err
	}
	if s.onDeleteEvent != nil {
		s.onDeleteEvent(id)
	}
	audit.Action = "event.deleted"
	audit.TargetType = "event"
	audit.TargetID = id
	audit.Reason = in.Reason
	return s.store.AddAudit(ctx, audit)
}
func (s *Service) DeleteUser(ctx context.Context, actor *auth.Organizer, id string, in DeleteRequest, audit AuditEntry) error {
	s.roleMu.Lock()
	defer s.roleMu.Unlock()
	target, err := s.authStore.FindOrganizerByID(ctx, id)
	if err != nil || target == nil {
		return sqlNotFound(err)
	}
	s.authService.ApplyEffectiveRole(target)
	if actor.ID == id || target.RoleManagedByEnvironment {
		return ErrProtected
	}
	if in.Confirmation != target.Email || strings.TrimSpace(in.Reason) == "" {
		return fmt.Errorf("confirmation and reason are required")
	}
	if target.IsSuperAdmin {
		n, e := s.store.CountStoredSuperAdmins(ctx)
		if e != nil {
			return e
		}
		if n <= 1 && len(s.cfg.SuperAdminEmails) == 0 {
			return ErrProtected
		}
	}
	eventIDs, err := s.store.ListOwnedEventIDs(ctx, id)
	if err != nil {
		return err
	}
	if err = s.authService.DeleteAccount(ctx, id); err != nil {
		return err
	}
	if s.onDeleteEvent != nil {
		for _, eventID := range eventIDs {
			s.onDeleteEvent(eventID)
		}
	}
	audit.Action = "user.deleted"
	audit.TargetType = "user"
	audit.TargetID = id
	audit.Reason = in.Reason
	return s.store.AddAudit(ctx, audit)
}
func sqlNotFound(err error) error {
	if err == nil {
		return errors.New("not found")
	}
	return err
}
