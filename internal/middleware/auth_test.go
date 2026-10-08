package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/testsupport"
)

type probeResult struct {
	UserID      string `json:"userId"`
	SessionID   string `json:"sessionId"`
	OrgID       string `json:"orgId"`
	ProjectID   string `json:"projectId"`
	ReachedBody bool   `json:"reachedBody"`
}

// probe runs AuthMiddleware in front of a trivial handler and reports what the
// handler observed on the gin context.
func probe(t *testing.T, headers map[string]string) probeResult {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, probeResult{
			UserID:      c.GetString("userId"),
			SessionID:   c.GetString("sessionId"),
			OrgID:       c.GetString("orgId"),
			ProjectID:   c.GetString("projectId"),
			ReachedBody: true,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var got probeResult
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("probe handler did not run; status=%d body=%s", w.Code, w.Body.String())
	}
	return got
}

// TC-AUTH-01
// Contract risk (P0): AuthMiddleware never aborts. A request with no credentials
// still reaches the business handler, so every endpoint is effectively public and
// authorization depends entirely on each handler checking for itself.
func TestAuthMiddleware_NoCredentialsStillReachesHandler(t *testing.T) {
	got := probe(t, nil)

	if !got.ReachedBody {
		t.Fatal("handler was blocked; middleware is expected to always call c.Next()")
	}
	if got.UserID != "" {
		t.Errorf("userId = %q, want empty when no token is presented", got.UserID)
	}
}

// TC-AUTH-02
// Contract: a token that exists in the session store resolves to its real user id,
// which is what every downstream audit and ownership check relies on.
func TestAuthMiddleware_KnownSessionResolvesToUser(t *testing.T) {
	const sessionID = "session-known-001"
	Store.Set(sessionID, "u-42")
	t.Cleanup(func() { Store.Delete(sessionID) })

	got := probe(t, map[string]string{"X-AUTH-TOKEN": sessionID})

	if got.UserID != "u-42" {
		t.Errorf("userId = %q, want %q", got.UserID, "u-42")
	}
	if got.SessionID != sessionID {
		t.Errorf("sessionId = %q, want %q", got.SessionID, sessionID)
	}
}

// TC-AUTH-03
// Contract risk (P0): an UNKNOWN token is not rejected. It is copied verbatim into
// the userId context key, so any caller can impersonate an arbitrary identity
// simply by inventing a token value.
func TestAuthMiddleware_UnknownTokenIsAcceptedAsUserId(t *testing.T) {
	const forged = "admin"
	got := probe(t, map[string]string{"X-AUTH-TOKEN": forged})

	if got.UserID != forged {
		t.Fatalf("userId = %q, want %q (documents that unknown tokens are trusted verbatim)", got.UserID, forged)
	}
}

// TC-AUTH-03b
// Requirement: an unrecognised credential must not authenticate the caller.
func TestAuthMiddleware_UnknownTokenMustBeRejected(t *testing.T) {
	got := probe(t, map[string]string{"X-AUTH-TOKEN": "forged-token-value"})

	testsupport.Contract(t, got.UserID == "",
		"an unknown token was accepted and published as userId=%q; an unauthenticated caller can impersonate any identity", got.UserID)
}

// TC-AUTH-01b
// Requirement: a request carrying no credentials must be refused before it
// reaches a business handler.
func TestAuthMiddleware_UncredentialedRequestMustBeRejected(t *testing.T) {
	got := probe(t, nil)

	testsupport.Contract(t, !got.ReachedBody,
		"a request with no credentials reached the business handler; the middleware must abort with 401")
}

// TC-AUTH-04
// Contract: X-AUTH-TOKEN wins over Authorization when both are present.
func TestAuthMiddleware_XAuthTokenTakesPrecedenceOverBearer(t *testing.T) {
	Store.Set("session-primary", "u-primary")
	Store.Set("session-secondary", "u-secondary")
	t.Cleanup(func() {
		Store.Delete("session-primary")
		Store.Delete("session-secondary")
	})

	got := probe(t, map[string]string{
		"X-AUTH-TOKEN":  "session-primary",
		"Authorization": "Bearer session-secondary",
	})

	if got.UserID != "u-primary" {
		t.Errorf("userId = %q, want %q (X-AUTH-TOKEN must win)", got.UserID, "u-primary")
	}
}

// TC-AUTH-05
// Contract: a Bearer Authorization header is accepted as a fallback credential.
func TestAuthMiddleware_BearerFallback(t *testing.T) {
	Store.Set("session-bearer", "u-bearer")
	t.Cleanup(func() { Store.Delete("session-bearer") })

	got := probe(t, map[string]string{"Authorization": "Bearer session-bearer"})

	if got.UserID != "u-bearer" {
		t.Errorf("userId = %q, want %q", got.UserID, "u-bearer")
	}
}

