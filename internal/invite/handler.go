package invite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yannkr/openrsvp/internal/errcode"
)

// OrganizerFromCtx extracts the organizer ID from the request context.
type OrganizerFromCtx func(ctx context.Context) (id string, ok bool)

// EventOwnershipChecker verifies that the given organizer owns the event.
// Returns nil if ownership is confirmed; a non-nil error otherwise.
type EventOwnershipChecker func(ctx context.Context, eventID, organizerID string) error

// Handler holds HTTP handlers for invite card endpoints.
type Handler struct {
	service         *Service
	authMiddleware  func(http.Handler) http.Handler
	organizerFrom   OrganizerFromCtx
	uploadsDir      string
	checkEventOwner EventOwnershipChecker
	logger          zerolog.Logger
}

// NewHandler creates a new invite Handler.
func NewHandler(service *Service, authMiddleware func(http.Handler) http.Handler, organizerFrom OrganizerFromCtx, uploadsDir string, checkEventOwner EventOwnershipChecker, logger zerolog.Logger) *Handler {
	return &Handler{
		service:         service,
		authMiddleware:  authMiddleware,
		organizerFrom:   organizerFrom,
		uploadsDir:      uploadsDir,
		checkEventOwner: checkEventOwner,
		logger:          logger,
	}
}

// Routes returns a chi.Router with all invite card routes mounted.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public route.
	r.Get("/templates", h.handleListTemplates)

	// Authenticated routes.
	r.Group(func(auth chi.Router) {
		auth.Use(h.authMiddleware)
		auth.Get("/event/{eventId}", h.handleGetByEvent)
		auth.Put("/event/{eventId}", h.handleSave)
		auth.Get("/event/{eventId}/preview", h.handlePreview)
		auth.Post("/event/{eventId}/image", h.handleUploadImage)
		auth.Get("/custom-templates", h.handleListCustomTemplates)
		auth.Post("/event/{eventId}/custom-template", h.handleSaveCustomTemplate)
		auth.Post("/event/{eventId}/custom-template/{templateId}/apply", h.handleApplyCustomTemplate)
		auth.Delete("/custom-templates/{templateId}", h.handleDeleteCustomTemplate)
	})

	return r
}

func (h *Handler) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := h.service.ListTemplates()
	writeJSON(w, http.StatusOK, map[string]any{"data": templates})
}

func (h *Handler) handleGetByEvent(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := h.organizerFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")

	if err := h.checkEventOwner(r.Context(), eventID, organizerID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}

	card, err := h.service.GetByEventID(r.Context(), eventID)
	if err != nil {
		if err.Error() == "invite card not found" {
			writeError(w, http.StatusNotFound, "not_found", "invite card not found")
			return
		}
		ref := errcode.Ref()
		h.logger.Error().Err(err).Str("error_code", ref).Str("event_id", eventID).Msg("failed to get invite card")
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred (ref: "+ref+")")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": card})
}

func (h *Handler) handleSave(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := h.organizerFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")

	if err := h.checkEventOwner(r.Context(), eventID, organizerID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}

	var req SaveInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	card, err := h.service.Save(r.Context(), eventID, req)
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid templateId:") {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		ref := errcode.Ref()
		h.logger.Error().Err(err).Str("error_code", ref).Str("event_id", eventID).Msg("failed to save invite card")
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred (ref: "+ref+")")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": card})
}

func (h *Handler) handlePreview(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := h.organizerFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")

	if err := h.checkEventOwner(r.Context(), eventID, organizerID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}

	card, err := h.service.GetPreview(r.Context(), eventID)
	if err != nil {
		ref := errcode.Ref()
		h.logger.Error().Err(err).Str("error_code", ref).Str("event_id", eventID).Msg("failed to get invite preview")
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred (ref: "+ref+")")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": card})
}

func (h *Handler) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := h.organizerFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")

	if err := h.checkEventOwner(r.Context(), eventID, organizerID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}

	// Limit request body to maxUploadSize.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "file too large (max 2MB)")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "missing image file")
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "failed to read file")
		return
	}

	_, ext := detectImageType(data)
	if ext == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid image type (allowed: JPEG, PNG, WebP)")
		return
	}

	slot := r.URL.Query().Get("slot")
	if slot == "" {
		slot = "photo"
	}
	if slot != "photo" && slot != "background" {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid image slot")
		return
	}

	// Remove any existing image in this event's selected slot.
	h.cleanupEventImages(eventID, slot)

	// Write new file.
	filename := fmt.Sprintf("%s_%s_%d%s", eventID, slot, time.Now().UnixNano(), ext)
	filePath := filepath.Join(h.uploadsDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save image")
		return
	}

	url := "/api/v1/uploads/" + filename
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"url": url}})
}

// cleanupEventImages removes existing uploads for one event image slot.
func (h *Handler) cleanupEventImages(eventID, slot string) {
	if h.uploadsDir == "" {
		return
	}
	entries, err := os.ReadDir(h.uploadsDir)
	if err != nil {
		return
	}
	prefix := eventID + "_" + slot + "_"
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			_ = os.Remove(filepath.Join(h.uploadsDir, entry.Name()))
		}
	}
}

func (h *Handler) handleListCustomTemplates(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := h.organizerFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	templates, err := h.service.ListCustomTemplates(r.Context(), organizerID)
	if err != nil {
		ref := errcode.Ref()
		h.logger.Error().Err(err).Str("error_code", ref).Str("organizer_id", organizerID).Msg("failed to list custom invite templates")
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred (ref: "+ref+")")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": templates})
}

func (h *Handler) handleSaveCustomTemplate(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := h.organizerFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	eventID := chi.URLParam(r, "eventId")
	if err := h.checkEventOwner(r.Context(), eventID, organizerID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	var req SaveCustomTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	tmpl, err := h.service.SaveCustomTemplate(r.Context(), organizerID, eventID, req)
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "only custom cards") || strings.HasPrefix(err.Error(), "invalid templateId:") || strings.Contains(err.Error(), "not owned by this event") || strings.Contains(err.Error(), "invalid type") {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		ref := errcode.Ref()
		h.logger.Error().Err(err).Str("error_code", ref).Str("event_id", eventID).Msg("failed to save custom invite template")
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred (ref: "+ref+")")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": tmpl})
}

func (h *Handler) handleApplyCustomTemplate(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := h.organizerFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	eventID := chi.URLParam(r, "eventId")
	if err := h.checkEventOwner(r.Context(), eventID, organizerID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	card, err := h.service.ApplyCustomTemplate(r.Context(), organizerID, eventID, chi.URLParam(r, "templateId"))
	if err != nil {
		if err.Error() == "custom template not found" {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		ref := errcode.Ref()
		h.logger.Error().Err(err).Str("error_code", ref).Str("event_id", eventID).Msg("failed to apply custom invite template")
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred (ref: "+ref+")")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": card})
}

func (h *Handler) handleDeleteCustomTemplate(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := h.organizerFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := h.service.DeleteCustomTemplate(r.Context(), organizerID, chi.URLParam(r, "templateId")); err != nil {
		if err.Error() == "custom template not found" {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		ref := errcode.Ref()
		h.logger.Error().Err(err).Str("error_code", ref).Str("organizer_id", organizerID).Msg("failed to delete custom invite template")
		writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred (ref: "+ref+")")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   errCode,
		"message": message,
	})
}
