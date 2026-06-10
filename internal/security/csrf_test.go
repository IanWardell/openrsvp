package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// csrfTokenFromCookies returns the csrf_token cookie value set on the response.
func csrfTokenFromCookies(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == "csrf_token" {
			return c.Value
		}
	}
	return ""
}

func newCSRFHandler() http.Handler {
	return CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

// TestCSRFSessionBoundTokenAccepted verifies that an authenticated request with
// a correctly session-bound token passes validation.
func TestCSRFSessionBoundTokenAccepted(t *testing.T) {
	handler := newCSRFHandler()
	session := "sess-abc"

	// Mint a session-bound token on a GET while authenticated.
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getReq.AddCookie(&http.Cookie{Name: "session", Value: session})
	getRR := httptest.NewRecorder()
	handler.ServeHTTP(getRR, getReq)

	token := csrfTokenFromCookies(t, getRR)
	require.NotEmpty(t, token, "GET with session should mint a csrf_token")
	require.Contains(t, token, ".", "authenticated token should be session-bound (nonce.hmac)")

	// POST with the bound token + matching session — accepted.
	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.AddCookie(&http.Cookie{Name: "session", Value: session})
	postReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	postReq.Header.Set("X-CSRF-Token", token)
	postRR := httptest.NewRecorder()
	handler.ServeHTTP(postRR, postReq)
	assert.Equal(t, http.StatusOK, postRR.Code, "session-bound token should be accepted")
}

// TestCSRFPlainTokenRejectedWhenAuthenticated verifies that a plain (dot-less)
// double-submit token is rejected once a session cookie is present.
func TestCSRFPlainTokenRejectedWhenAuthenticated(t *testing.T) {
	handler := newCSRFHandler()
	plain, err := generateCSRFNonce()
	require.NoError(t, err)

	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.AddCookie(&http.Cookie{Name: "session", Value: "sess-abc"})
	postReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: plain})
	postReq.Header.Set("X-CSRF-Token", plain)
	postRR := httptest.NewRecorder()
	handler.ServeHTTP(postRR, postReq)

	assert.Equal(t, http.StatusForbidden, postRR.Code,
		"plain dot-less token must be rejected for authenticated requests")
}

// TestCSRFWrongSessionRejected verifies that a token bound to one session is
// rejected when presented with a different session.
func TestCSRFWrongSessionRejected(t *testing.T) {
	handler := newCSRFHandler()

	// Build a token bound to session A.
	tokenA, err := buildCSRFToken("session-A")
	require.NoError(t, err)
	require.Contains(t, tokenA, ".")

	// Present it with session B.
	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.AddCookie(&http.Cookie{Name: "session", Value: "session-B"})
	postReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: tokenA})
	postReq.Header.Set("X-CSRF-Token", tokenA)
	postRR := httptest.NewRecorder()
	handler.ServeHTTP(postRR, postReq)

	assert.Equal(t, http.StatusForbidden, postRR.Code,
		"token bound to a different session must be rejected")
}

// TestCSRFRebindIssuesBoundCookieOnGet verifies that a GET from an authenticated
// user holding a stale pre-login plain token gets a fresh session-bound cookie.
func TestCSRFRebindIssuesBoundCookieOnGet(t *testing.T) {
	handler := newCSRFHandler()
	session := "sess-rebind"

	plain, err := generateCSRFNonce()
	require.NoError(t, err)

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getReq.AddCookie(&http.Cookie{Name: "session", Value: session})
	getReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: plain})
	getRR := httptest.NewRecorder()
	handler.ServeHTTP(getRR, getReq)

	newToken := csrfTokenFromCookies(t, getRR)
	require.NotEmpty(t, newToken, "GET should re-mint a bound csrf_token")
	require.NotEqual(t, plain, newToken, "stale plain token should be replaced")
	assert.True(t, isBoundToSession(newToken, session),
		"re-minted token should be bound to the current session")
}

// TestCSRFRebindIssuesBoundCookieOnRejectedMutation verifies that a rejected
// authenticated mutation (stale plain token) still hands the client a fresh
// bound cookie so the next attempt can succeed.
func TestCSRFRebindIssuesBoundCookieOnRejectedMutation(t *testing.T) {
	handler := newCSRFHandler()
	session := "sess-rebind-post"

	plain, err := generateCSRFNonce()
	require.NoError(t, err)

	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.AddCookie(&http.Cookie{Name: "session", Value: session})
	postReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: plain})
	postReq.Header.Set("X-CSRF-Token", plain)
	postRR := httptest.NewRecorder()
	handler.ServeHTTP(postRR, postReq)

	assert.Equal(t, http.StatusForbidden, postRR.Code)
	newToken := csrfTokenFromCookies(t, postRR)
	require.NotEmpty(t, newToken, "rejected auth mutation should set a fresh bound cookie")
	assert.True(t, isBoundToSession(newToken, session),
		"re-minted token should be bound to the current session")
}

// TestCSRFUnauthenticatedPlainDoubleSubmit verifies the pre-login flow: a plain
// double-submit token (no session) is still accepted.
func TestCSRFUnauthenticatedPlainDoubleSubmit(t *testing.T) {
	handler := newCSRFHandler()

	// GET with no session mints a plain token.
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRR := httptest.NewRecorder()
	handler.ServeHTTP(getRR, getReq)

	token := csrfTokenFromCookies(t, getRR)
	require.NotEmpty(t, token)
	require.NotContains(t, token, ".", "unauthenticated token should be a plain nonce")

	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	postReq.Header.Set("X-CSRF-Token", token)
	postRR := httptest.NewRecorder()
	handler.ServeHTTP(postRR, postReq)
	assert.Equal(t, http.StatusOK, postRR.Code,
		"plain double-submit token should work for unauthenticated requests")
}

// TestCSRFAuthenticatedMutationRequiresToken verifies that a PATCH /me-style
// authenticated mutation (no longer excluded) requires a valid CSRF token.
func TestCSRFAuthenticatedMutationRequiresToken(t *testing.T) {
	// /api/v1/auth/ is no longer a blanket exclusion; only the pre-auth
	// endpoints are excluded.
	handler := CSRFMiddleware([]string{"/api/v1/auth/magic-link", "/api/v1/auth/verify"})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess-abc"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code,
		"PATCH /me should require a CSRF token now that the path is not excluded")
}

// TestCSRFLogoutNotExcluded verifies POST /logout requires a CSRF token (it is a
// cookie-auth mutation, so it must be protected).
func TestCSRFLogoutNotExcluded(t *testing.T) {
	handler := CSRFMiddleware([]string{"/api/v1/auth/magic-link", "/api/v1/auth/verify"})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess-abc"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code,
		"POST /logout should require a CSRF token")
}

// TestVerifyCSRFTokenSessionBoundRejectedWithoutSession verifies that a
// session-bound token cannot be replayed on an unauthenticated request.
func TestVerifyCSRFTokenSessionBoundRejectedWithoutSession(t *testing.T) {
	bound, err := buildCSRFToken("some-session")
	require.NoError(t, err)
	require.Contains(t, bound, ".")

	// No session value: a bound token should not be accepted.
	assert.False(t, verifyCSRFToken(bound, bound, ""),
		"session-bound token must be rejected when no session is present")
}
