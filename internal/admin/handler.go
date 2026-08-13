package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/yannkr/openrsvp/internal/auth"
)

type Handler struct {
	service                  *Service
	authMw, adminMw, superMw func(http.Handler) http.Handler
	stats                    http.HandlerFunc
	logger                   zerolog.Logger
}

func NewHandler(s *Service, authMw, adminMw, superMw func(http.Handler) http.Handler, stats http.HandlerFunc, l zerolog.Logger) *Handler {
	return &Handler{s, authMw, adminMw, superMw, stats, l}
}
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(h.authMw)
	r.Use(h.adminMw)
	r.Get("/stats", h.stats)
	r.Get("/events", h.listEvents)
	r.Get("/users", h.listUsers)
	r.Post("/events/{id}/suspend", h.suspendEvent(true))
	r.Post("/events/{id}/restore", h.suspendEvent(false))
	r.Post("/users/{id}/suspend", h.suspendUser(true))
	r.Post("/users/{id}/restore", h.suspendUser(false))
	r.Group(func(sr chi.Router) {
		sr.Use(h.superMw)
		sr.Post("/users", h.createUser)
		sr.Patch("/users/{id}/role", h.changeRole)
		sr.Post("/users/{id}/magic-link", h.magicLink)
		sr.Post("/users/{id}/sessions/revoke", h.revokeSessions)
		sr.Get("/invitations/events", h.listEvents)
		sr.Get("/invitations/guests", h.listGuests(false))
		sr.Post("/invitations/guests/reveal-page", h.listGuests(true))
		sr.Post("/invitations/guests/{id}/reveal", h.revealGuest)
		sr.Get("/audit-log", h.listAudit)
		sr.Delete("/events/{id}", h.deleteEvent)
		sr.Delete("/users/{id}", h.deleteUser)
	})
	return r
}
func page(r *http.Request) (int, int) {
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	s, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if p < 1 {
		p = 1
	}
	if s < 1 {
		s = 25
	}
	return p, s
}
func audit(r *http.Request) *AuditEntry {
	o := auth.OrganizerFromContext(r.Context())
	return &AuditEntry{ActorID: o.ID, ActorRole: o.Role, Metadata: "{}", RequestID: middleware.GetReqID(r.Context()), RemoteIP: r.RemoteAddr}
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		write(w, 400, map[string]string{"error": "bad_request", "message": "invalid request body"})
		return false
	}
	return true
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (h *Handler) fail(w http.ResponseWriter, err error) {
	status := 500
	code := "internal_error"
	msg := "an internal error occurred"
	switch {
	case errors.Is(err, ErrForbidden):
		status = 403
		code = "forbidden"
		msg = err.Error()
	case errors.Is(err, ErrProtected), errors.Is(err, ErrConflict):
		status = 409
		code = "conflict"
		msg = err.Error()
	case strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid"):
		status = 400
		code = "bad_request"
		msg = err.Error()
	case strings.Contains(err.Error(), "not found"):
		status = 404
		code = "not_found"
		msg = "not found"
	}
	write(w, status, map[string]string{"error": code, "message": msg})
}
func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	p, s := page(r)
	v, e := h.service.ListUsers(r.Context(), r.URL.Query().Get("search"), p, s)
	if e != nil {
		h.fail(w, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	p, s := page(r)
	v, e := h.service.ListEvents(r.Context(), r.URL.Query().Get("search"), r.URL.Query().Get("status"), p, s)
	if e != nil {
		h.fail(w, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) listGuests(reveal bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, s := page(r)
		a := audit(r)
		if reveal {
			var in ActionRequest
			if !decode(w, r, &in) {
				return
			}
			a.Reason = in.Reason
			if strings.TrimSpace(in.Reason) == "" {
				h.fail(w, errors.New("reason is required"))
				return
			}
			a.Action = "guest.page_revealed"
			a.TargetType = "guest_page"
			a.TargetID = "current"
			_ = h.service.store.AddAudit(r.Context(), *a)
		}
		v, e := h.service.ListGuests(r.Context(), r.URL.Query().Get("search"), p, s, reveal)
		if e != nil {
			h.fail(w, e)
			return
		}
		if reveal {
			w.Header().Set("Cache-Control", "no-store")
		}
		write(w, 200, v)
	}
}
func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	p, s := page(r)
	v, e := h.service.ListAudit(r.Context(), p, s)
	if e != nil {
		h.fail(w, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) suspendUser(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in ActionRequest
		if !decode(w, r, &in) {
			return
		}
		a := audit(r)
		e := h.service.SuspendUser(r.Context(), auth.OrganizerFromContext(r.Context()), chi.URLParam(r, "id"), in.Reason, on, *a)
		if e != nil {
			h.fail(w, e)
			return
		}
		write(w, 200, map[string]string{"message": "updated"})
	}
}
func (h *Handler) suspendEvent(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in ActionRequest
		if !decode(w, r, &in) {
			return
		}
		a := audit(r)
		e := h.service.SuspendEvent(r.Context(), auth.OrganizerFromContext(r.Context()), chi.URLParam(r, "id"), in.Reason, on, *a)
		if e != nil {
			h.fail(w, e)
			return
		}
		write(w, 200, map[string]string{"message": "updated"})
	}
}
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var in CreateUserRequest
	if !decode(w, r, &in) {
		return
	}
	a := audit(r)
	o, e := h.service.CreateUser(r.Context(), auth.OrganizerFromContext(r.Context()), in, *a)
	if e != nil {
		h.fail(w, e)
		return
	}
	write(w, 201, map[string]any{"data": o})
}
func (h *Handler) changeRole(w http.ResponseWriter, r *http.Request) {
	var in RoleRequest
	if !decode(w, r, &in) {
		return
	}
	a := audit(r)
	e := h.service.ChangeRole(r.Context(), auth.OrganizerFromContext(r.Context()), chi.URLParam(r, "id"), in, *a)
	if e != nil {
		h.fail(w, e)
		return
	}
	write(w, 200, map[string]string{"message": "role updated"})
}
func (h *Handler) magicLink(w http.ResponseWriter, r *http.Request) {
	a := audit(r)
	if e := h.service.SendMagicLink(r.Context(), auth.OrganizerFromContext(r.Context()), chi.URLParam(r, "id"), *a); e != nil {
		h.fail(w, e)
		return
	}
	write(w, 200, map[string]string{"message": "magic link sent"})
}
func (h *Handler) revokeSessions(w http.ResponseWriter, r *http.Request) {
	a := audit(r)
	if e := h.service.RevokeSessions(r.Context(), chi.URLParam(r, "id"), *a); e != nil {
		h.fail(w, e)
		return
	}
	write(w, 200, map[string]string{"message": "sessions revoked"})
}
func (h *Handler) revealGuest(w http.ResponseWriter, r *http.Request) {
	var in ActionRequest
	if !decode(w, r, &in) {
		return
	}
	a := audit(r)
	a.Reason = in.Reason
	v, e := h.service.RevealGuest(r.Context(), chi.URLParam(r, "id"), *a)
	if e != nil {
		h.fail(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	write(w, 200, map[string]any{"data": v})
}
func (h *Handler) deleteEvent(w http.ResponseWriter, r *http.Request) {
	var in DeleteRequest
	if !decode(w, r, &in) {
		return
	}
	a := audit(r)
	if e := h.service.DeleteEvent(r.Context(), auth.OrganizerFromContext(r.Context()), chi.URLParam(r, "id"), in, *a); e != nil {
		h.fail(w, e)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	var in DeleteRequest
	if !decode(w, r, &in) {
		return
	}
	a := audit(r)
	if e := h.service.DeleteUser(r.Context(), auth.OrganizerFromContext(r.Context()), chi.URLParam(r, "id"), in, *a); e != nil {
		h.fail(w, e)
		return
	}
	w.WriteHeader(204)
}
