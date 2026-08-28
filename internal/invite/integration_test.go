package invite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yannkr/openrsvp/internal/auth"
	"github.com/yannkr/openrsvp/internal/event"
	"github.com/yannkr/openrsvp/internal/testutil"
)

// TestIntegration_AllTemplatesRenderableE2E verifies the full lifecycle:
//   - All invite templates are available via the API
//   - Each template can be saved to an invite card
//   - An image can be uploaded and referenced in customData
//   - Saving a new image cleans up the old one on disk
//   - The public preview endpoint returns the correct data
//
// This replaces the manual test plan steps:
//
//	"start dev, log in, open invite designer, test all invite templates,
//	 upload image, verify it shows in preview and public invite page"
func TestIntegration_AllTemplatesRenderableE2E(t *testing.T) {
	db := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	uploadsDir := t.TempDir()

	// Setup auth + event.
	authStore := auth.NewStore(db)
	org, err := authStore.CreateOrganizer(context.Background(), "e2e@example.com")
	require.NoError(t, err)

	eventStore := event.NewStore(db)
	eventSvc := event.NewService(eventStore, cfg.DefaultRetentionDays)
	ev, err := eventSvc.Create(context.Background(), org.ID, event.CreateEventRequest{
		Title:     "E2E Test Event",
		EventDate: "2026-09-15T18:00",
		Location:  "Test Venue",
	})
	require.NoError(t, err)

	inviteStore := NewStore(db)
	svc := NewService(inviteStore, uploadsDir)

	authMW := testutil.FakeAuthMiddleware(func(ctx context.Context) context.Context {
		return auth.ContextWithOrganizer(ctx, org)
	})
	handler := NewHandler(svc, authMW, inviteOrgFromCtx(), uploadsDir, makeCheckEventOwner(eventSvc), zerolog.Nop())
	router := handler.Routes()
	var uploadedPhotoURL, uploadedBackgroundURL string

	// ── Step 1: Verify all templates are listed ──
	t.Run("list_all_templates", func(t *testing.T) {
		rr := testutil.DoRequest(t, router, "GET", "/templates", nil)
		assert.Equal(t, http.StatusOK, rr.Code)

		body := testutil.ParseJSON(t, rr)
		data := body["data"].([]any)
		assert.Len(t, data, 12, "should have exactly 12 templates")

		expectedIDs := []string{
			"balloon-party", "confetti", "unicorn-magic", "superhero", "garden-picnic",
			"elegant-affair", "clean-minimal", "tropical-vibes", "vintage-retro", "chalkboard", "botanical-save-the-date", "custom-card",
		}
		for i, tmpl := range data {
			m := tmpl.(map[string]any)
			assert.Equal(t, expectedIDs[i], m["id"], "template %d should be %s", i, expectedIDs[i])
			assert.NotEmpty(t, m["name"], "template %s should have a name", expectedIDs[i])
			assert.NotEmpty(t, m["description"], "template %s should have a description", expectedIDs[i])
		}
	})

	// ── Step 2: Save invite with each template ──
	allTemplates := []string{
		"balloon-party", "confetti", "unicorn-magic", "superhero", "garden-picnic",
		"elegant-affair", "clean-minimal", "tropical-vibes", "vintage-retro", "chalkboard", "botanical-save-the-date", "custom-card",
	}
	for _, tmplID := range allTemplates {
		t.Run("save_template_"+tmplID, func(t *testing.T) {
			rr := testutil.DoRequest(t, router, "PUT", "/event/"+ev.ID, map[string]string{
				"templateId": tmplID,
				"heading":    "Test " + tmplID,
				"body":       "Body for " + tmplID,
				"footer":     "Footer",
			})
			assert.Equal(t, http.StatusOK, rr.Code)
			body := testutil.ParseJSON(t, rr)
			data := body["data"].(map[string]any)
			assert.Equal(t, tmplID, data["templateId"])
		})
	}

	// ── Step 3: Upload independent photo and card-background slots ──
	t.Run("upload_independent_images", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 10, 10))
		img.Set(5, 5, color.RGBA{R: 255, G: 128, B: 0, A: 255})
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, img))

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("image", "bg.png")
		require.NoError(t, err)
		_, err = part.Write(buf.Bytes())
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		req := httptest.NewRequest(http.MethodPost, "/event/"+ev.ID+"/image?slot=photo", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		resp := testutil.ParseJSON(t, rr)
		data := resp["data"].(map[string]any)
		uploadedPhotoURL = data["url"].(string)
		assert.Contains(t, uploadedPhotoURL, ev.ID+"_photo_")

		// Verify file exists on disk.
		filename := filepath.Base(uploadedPhotoURL)
		_, statErr := os.Stat(filepath.Join(uploadsDir, filename))
		assert.NoError(t, statErr, "uploaded file should exist on disk")

		body.Reset()
		writer = multipart.NewWriter(&body)
		part, err = writer.CreateFormFile("image", "art.png")
		require.NoError(t, err)
		_, err = part.Write(buf.Bytes())
		require.NoError(t, err)
		require.NoError(t, writer.Close())
		req = httptest.NewRequest(http.MethodPost, "/event/"+ev.ID+"/image?slot=background", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		uploadedBackgroundURL = testutil.ParseJSON(t, rr)["data"].(map[string]any)["url"].(string)
		assert.Contains(t, uploadedBackgroundURL, ev.ID+"_background_")
		_, statErr = os.Stat(filepath.Join(uploadsDir, filepath.Base(uploadedPhotoURL)))
		assert.NoError(t, statErr, "uploading artwork must not remove the feature photo")
	})

	// ── Step 4: Save invite with backgroundImage in customData ──
	t.Run("save_with_background_image", func(t *testing.T) {
		customData := fmt.Sprintf(`{"backgroundImage":%q,"cardBackgroundImage":%q,"customCardName":"Summer Garden Party","envelopeFlapColor":"#C47A68","cardBackground":{"fit":"cover","x":40,"y":60,"zoom":1.2}}`, uploadedPhotoURL, uploadedBackgroundURL)
		rr := testutil.DoRequest(t, router, "PUT", "/event/"+ev.ID, map[string]string{
			"templateId": "custom-card",
			"heading":    "Dark Theme Party",
			"body":       "Come to the dark side",
			"customData": customData,
		})
		assert.Equal(t, http.StatusOK, rr.Code)
		body := testutil.ParseJSON(t, rr)
		data := body["data"].(map[string]any)
		assert.Equal(t, "custom-card", data["templateId"])
		assert.JSONEq(t, customData, data["customData"].(string))
	})

	// ── Step 5: Preview returns correct data ──
	t.Run("preview_returns_saved_data", func(t *testing.T) {
		rr := testutil.DoRequest(t, router, "GET", "/event/"+ev.ID+"/preview", nil)
		assert.Equal(t, http.StatusOK, rr.Code)
		body := testutil.ParseJSON(t, rr)
		data := body["data"].(map[string]any)
		assert.Equal(t, "custom-card", data["templateId"])
		assert.Equal(t, "Dark Theme Party", data["heading"])

		// Verify customData is preserved.
		var cd map[string]any
		err := json.Unmarshal([]byte(data["customData"].(string)), &cd)
		require.NoError(t, err)
		assert.Equal(t, uploadedPhotoURL, cd["backgroundImage"])
		assert.Equal(t, uploadedBackgroundURL, cd["cardBackgroundImage"])
		assert.Equal(t, "Summer Garden Party", cd["customCardName"])
		assert.Equal(t, "#C47A68", cd["envelopeFlapColor"])
	})

	// ── Step 6: Removing backgroundImage cleans up ──
	t.Run("save_and_apply_reusable_custom_template", func(t *testing.T) {
		card, err := svc.GetByEventID(context.Background(), ev.ID)
		require.NoError(t, err)
		rr := testutil.DoRequest(t, router, "POST", "/event/"+ev.ID+"/custom-template", map[string]any{
			"name": "Summer Garden Party", "templateId": "custom-card", "heading": card.Heading,
			"body": card.Body, "footer": card.Footer, "primaryColor": card.PrimaryColor,
			"secondaryColor": card.SecondaryColor, "font": card.Font, "customData": card.CustomData,
		})
		require.Equal(t, http.StatusOK, rr.Code)
		templateID := testutil.ParseJSON(t, rr)["data"].(map[string]any)["id"].(string)

		rr = testutil.DoRequest(t, router, "GET", "/custom-templates", nil)
		require.Equal(t, http.StatusOK, rr.Code)
		assert.Len(t, testutil.ParseJSON(t, rr)["data"].([]any), 1)

		target, err := eventSvc.Create(context.Background(), org.ID, event.CreateEventRequest{Title: "Template Target", EventDate: "2026-11-01T12:00"})
		require.NoError(t, err)
		rr = testutil.DoRequest(t, router, "POST", "/event/"+target.ID+"/custom-template/"+templateID+"/apply", nil)
		require.Equal(t, http.StatusOK, rr.Code)
		applied := testutil.ParseJSON(t, rr)["data"].(map[string]any)
		assert.Equal(t, "custom-card", applied["templateId"])
		var appliedData map[string]any
		require.NoError(t, json.Unmarshal([]byte(applied["customData"].(string)), &appliedData))
		for _, key := range []string{"backgroundImage", "cardBackgroundImage"} {
			imageURL := appliedData[key].(string)
			assert.Contains(t, imageURL, target.ID+"_")
			_, err = os.Stat(filepath.Join(uploadsDir, filepath.Base(imageURL)))
			assert.NoError(t, err)
		}
	})

	// ── Step 7: Removing backgroundImage cleans up ──
	t.Run("remove_background_image_cleans_up", func(t *testing.T) {
		// First, create a real file that matches the old customData.
		// Save with empty customData.
		rr := testutil.DoRequest(t, router, "PUT", "/event/"+ev.ID, map[string]string{
			"templateId": "clean-minimal",
			"heading":    "Minimal Party",
			"customData": "{}",
		})
		assert.Equal(t, http.StatusOK, rr.Code)

		// Old file should be deleted.
		for _, oldURL := range []string{uploadedPhotoURL, uploadedBackgroundURL} {
			_, statErr := os.Stat(filepath.Join(uploadsDir, filepath.Base(oldURL)))
			assert.True(t, os.IsNotExist(statErr), "old invite image should be cleaned up")
		}
	})

	// ── Step 8: Verify default preview (no saved card) ──
	t.Run("default_preview_for_new_event", func(t *testing.T) {
		// Create a second event with no invite card.
		ev2, err := eventSvc.Create(context.Background(), org.ID, event.CreateEventRequest{
			Title:     "Second Event",
			EventDate: "2026-10-01T12:00",
		})
		require.NoError(t, err)

		rr := testutil.DoRequest(t, router, "GET", "/event/"+ev2.ID+"/preview", nil)
		assert.Equal(t, http.StatusOK, rr.Code)
		body := testutil.ParseJSON(t, rr)
		data := body["data"].(map[string]any)
		assert.Equal(t, "balloon-party", data["templateId"], "default template should be balloon-party")
		assert.Equal(t, "{}", data["customData"], "default customData should be empty JSON")
	})
}
