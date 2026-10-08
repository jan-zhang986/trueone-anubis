package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
	os.Exit(m.Run())
}

type envelope struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

func call(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var got envelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not a JSON envelope: %v; raw=%s", err, w.Body.String())
	}
	return got
}

// TC-ROUTE-01
// Contract: the entire surface is registered twice - once at the root and once
// under /api - so that both the legacy web shell and the new console resolve the
// same handler. Any route missing its twin breaks one of the two callers.
func TestRouteInventory_DualPrefixParity(t *testing.T) {
	r := SetupRouter()
	routes := r.Routes()

	byKey := make(map[string]uintptr, len(routes))
	for _, rt := range routes {
		key := rt.Method + " " + rt.Path
		if _, duplicate := byKey[key]; duplicate {
			t.Fatalf("duplicate route registered: %s (gin would normally panic)", key)
		}
		byKey[key] = reflect.ValueOf(rt.Handler).Pointer()
	}

	var rootCount, apiCount int
	for _, rt := range routes {
		if strings.HasPrefix(rt.Path, "/api/") {
			apiCount++
			continue
		}
		rootCount++

		twinKey := rt.Method + " /api" + rt.Path
		twinPtr, ok := byKey[twinKey]
		if !ok {
			t.Errorf("route %s %s has no /api twin (%s)", rt.Method, rt.Path, twinKey)
			continue
		}
		if twinPtr != reflect.ValueOf(rt.Handler).Pointer() {
			t.Errorf("route %s and %s resolve to different handlers", rt.Method+" "+rt.Path, twinKey)
		}
	}

	if rootCount != apiCount {
		t.Errorf("prefix parity broken: %d root routes vs %d /api routes", rootCount, apiCount)
	}
	if len(routes) < 200 {
		t.Errorf("route count = %d; the registered surface is expected to exceed 200", len(routes))
	}
	t.Logf("route inventory: total=%d root=%d api=%d", len(routes), rootCount, apiCount)
}

// TC-ROUTE-02
// Contract: both prefixes must answer identically, not merely be registered.
func TestRouteInventory_BothPrefixesAnswerIdentically(t *testing.T) {
	r := SetupRouter()

	for _, path := range []string{"/case/repository/list", "/project/version/options/p-1"} {
		root := call(r, http.MethodGet, path, "")
		api := call(r, http.MethodGet, "/api"+path, "")

		if root.Code != api.Code {
			t.Errorf("%s: root HTTP %d vs /api HTTP %d", path, root.Code, api.Code)
		}
		if root.Body.String() != api.Body.String() {
			t.Errorf("%s: bodies differ\n root=%s\n  api=%s", path, root.Body.String(), api.Body.String())
		}
	}
}

// TC-ROUTE-03
// Contract: skeleton endpoints must still return the {code,data,message}
// envelope with an empty array (not null) so list rendering never throws.
func TestStubEndpoints_ReturnEnvelopeWithEmptyArray(t *testing.T) {
	r := SetupRouter()

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"case repository list", http.MethodGet, "/case/repository/list", ""},
		{"project version options", http.MethodGet, "/project/version/options/p-1", ""},
		{"organization template list", http.MethodGet, "/organization/template/list/100001/case", ""},
		{"status flow setting", http.MethodGet, "/organization/status/flow/setting/get/100001/case", ""},
		{"organization task center", http.MethodPost, "/organization/task-center/anything", "{}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := call(r, tc.method, tc.path, tc.body)
			if w.Code != http.StatusOK {
				t.Fatalf("HTTP status = %d, want 200", w.Code)
			}
			got := decodeEnvelope(t, w)
			if got.Code != 200 {
				t.Errorf("body.code = %d, want 200 (message=%s)", got.Code, got.Message)
			}
			if string(got.Data) != "[]" {
				t.Errorf("body.data = %s, want [] so the frontend can map over it", got.Data)
			}
		})
	}
}

