package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"trueone-anubis/internal/middleware"
	"trueone-anubis/internal/testsupport"
)

type sessionPayload struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Email              string   `json:"email"`
	LastOrganizationID string   `json:"lastOrganizationId"`
	LastProjectID      string   `json:"lastProjectId"`
	SessionID          string   `json:"sessionId"`
	CsrfToken          string   `json:"csrfToken"`
	Permissions        []string `json:"permissions"`
}

// TC-IT-AUTH-01
// Contract: a valid login returns the "true" sentinel that the web shell treats as
// success, issues a session registered in the server-side store, and appends
// exactly one LOGIN row to operation_log.
func TestLogin_SuccessIssuesSessionAndWritesAuditLog(t *testing.T) {
	requireDB(t)

	before := countAuditLogs(t, "type = ? AND path = ?", "LOGIN", "/login")

	resp := request(t, http.MethodPost, "/login", map[string]string{
		"username": adminUserID,
		"password": adminPassword,
	}, nil)

	if resp.HTTP != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200", resp.HTTP)
	}
	if resp.Code != 200 {
		t.Fatalf("body.code = %d, want 200 (message=%s)", resp.Code, resp.Message)
	}
	if resp.Message != "true" {
		t.Errorf("body.message = %q, want the %q success sentinel", resp.Message, "true")
	}

	var session sessionPayload
	decode(t, resp, &session)

	if session.ID != adminUserID {
		t.Errorf("session.id = %q, want %q", session.ID, adminUserID)
	}
	if session.SessionID == "" {
		t.Error("session.sessionId is empty; the client cannot authenticate subsequent calls")
	}
	if session.CsrfToken == "" {
		t.Error("session.csrfToken is empty")
	}
	if session.LastOrganizationID != defaultOrgID {
		t.Errorf("session.lastOrganizationId = %q, want %q", session.LastOrganizationID, defaultOrgID)
	}
	if !containsString(session.Permissions, "*") {
		t.Errorf("admin permissions = %v, want them to include the wildcard", session.Permissions)
	}

	// Evidence: the issued session must be resolvable server-side.
	if uid, ok := middleware.Store.Get(session.SessionID); !ok || uid != adminUserID {
		t.Errorf("session store lookup for %q = (%q, %v), want (%q, true)",
			session.SessionID, uid, ok, adminUserID)
	}

	after := countAuditLogs(t, "type = ? AND path = ?", "LOGIN", "/login")
	if after != before+1 {
		t.Errorf("operation_log LOGIN rows: before=%d after=%d, want exactly one new row", before, after)
	}
}

// TC-IT-AUTH-02
// Contract: a wrong password is rejected with code 400 and produces no session and
// no audit row.
func TestLogin_WrongPasswordIsRejected(t *testing.T) {
	requireDB(t)

	before := countAuditLogs(t, "type = ? AND path = ?", "LOGIN", "/login")

	resp := request(t, http.MethodPost, "/login", map[string]string{
		"username": adminUserID,
		"password": "definitely-not-the-password",
	}, nil)

	if resp.Code != 400 {
		t.Errorf("body.code = %d, want 400", resp.Code)
	}
	if resp.Message != "用户名或密码错误" {
		t.Errorf("body.message = %q, want the generic credential error", resp.Message)
	}
	if after := countAuditLogs(t, "type = ? AND path = ?", "LOGIN", "/login"); after != before {
		t.Errorf("a failed login wrote %d audit rows, want 0", after-before)
	}
}

// TC-IT-AUTH-03
// Contract: an unknown account and a wrong password must be indistinguishable, so
// the endpoint cannot be used to enumerate valid user ids.
func TestLogin_UnknownUserIsIndistinguishableFromWrongPassword(t *testing.T) {
	requireDB(t)

	unknown := request(t, http.MethodPost, "/login", map[string]string{
		"username": "no-such-user-" + newTestID("x"),
		"password": "whatever",
	}, nil)
	wrongPassword := request(t, http.MethodPost, "/login", map[string]string{
		"username": adminUserID,
		"password": "wrong",
	}, nil)

	if unknown.Code != wrongPassword.Code {
		t.Errorf("unknown user code = %d but wrong password code = %d; the difference leaks account existence",
			unknown.Code, wrongPassword.Code)
	}
	if unknown.Message != wrongPassword.Message {
		t.Errorf("unknown user message = %q but wrong password message = %q; messages must match",
			unknown.Message, wrongPassword.Message)
	}
}

