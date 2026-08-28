package invite

import "testing"

func TestSanitizeColor(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty stays empty", "", ""},
		{"3-digit hex", "#abc", "#abc"},
		{"6-digit hex", "#A1B2C3", "#A1B2C3"},
		{"8-digit hex (alpha)", "#11223344", "#11223344"},
		{"trim whitespace", "  #abc  ", "#abc"},
		{"css breakout via semicolon", "red; --x: url(http://evil)", ""},
		{"css breakout via brace", "red}body{color:red}", ""},
		{"named color rejected", "red", ""},
		{"rgb() rejected", "rgb(255,0,0)", ""},
		{"empty after trim", "   ", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeColor(tc.in)
			if got != tc.want {
				t.Fatalf("sanitizeColor(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSanitizeFont(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"benign Inter", "Inter", "Inter"},
		{"font stack", "'Helvetica Neue', Arial, sans-serif", "'Helvetica Neue', Arial, sans-serif"},
		{"trim whitespace", "  Inter  ", "Inter"},
		{"reject curly brace", "Inter{evil:1}", ""},
		{"reject semicolon", "Inter; color: red", ""},
		{"reject @import", "@import url(http://evil)", ""},
		{"reject very long", string(make([]byte, 200)), ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeFont(tc.in)
			if got != tc.want {
				t.Fatalf("sanitizeFont(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSanitizeBackgroundURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"relative path", "/api/v1/uploads/abc.png", "/api/v1/uploads/abc.png"},
		{"https URL", "https://example.com/img.png", "https://example.com/img.png"},
		{"http URL", "http://example.com/img.png", "http://example.com/img.png"},
		{"javascript scheme blocked", "javascript:alert(1)", ""},
		{"data URI blocked", "data:image/svg+xml,<svg onload=alert(1)/>", ""},
		{"url() breakout via paren", "https://example.com/a.png)x{x:url(http://evil)", ""},
		{"backslash breakout", "https://example.com/a\\.png", ""},
		{"quote breakout", "https://example.com/a\".png", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeBackgroundURL(tc.in)
			if got != tc.want {
				t.Fatalf("sanitizeBackgroundURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSanitizeCustomData_DropsHostileBackgroundImage(t *testing.T) {
	in := `{"backgroundImage":"javascript:alert(1)","keep":"value"}`
	out := sanitizeCustomData(in)
	if out == in {
		t.Fatalf("expected hostile backgroundImage to be dropped, got %q", out)
	}
	// "keep" must survive.
	if !contains(out, `"keep":"value"`) {
		t.Fatalf("expected other keys preserved, got %q", out)
	}
	// "backgroundImage" must NOT appear.
	if contains(out, "backgroundImage") {
		t.Fatalf("hostile backgroundImage leaked: %q", out)
	}
}

func TestSanitizeCustomData_InvalidJSONReturnsEmptyObject(t *testing.T) {
	out := sanitizeCustomData(`{ not json`)
	if out != "{}" {
		t.Fatalf("expected {} for malformed JSON, got %q", out)
	}
}

func TestSanitizeCustomData_ClampsLayoutAndPhotoSettings(t *testing.T) {
	in := `{"photo":{"fit":"cover","x":-20,"y":140,"zoom":9,"layoutX":-4,"layoutY":96,"size":70},"qr":{"enabled":true,"url":"https://example.com/invite","caption":" Scan me ","foreground":"#112233","background":"#ffffff","x":-8,"y":99,"size":40,"unknown":"drop me too"},"textAreas":{"heading":{"text":"  Garden  ","x":-5,"y":120,"width":2,"fontSize":99,"fontFamily":"Georgia","align":"center"},"unknown":{"text":"drop me"}}}`
	out := sanitizeCustomData(in)
	for _, want := range []string{`"x":0`, `"y":100`, `"zoom":2`, `"layoutX":0`, `"layoutY":82`, `"size":50`, `"enabled":true`, `"url":"https://example.com/invite"`, `"caption":"Scan me"`, `"foreground":"#112233"`, `"y":90`, `"size":28`, `"text":"Garden"`, `"width":10`, `"fontSize":64`, `"fontFamily":"Georgia"`, `"align":"center"`} {
		if !contains(out, want) {
			t.Fatalf("expected sanitized custom data to contain %q, got %q", want, out)
		}
	}
	if contains(out, "drop me") || contains(out, "drop me too") {
		t.Fatalf("unexpected unknown text area in sanitized custom data: %q", out)
	}
}

func TestSanitizeCustomData_CustomCardNameAndEnvelopeFlapColor(t *testing.T) {
	in := `{"customCardName":"  Summer Garden Party  ","envelopeFlapColor":"#C47A68","cardBackgroundImage":"/api/v1/uploads/event_background_1.png","cardBackground":{"fit":"cover","x":-5,"y":120,"zoom":3}}`
	out := sanitizeCustomData(in)
	for _, want := range []string{`"customCardName":"Summer Garden Party"`, `"envelopeFlapColor":"#C47A68"`, `"cardBackgroundImage":"/api/v1/uploads/event_background_1.png"`, `"x":0`, `"y":100`, `"zoom":2`} {
		if !contains(out, want) {
			t.Fatalf("expected sanitized custom data to contain %q, got %q", want, out)
		}
	}

	invalid := sanitizeCustomData(`{"customCardName":"   ","envelopeFlapColor":"red; background:url(evil)"}`)
	if contains(invalid, "customCardName") || contains(invalid, "envelopeFlapColor") {
		t.Fatalf("expected invalid custom-card fields to be dropped, got %q", invalid)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || indexOf(s, substr) >= 0)
}
func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