// TC-ROUTE-04
// Contract: the test-plan paging stub must emit a complete page object, because
// the list page binds total/totalPages directly.
func TestStubEndpoint_TestPlanPageEmitsPageEnvelope(t *testing.T) {
	r := SetupRouter()
	w := call(r, http.MethodPost, "/test-plan/page", "{}")

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200", w.Code)
	}

	var payload struct {
		Code int `json:"code"`
		Data struct {
			List     json.RawMessage `json:"list"`
			Total    int64           `json:"total"`
			Current  int             `json:"current"`
			PageSize int             `json:"pageSize"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if payload.Code != 200 {
		t.Errorf("code = %d, want 200", payload.Code)
	}
	if string(payload.Data.List) != "[]" {
		t.Errorf("data.list = %s, want []", payload.Data.List)
	}
	if payload.Data.Total != 0 {
		t.Errorf("data.total = %d, want 0", payload.Data.Total)
	}
	if payload.Data.Current != 1 || payload.Data.PageSize != 10 {
		t.Errorf("paging = current %d / pageSize %d, want 1/10",
			payload.Data.Current, payload.Data.PageSize)
	}
}

// TC-ROUTE-05
// Contract: the string-returning stub endpoints answer with the literal "ok",
// which the web shell asserts on before showing a success toast.
func TestStubEndpoints_ReturnOkLiteral(t *testing.T) {
	r := SetupRouter()

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/user/role/organization/add"},
		{http.MethodPost, "/user/role/organization/update"},
		{http.MethodGet, "/user/role/organization/delete/r-1"},
		{http.MethodPost, "/organization/status/flow/setting/status/definition/update"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := call(r, tc.method, tc.path, "{}")
			if w.Code != http.StatusOK {
				t.Fatalf("HTTP status = %d, want 200 (body=%s)", w.Code, w.Body.String())
			}
			got := decodeEnvelope(t, w)
			if got.Code != 200 {
				t.Errorf("code = %d, want 200", got.Code)
			}
			if string(got.Data) != `"ok"` {
				t.Errorf("data = %s, want \"ok\"", got.Data)
			}
		})
	}
}

// TC-ROUTE-06
// Contract: unknown paths must produce a 404 rather than falling through to a
// catch-all handler, so a typo in the client is immediately visible.
func TestUnknownRouteReturns404(t *testing.T) {
	r := SetupRouter()

	for _, path := range []string{"/definitely-not-a-route", "/api/definitely-not-a-route"} {
		w := call(r, http.MethodGet, path, "")
		if w.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, w.Code)
		}
	}
}

// TC-ROUTE-07
// Contract: a path that exists under a different verb must not be silently
// accepted. The surface mixes GET and POST for the same resource, so a client
// using the wrong verb must be told.
func TestMethodMismatchIsRejected(t *testing.T) {
	r := SetupRouter()

	// /login is POST-only; a GET must not reach the login handler.
	w := call(r, http.MethodGet, "/login", "")
	if w.Code == http.StatusOK {
		got := decodeEnvelope(t, w)
		if got.Code == 200 {
			t.Errorf("GET /login returned a success envelope; it is registered as POST only")
		}
	}
}

// TC-ROUTE-08
// Contract: the middleware chain is global, and CORS answers preflight before any
// business handler runs, for both prefixes.
func TestPreflightIsHandledGlobally(t *testing.T) {
	r := SetupRouter()

	for _, path := range []string{"/login", "/api/login", "/functional/case/page"} {
		req := httptest.NewRequest(http.MethodOptions, path, nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("OPTIONS %s = %d, want 204", path, w.Code)
		}
	}
}

// TC-ROUTE-09
// Contract: every registered path sits under a known, auditable prefix. A stray
// root-level path is a sign of an accidental registration or a typo'd group.
func TestRouteInventory_PrefixesAreKnown(t *testing.T) {
	r := SetupRouter()

	known := []string{
		"/api/", "/login", "/is-login", "/signout", "/get-key", "/lark", "/project",
		"/user", "/system", "/organization", "/quality-workspace", "/functional",
		"/testcase", "/test-plan", "/bug", "/metrics", "/notification", "/cov",
		"/notice", "/service", "/operation", "/display", "/setting", "/projects",
		"/case", "/organization", "/workflow",
	}

	for _, rt := range r.Routes() {
		matched := false
		for _, prefix := range known {
			if strings.HasPrefix(rt.Path, prefix) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("route %s %s does not match any known prefix", rt.Method, rt.Path)
		}
	}
}