// TC-IT-AUTH-04
// Contract: the login identifier accepts either the id or the email address.
func TestLogin_AcceptsEmailAddress(t *testing.T) {
	requireDB(t)

	resp := request(t, http.MethodPost, "/login", map[string]string{
		"username": "admin@metersphere.io",
		"password": adminPassword,
	}, nil)

	if resp.Code != 200 {
		t.Fatalf("login by email failed: code=%d message=%s", resp.Code, resp.Message)
	}

	var session sessionPayload
	decode(t, resp, &session)
	if session.ID != adminUserID {
		t.Errorf("session.id = %q, want %q when authenticating by email", session.ID, adminUserID)
	}
}

// TC-IT-AUTH-05
// Contract: a login request missing a required field is rejected by the binding
// layer rather than reaching the service with an empty credential.
func TestLogin_MissingRequiredFieldIsRejected(t *testing.T) {
	requireDB(t)

	resp := request(t, http.MethodPost, "/login", map[string]string{"username": adminUserID}, nil)

	if resp.Code != 400 {
		t.Errorf("body.code = %d, want 400 for a missing password", resp.Code)
	}
}

// TC-IT-AUTH-06
// Contract risk (P0): /is-login falls back to the admin account when no credential
// is presented, so an anonymous caller receives a fully populated admin session
// including the wildcard permission.
func TestIsLogin_WithoutCredentialsMustNotReturnAdminSession(t *testing.T) {
	requireDB(t)

	resp := request(t, http.MethodGet, "/is-login", nil, nil)

	var session sessionPayload
	if resp.Code == 200 {
		decode(t, resp, &session)
	}

	t.Logf("observed: HTTP=%d code=%d id=%q permissions=%v", resp.HTTP, resp.Code, session.ID, session.Permissions)

	testsupport.Contract(t, resp.HTTP == http.StatusUnauthorized,
		"/is-login returned HTTP %d for an unauthenticated caller; it must return 401", resp.HTTP)
	testsupport.Contract(t, session.ID != adminUserID,
		"/is-login handed an anonymous caller the %q session (permissions=%v)", session.ID, session.Permissions)
}

// TC-IT-AUTH-07
// Contract: /is-login resolves a valid session token into the owning user.
func TestIsLogin_WithValidTokenReturnsOwningUser(t *testing.T) {
	requireDB(t)

	login := request(t, http.MethodPost, "/login", map[string]string{
		"username": adminUserID,
		"password": adminPassword,
	}, nil)

	var issued sessionPayload
	decode(t, login, &issued)
	if issued.SessionID == "" {
		t.Fatal("login did not issue a session id")
	}

	resp := request(t, http.MethodGet, "/is-login", nil, map[string]string{
		"X-AUTH-TOKEN": issued.SessionID,
	})

	if resp.Code != 200 {
		t.Fatalf("/is-login with a valid token: code=%d message=%s", resp.Code, resp.Message)
	}

	var session sessionPayload
	decode(t, resp, &session)
	if session.ID != adminUserID {
		t.Errorf("session.id = %q, want %q", session.ID, adminUserID)
	}
}

// TC-IT-AUTH-08
// Contract: signout removes the session from the store so the token can no longer
// authenticate, and records a LOGOUT audit row.
func TestSignout_InvalidatesSession(t *testing.T) {
	requireDB(t)

	login := request(t, http.MethodPost, "/login", map[string]string{
		"username": adminUserID,
		"password": adminPassword,
	}, nil)

	var issued sessionPayload
	decode(t, login, &issued)

	resp := request(t, http.MethodGet, "/signout", nil, map[string]string{
		"X-AUTH-TOKEN": issued.SessionID,
	})
	if resp.Code != 200 {
		t.Fatalf("signout: code=%d message=%s", resp.Code, resp.Message)
	}

	if _, ok := middleware.Store.Get(issued.SessionID); ok {
		t.Errorf("session %q is still resolvable after signout", issued.SessionID)
	}

	if !waitForAuditLog(t, "type = ? AND path = ? AND create_user = ?", "LOGOUT", "/signout", adminUserID) {
		t.Error("signout did not append a LOGOUT row to operation_log within 2s")
	}
}

// TC-IT-AUTH-09
// Contract: the mock key endpoint returns the shape the web shell expects, and the
// payload is a placeholder rather than a usable key.
func TestGetKey_ReturnsPlaceholder(t *testing.T) {
	resp := request(t, http.MethodGet, "/get-key", nil, nil)

	if resp.Code != 200 {
		t.Fatalf("get-key: code=%d", resp.Code)
	}

	var key string
	decode(t, resp, &key)
	if key == "" {
		t.Error("get-key returned an empty payload")
	}

	var raw json.RawMessage = resp.Data
	if len(raw) == 0 {
		t.Error("get-key returned no data")
	}
}
