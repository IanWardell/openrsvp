package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yannkr/openrsvp/internal/database"
)

type Store struct{ db database.DB }

func NewStore(db database.DB) *Store { return &Store{db: db} }

func pageValues(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 25
	}
	if size > 100 {
		size = 100
	}
	return size, (page - 1) * size
}
func like(search string) string {
	return "%" + strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(search)), "%", "\\%"), "_", "\\_") + "%"
}
func parseOptionalTime(v sql.NullString) *time.Time {
	if !v.Valid {
		return nil
	}
	t, e := time.Parse(time.RFC3339, v.String)
	if e != nil {
		return nil
	}
	return &t
}

func (s *Store) ListUsers(ctx context.Context, search string, page, size int) ([]UserSummary, int, error) {
	limit, offset := pageValues(page, size)
	pattern := like(search)
	where := ""
	args := []any{}
	if strings.TrimSpace(search) != "" {
		where = " WHERE LOWER(o.email) LIKE ? ESCAPE '\\' OR LOWER(o.name) LIKE ? ESCAPE '\\'"
		args = []any{pattern, pattern}
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM organizers o"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `SELECT o.id,o.email,o.name,o.role,o.suspended_at,o.suspension_reason,o.created_at,o.last_login_at,
	 (SELECT COUNT(*) FROM events e WHERE e.organizer_id=o.id),
	 (SELECT COUNT(*) FROM event_cohosts c WHERE c.organizer_id=o.id)
	 FROM organizers o`+where+` ORDER BY o.created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []UserSummary{}
	for rows.Next() {
		var u UserSummary
		var suspended, last sql.NullString
		var created string
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.StoredRole, &suspended, &u.SuspensionReason, &created, &last, &u.OwnedEvents, &u.CohostedEvents); err != nil {
			return nil, 0, err
		}
		u.Role = u.StoredRole
		u.SuspendedAt = parseOptionalTime(suspended)
		u.LastLoginAt = parseOptionalTime(last)
		u.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func (s *Store) ListEvents(ctx context.Context, search, status string, page, size int) ([]EventSummary, int, error) {
	limit, offset := pageValues(page, size)
	clauses := []string{"1=1"}
	args := []any{}
	if strings.TrimSpace(search) != "" {
		p := like(search)
		clauses = append(clauses, "(LOWER(e.title) LIKE ? ESCAPE '\\' OR LOWER(o.email) LIKE ? ESCAPE '\\' OR LOWER(o.name) LIKE ? ESCAPE '\\')")
		args = append(args, p, p, p)
	}
	if status != "" {
		clauses = append(clauses, "e.status = ?")
		args = append(args, status)
	}
	where := " WHERE " + strings.Join(clauses, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events e JOIN organizers o ON o.id=e.organizer_id"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	q := `SELECT e.id,e.title,e.organizer_id,o.email,o.name,e.status,e.event_date,e.timezone,e.suspended_at,e.suspension_reason,
	 o.suspended_at,COALESCE(i.template_id,''),i.updated_at,e.share_token,
	 COALESCE(SUM(CASE WHEN a.rsvp_status='attending' THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.rsvp_status='maybe' THEN 1 ELSE 0 END),0),
	 COALESCE(SUM(CASE WHEN a.rsvp_status='declined' THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.rsvp_status='pending' THEN 1 ELSE 0 END),0),
	 COALESCE(SUM(CASE WHEN a.rsvp_status='waitlisted' THEN 1 ELSE 0 END),0),COUNT(a.id),COALESCE(SUM(CASE WHEN a.rsvp_status='attending' THEN 1+a.plus_ones ELSE 0 END),0)
	 FROM events e JOIN organizers o ON o.id=e.organizer_id LEFT JOIN invite_cards i ON i.event_id=e.id LEFT JOIN attendees a ON a.event_id=e.id` + where + `
	 GROUP BY e.id,e.title,e.organizer_id,o.email,o.name,e.status,e.event_date,e.timezone,e.suspended_at,e.suspension_reason,o.suspended_at,i.template_id,i.updated_at,e.share_token ORDER BY e.event_date DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []EventSummary{}
	for rows.Next() {
		var e EventSummary
		var eventDate string
		var suspended, ownerSuspended, inviteUpdated sql.NullString
		var share string
		if err := rows.Scan(&e.ID, &e.Title, &e.OrganizerID, &e.OrganizerEmail, &e.OrganizerName, &e.Status, &eventDate, &e.Timezone, &suspended, &e.SuspensionReason, &ownerSuspended, &e.TemplateID, &inviteUpdated, &share, &e.Attending, &e.Maybe, &e.Declined, &e.Pending, &e.Waitlisted, &e.Responses, &e.Headcount); err != nil {
			return nil, 0, err
		}
		e.EventDate, _ = time.Parse(time.RFC3339, eventDate)
		e.SuspendedAt = parseOptionalTime(suspended)
		e.OwnerSuspended = ownerSuspended.Valid
		e.EffectiveSuspended = e.SuspendedAt != nil || e.OwnerSuspended
		e.InviteUpdatedAt = parseOptionalTime(inviteUpdated)
		if e.Status == "published" && !e.EffectiveSuspended {
			e.PublicURL = share
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func (s *Store) ListGuests(ctx context.Context, search string, page, size int, reveal bool) ([]GuestSummary, int, error) {
	limit, offset := pageValues(page, size)
	where := ""
	args := []any{}
	if strings.TrimSpace(search) != "" {
		p := like(search)
		where = " WHERE LOWER(a.name) LIKE ? ESCAPE '\\' OR LOWER(COALESCE(a.email,'')) LIKE ? ESCAPE '\\' OR LOWER(COALESCE(a.phone,'')) LIKE ? ESCAPE '\\'"
		args = []any{p, p, p}
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM attendees a"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.event_id,e.title,a.name,COALESCE(a.email,''),COALESCE(a.phone,''),a.rsvp_status,a.plus_ones,COALESCE(a.import_source,''),a.dietary_notes,a.created_at FROM attendees a JOIN events e ON e.id=a.event_id`+where+` ORDER BY a.created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []GuestSummary{}
	for rows.Next() {
		var g GuestSummary
		var created string
		if err := rows.Scan(&g.ID, &g.EventID, &g.EventTitle, &g.Name, &g.Email, &g.Phone, &g.RSVPStatus, &g.PlusOnes, &g.ImportSource, &g.DietaryNotes, &created); err != nil {
			return nil, 0, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if !reveal {
			g.Name = maskName(g.Name)
			g.Email = maskEmail(g.Email)
			g.Phone = maskPhone(g.Phone)
			g.DietaryNotes = ""
		}
		out = append(out, g)
	}
	return out, total, rows.Err()
}
func maskName(v string) string {
	r := []rune(v)
	if len(r) < 2 {
		return "*"
	}
	return string(r[0]) + strings.Repeat("*", len(r)-1)
}
func maskEmail(v string) string {
	p := strings.SplitN(v, "@", 2)
	if len(p) != 2 {
		return ""
	}
	return maskName(p[0]) + "@" + p[1]
}
func maskPhone(v string) string {
	if len(v) < 4 {
		return strings.Repeat("*", len(v))
	}
	return strings.Repeat("*", len(v)-4) + v[len(v)-4:]
}

func (s *Store) SetUserSuspension(ctx context.Context, id, actor, reason string, suspend bool) error {
	var at any = nil
	by := ""
	why := ""
	if suspend {
		at = time.Now().UTC().Format(time.RFC3339)
		by = actor
		why = reason
	}
	_, err := s.db.ExecContext(ctx, "UPDATE organizers SET suspended_at=?,suspended_by=?,suspension_reason=?,updated_at=? WHERE id=?", at, by, why, time.Now().UTC().Format(time.RFC3339), id)
	return err
}
func (s *Store) SetEventSuspension(ctx context.Context, id, actor, reason string, suspend bool) error {
	var at any = nil
	by := ""
	why := ""
	if suspend {
		at = time.Now().UTC().Format(time.RFC3339)
		by = actor
		why = reason
	}
	_, err := s.db.ExecContext(ctx, "UPDATE events SET suspended_at=?,suspended_by=?,suspension_reason=?,updated_at=? WHERE id=?", at, by, why, time.Now().UTC().Format(time.RFC3339), id)
	return err
}
func (s *Store) AddAudit(ctx context.Context, a AuditEntry) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO admin_audit_log(id,actor_id,actor_role,action,target_type,target_id,reason,metadata,request_id,remote_ip,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, uuid.Must(uuid.NewV7()).String(), a.ActorID, a.ActorRole, a.Action, a.TargetType, a.TargetID, a.Reason, a.Metadata, a.RequestID, a.RemoteIP, time.Now().UTC().Format(time.RFC3339))
	return err
}
func (s *Store) ListAudit(ctx context.Context, page, size int) ([]AuditEntry, int, error) {
	limit, offset := pageValues(page, size)
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_audit_log").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.actor_id,COALESCE(actor.email,''),a.actor_role,a.action,a.target_type,a.target_id,
	 CASE a.target_type
	  WHEN 'user' THEN COALESCE((SELECT email FROM organizers WHERE id=a.target_id),'')
	  WHEN 'event' THEN COALESCE((SELECT title FROM events WHERE id=a.target_id),'')
	  WHEN 'guest' THEN COALESCE((SELECT name FROM attendees WHERE id=a.target_id),'')
	  WHEN 'guest_page' THEN 'Current guest results'
	  ELSE '' END,
	 a.reason,a.metadata,a.request_id,a.remote_ip,a.created_at
	 FROM admin_audit_log a LEFT JOIN organizers actor ON actor.id=a.actor_id ORDER BY a.created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []AuditEntry{}
	for rows.Next() {
		var a AuditEntry
		var created string
		if err := rows.Scan(&a.ID, &a.ActorID, &a.ActorEmail, &a.ActorRole, &a.Action, &a.TargetType, &a.TargetID, &a.TargetLabel, &a.Reason, &a.Metadata, &a.RequestID, &a.RemoteIP, &created); err != nil {
			return nil, 0, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, a)
	}
	return out, total, rows.Err()
}
func (s *Store) CountStoredSuperAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM organizers WHERE role='super_admin' AND suspended_at IS NULL").Scan(&n)
	return n, err
}
func (s *Store) DeleteEvent(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM events WHERE id=?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("event not found")
	}
	return nil
}

func (s *Store) ListOwnedEventIDs(ctx context.Context, organizerID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM events WHERE organizer_id=?", organizerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) GetEventIdentity(ctx context.Context, id string) (title string, ownerID string, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT title,organizer_id FROM events WHERE id=?", id).Scan(&title, &ownerID)
	return
}
func (s *Store) GetGuest(ctx context.Context, id string) (*GuestSummary, error) {
	row := s.db.QueryRowContext(ctx, `SELECT a.id,a.event_id,e.title,a.name,COALESCE(a.email,''),COALESCE(a.phone,''),a.rsvp_status,a.plus_ones,COALESCE(a.import_source,''),a.dietary_notes,a.created_at FROM attendees a JOIN events e ON e.id=a.event_id WHERE a.id=?`, id)
	var g GuestSummary
	var created string
	if err := row.Scan(&g.ID, &g.EventID, &g.EventTitle, &g.Name, &g.Email, &g.Phone, &g.RSVPStatus, &g.PlusOnes, &g.ImportSource, &g.DietaryNotes, &created); err != nil {
		return nil, err
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &g, nil
}
func (s *Store) FindGuestParticipation(ctx context.Context, g *GuestSummary) ([]GuestSummary, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.event_id,e.title,a.name,COALESCE(a.email,''),COALESCE(a.phone,''),a.rsvp_status,a.plus_ones,COALESCE(a.import_source,''),a.dietary_notes,a.created_at FROM attendees a JOIN events e ON e.id=a.event_id WHERE (? <> '' AND LOWER(a.email)=LOWER(?)) OR (? <> '' AND a.phone=?) OR LOWER(a.name)=LOWER(?) ORDER BY a.created_at DESC`, g.Email, g.Email, g.Phone, g.Phone, g.Name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []GuestSummary{}
	for rows.Next() {
		var x GuestSummary
		var created string
		if err := rows.Scan(&x.ID, &x.EventID, &x.EventTitle, &x.Name, &x.Email, &x.Phone, &x.RSVPStatus, &x.PlusOnes, &x.ImportSource, &x.DietaryNotes, &created); err != nil {
			return nil, err
		}
		x.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, x)
	}
	return out, rows.Err()
}