// TC-AUTH-06
// Contract: an Authorization header without the Bearer scheme must not be parsed.
func TestAuthMiddleware_NonBearerAuthorizationIsIgnored(t *testing.T) {
	got := probe(t, map[string]string{"Authorization": "Basic YWRtaW46YWRtaW4="})

	if got.UserID != "" {
		t.Errorf("userId = %q, want empty for a non-Bearer scheme", got.UserID)
	}
}

// TC-AUTH-07
// Contract: ORGANIZATION and PROJECT headers carry the tenant context that every
// query is supposed to be scoped by. They are stored unvalidated.
func TestAuthMiddleware_TenantHeadersArePropagated(t *testing.T) {
	got := probe(t, map[string]string{
		"ORGANIZATION": "100001",
		"PROJECT":      "100001100001",
	})

	if got.OrgID != "100001" {
		t.Errorf("orgId = %q, want %q", got.OrgID, "100001")
	}
	if got.ProjectID != "100001100001" {
		t.Errorf("projectId = %q, want %q", got.ProjectID, "100001100001")
	}
}

// TC-AUTH-08
// Contract: absent tenant headers leave the context keys unset rather than
// defaulting, so handlers must supply their own fallback.
func TestAuthMiddleware_AbsentTenantHeadersLeaveContextEmpty(t *testing.T) {
	got := probe(t, nil)

	if got.OrgID != "" || got.ProjectID != "" {
		t.Errorf("orgId=%q projectId=%q, want both empty", got.OrgID, got.ProjectID)
	}
}

// TC-STORE-01
// Contract: session lifecycle is set/get/delete with existence reporting.
func TestSessionStore_Lifecycle(t *testing.T) {
	Store.Set("st-1", "user-1")

	if v, ok := Store.Get("st-1"); !ok || v != "user-1" {
		t.Errorf("Get after Set = (%q, %v), want (user-1, true)", v, ok)
	}

	Store.Delete("st-1")
	if v, ok := Store.Get("st-1"); ok {
		t.Errorf("Get after Delete = (%q, true), want (_, false)", v)
	}
}

// TC-STORE-02
// Contract: the store is guarded by an RWMutex and must survive concurrent use
// without data races (run with -race).
func TestSessionStore_ConcurrentAccessIsRaceFree(t *testing.T) {
	const n = 200

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("concurrent-%d", i)
			Store.Set(id, "u")
			if v, ok := Store.Get(id); !ok || v != "u" {
				t.Errorf("Get(%s) = (%q, %v), want (u, true)", id, v, ok)
			}
			Store.Delete(id)
			if _, ok := Store.Get(id); ok {
				t.Errorf("Get(%s) still present after Delete", id)
			}
		}(i)
	}
	wg.Wait()
}

func corsEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Cors())
	r.POST("/echo", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

// TC-CORS-01
// Contract: browser preflight is answered with 204 and no body so the real request
// can proceed.
func TestCors_PreflightReturns204(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/echo", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	corsEngine().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods missing on preflight")
	}
}

// TC-CORS-02
// Contract risk (P0): the middleware reflects whatever Origin the caller sends
// while also advertising Access-Control-Allow-Credentials: true, and there is no
// allowlist. Any third-party site can therefore issue credentialed cross-origin
// requests and read the authenticated response body.
func TestCors_UntrustedOriginMustNotBeGrantedAccess(t *testing.T) {
	const attacker = "https://evil.example"

	req := httptest.NewRequest(http.MethodPost, "/echo", nil)
	req.Header.Set("Origin", attacker)
	w := httptest.NewRecorder()
	corsEngine().ServeHTTP(w, req)

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	allowCredentials := w.Header().Get("Access-Control-Allow-Credentials")
	vary := w.Header().Get("Vary")

	t.Logf("observed headers: Allow-Origin=%q Allow-Credentials=%q Vary=%q",
		allowOrigin, allowCredentials, vary)

	testsupport.Contract(t, allowOrigin != attacker,
		"Access-Control-Allow-Origin reflects the untrusted origin %q; an allowlist is required", attacker)
	testsupport.Contract(t, !(allowOrigin == attacker && allowCredentials == "true"),
		"credentialed cross-origin access is granted to %q (reflected Allow-Origin + Allow-Credentials: true)", attacker)
	testsupport.Contract(t, vary == "Origin",
		"Vary is %q; it must be \"Origin\" so shared caches key on the request origin", vary)
}

// TC-CORS-03
// Contract: a request without an Origin header falls back to the wildcard.
func TestCors_WildcardWhenOriginAbsent(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/echo", nil)
	w := httptest.NewRecorder()
	corsEngine().ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

// TC-CORS-04
// Contract: the custom auth headers the web shell sends must be allowed, or every
// cross-origin API call fails at preflight.
func TestCors_AllowsCustomAuthHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/echo", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	corsEngine().ServeHTTP(w, req)

	allowed := strings.ToLower(w.Header().Get("Access-Control-Allow-Headers"))
	for _, required := range []string{"x-auth-token", "organization", "project", "authorization", "csrf-token"} {
		if !strings.Contains(allowed, required) {
			t.Errorf("Access-Control-Allow-Headers %q is missing %q", allowed, required)
		}
	}
}
