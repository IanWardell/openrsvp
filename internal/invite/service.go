package invite

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// colorPattern matches a CSS color in #RGB, #RRGGBB, or #RRGGBBAA form.
// Anything that does not match is rejected (named colors like "red" and
// rgb()/hsl() syntax are intentionally excluded because they widen the
// attack surface for CSS-context breakouts via ; and {}.
var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

// fontPattern matches a comma-separated font-family list of single-quoted,
// double-quoted, or bare identifiers. Bare identifiers may contain letters,
// digits, hyphens, and spaces only. Anything outside this grammar is
// rejected to defeat CSS-context injection via the --card-font variable.
var fontPattern = regexp.MustCompile(`^[A-Za-z0-9 ,\-_'"]+$`)

// sanitizeColor returns the trimmed color value when it matches the
// allowlisted #hex format. Empty strings are returned as-is so that
// the default-fill logic in Save() can substitute the template default.
func sanitizeColor(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return ""
	}
	if colorPattern.MatchString(c) {
		return c
	}
	return ""
}

// sanitizeFont returns the trimmed font value when it matches the
// allowlist; otherwise returns empty so the default font is used.
func sanitizeFont(f string) string {
	f = strings.TrimSpace(f)
	if f == "" || len(f) > 100 {
		return ""
	}
	if fontPattern.MatchString(f) {
		return f
	}
	return ""
}

// sanitizeCustomData re-encodes the JSON blob with validated image URLs,
// layout values, colors, fonts, and text-area settings. If the JSON is
// invalid, an empty object is returned.
func sanitizeCustomData(raw string) string {
	if raw == "" {
		return "{}"
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return "{}"
	}
	if bg, ok := data["backgroundImage"].(string); ok {
		if cleaned := sanitizeBackgroundURL(bg); cleaned == "" {
			delete(data, "backgroundImage")
		} else {
			data["backgroundImage"] = cleaned
		}
	}
	if qr, ok := data["qr"].(map[string]any); ok {
		cleanQR := map[string]any{}
		if enabled, exists := qr["enabled"].(bool); exists {
			cleanQR["enabled"] = enabled
		}
		if value, exists := qr["url"].(string); exists {
			if cleaned := sanitizeExternalURL(value); cleaned != "" {
				cleanQR["url"] = cleaned
			}
		}
		for _, key := range []string{"foreground", "background"} {
			if value, exists := qr[key].(string); exists {
				if cleaned := sanitizeColor(value); cleaned != "" {
					cleanQR[key] = cleaned
				}
			}
		}
		if caption, exists := qr["caption"].(string); exists {
			cleanQR["caption"] = truncateRunes(strings.TrimSpace(caption), 120)
		}
		for _, key := range []string{"x", "y"} {
			if value, exists := qr[key].(float64); exists {
				cleanQR[key] = clampFloat(value, 0, 90)
			}
		}
		if value, exists := qr["size"].(float64); exists {
			cleanQR["size"] = clampFloat(value, 10, 28)
		}
		data["qr"] = cleanQR
	}
	if photo, ok := data["photo"].(map[string]any); ok {
		cleanPhoto := map[string]any{}
		if fit, exists := photo["fit"].(string); exists && (fit == "cover" || fit == "contain") {
			cleanPhoto["fit"] = fit
		}
		for _, key := range []string{"x", "y"} {
			if value, exists := photo[key].(float64); exists {
				cleanPhoto[key] = clampFloat(value, 0, 100)
			}
		}
		if value, exists := photo["zoom"].(float64); exists {
			cleanPhoto["zoom"] = clampFloat(value, 1, 2)
		}
		for _, key := range []string{"layoutX", "layoutY"} {
			if value, exists := photo[key].(float64); exists {
				cleanPhoto[key] = clampFloat(value, 0, 82)
			}
		}
		if value, exists := photo["size"].(float64); exists {
			cleanPhoto["size"] = clampFloat(value, 18, 50)
		}
		data["photo"] = cleanPhoto
	}
	if areas, ok := data["textAreas"].(map[string]any); ok {
		cleanAreas := map[string]any{}
		allowedKeys := map[string]bool{
			"eyebrowLeft": true, "eyebrowRight": true, "heading": true, "body": true,
			"eventDate": true, "eventLocation": true, "footer": true,
		}
		for key, rawArea := range areas {
			if !allowedKeys[key] {
				continue
			}
			area, ok := rawArea.(map[string]any)
			if !ok {
				continue
			}
			cleanArea := map[string]any{}
			if text, exists := area["text"].(string); exists {
				cleanArea["text"] = truncateRunes(strings.TrimSpace(text), 500)
			}
			for _, axis := range []string{"x", "y"} {
				if value, exists := area[axis].(float64); exists {
					cleanArea[axis] = clampFloat(value, 0, 92)
				}
			}
			if value, exists := area["width"].(float64); exists {
				cleanArea["width"] = clampFloat(value, 10, 90)
			}
			if value, exists := area["fontSize"].(float64); exists {
				cleanArea["fontSize"] = clampFloat(value, 8, 64)
			}
			if family, exists := area["fontFamily"].(string); exists {
				if cleaned := sanitizeFont(family); cleaned != "" {
					cleanArea["fontFamily"] = cleaned
				}
			}
			if align, exists := area["align"].(string); exists && (align == "left" || align == "center" || align == "right") {
				cleanArea["align"] = align
			}
			cleanAreas[key] = cleanArea
		}
		data["textAreas"] = cleanAreas
	}
	out, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(out)
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) > max {
		runes = runes[:max]
	}
	return string(runes)
}

func sanitizeExternalURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || strings.ContainsAny(s, "<>\\\"'") {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return s
}

func sanitizeBackgroundURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Reject any character that could break out of a CSS url() context.
	if strings.ContainsAny(s, "()\"'<>\\") {
		return ""
	}
	// Same-origin relative paths only.
	if strings.HasPrefix(s, "/") {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return s
}

// builtInTemplates holds the default set of invite card templates.
var builtInTemplates = []*Template{
	{ID: "balloon-party", Name: "Balloon Party", Description: "Colorful balloons and festive decorations for a fun celebration."},
	{ID: "confetti", Name: "Confetti", Description: "Bright confetti bursts for a joyful and lively event."},
	{ID: "unicorn-magic", Name: "Unicorn Magic", Description: "Whimsical unicorns and rainbow colors for a magical gathering."},
	{ID: "superhero", Name: "Superhero", Description: "Bold superhero theme with dynamic comic-style graphics."},
	{ID: "garden-picnic", Name: "Garden Picnic", Description: "Relaxed garden vibes with floral accents for outdoor events."},
	{ID: "elegant-affair", Name: "Elegant Affair", Description: "Thin border, italic heading, and subtle shadow for a refined look."},
	{ID: "clean-minimal", Name: "Clean Minimal", Description: "No frills, white background, and clean lines for a modern feel."},
	{ID: "tropical-vibes", Name: "Tropical Vibes", Description: "Warm colors and wave decorations for a beachy, tropical event."},
	{ID: "vintage-retro", Name: "Vintage Retro", Description: "Double border, uppercase heading, and sepia tones for a classic vibe."},
	{ID: "chalkboard", Name: "Chalkboard", Description: "Dark background with chalk-style text for a cozy, handwritten feel."},
	{ID: "botanical-save-the-date", Name: "Botanical Save the Date", Description: "An animated envelope reveal with botanical artwork, photo, and QR code."},
}

// Service contains the business logic for invite card management.
type Service struct {
	store      *Store
	uploadsDir string
}

// NewService creates a new invite Service.
func NewService(store *Store, uploadsDir string) *Service {
	return &Service{store: store, uploadsDir: uploadsDir}
}

// ListTemplates returns all available built-in templates.
func (s *Service) ListTemplates() []*Template {
	return builtInTemplates
}

// GetByEventID retrieves the invite card for a given event.
func (s *Service) GetByEventID(ctx context.Context, eventID string) (*InviteCard, error) {
	card, err := s.store.FindByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, fmt.Errorf("invite card not found")
	}
	return card, nil
}

// Save creates or updates the invite card for an event.
func (s *Service) Save(ctx context.Context, eventID string, req SaveInviteRequest) (*InviteCard, error) {
	if req.TemplateID == "" {
		req.TemplateID = "balloon-party"
	}

	// Validate the template ID.
	valid := false
	for _, t := range builtInTemplates {
		if t.ID == req.TemplateID {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid templateId: %s", req.TemplateID)
	}

	card := &InviteCard{
		ID:             uuid.Must(uuid.NewV7()).String(),
		EventID:        eventID,
		TemplateID:     req.TemplateID,
		Heading:        req.Heading,
		Body:           req.Body,
		Footer:         req.Footer,
		PrimaryColor:   sanitizeColor(req.PrimaryColor),
		SecondaryColor: sanitizeColor(req.SecondaryColor),
		Font:           sanitizeFont(req.Font),
		CustomData:     sanitizeCustomData(req.CustomData),
	}

	if card.PrimaryColor == "" {
		card.PrimaryColor = "#6366f1"
	}
	if card.SecondaryColor == "" {
		card.SecondaryColor = "#f0abfc"
	}
	if card.Font == "" {
		card.Font = "Inter"
	}
	if card.CustomData == "" {
		card.CustomData = "{}"
	}

	// Clean up old background image if it changed.
	s.cleanupOldImage(ctx, eventID, card.CustomData)

	if err := s.store.Upsert(ctx, card); err != nil {
		return nil, err
	}

	return card, nil
}

// cleanupOldImage removes the previous background image file from disk when
// the customData.backgroundImage value has changed or been removed.
func (s *Service) cleanupOldImage(ctx context.Context, eventID, newCustomData string) {
	if s.uploadsDir == "" {
		return
	}

	old, err := s.store.FindByEventID(ctx, eventID)
	if err != nil || old == nil {
		return
	}

	oldURL := extractBackgroundImage(old.CustomData)
	newURL := extractBackgroundImage(newCustomData)

	if oldURL != "" && oldURL != newURL {
		// Extract filename from URL path like /api/v1/uploads/filename.jpg
		parts := strings.Split(oldURL, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			_ = os.Remove(filepath.Join(s.uploadsDir, filename))
		}
	}
}

// extractBackgroundImage pulls the backgroundImage value from a customData JSON string.
func extractBackgroundImage(customData string) string {
	if customData == "" || customData == "{}" {
		return ""
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(customData), &data); err != nil {
		return ""
	}
	if bg, ok := data["backgroundImage"].(string); ok {
		return bg
	}
	return ""
}

// GetPreview retrieves the invite card for an event, returning a default card
// if none exists yet.
func (s *Service) GetPreview(ctx context.Context, eventID string) (*InviteCard, error) {
	card, err := s.store.FindByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if card != nil {
		return card, nil
	}

	// Return a default preview card without persisting it.
	return &InviteCard{
		EventID:        eventID,
		TemplateID:     "balloon-party",
		Heading:        "",
		Body:           "",
		Footer:         "",
		PrimaryColor:   "#6366f1",
		SecondaryColor: "#f0abfc",
		Font:           "Inter",
		CustomData:     "{}",
	}, nil
}
