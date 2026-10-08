package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/response"
)

// envelope mirrors the wire contract every aegis-next-web caller depends on.
// Data is kept raw so tests can distinguish JSON null from [] and from {}.
type envelope struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

type pageEnvelope struct {
	Code int `json:"code"`
	Data struct {
		List       json.RawMessage `json:"list"`
		Total      int64           `json:"total"`
		Current    int             `json:"current"`
		PageSize   int             `json:"pageSize"`
		TotalPages int64           `json:"totalPages"`
	} `json:"data"`
	Message string `json:"message"`
}

// run drives a single gin handler against a fresh recorder, with an empty
// request so that handlers never touch the database.
func run(h gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/probe", nil)
	h(c)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var got envelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("response body is not valid JSON: %v; raw=%s", err, w.Body.String())
	}
	return got
}

// TC-RESP-01
// Contract: a success envelope is HTTP 200 + business code 200 + message "success".
func TestSuccess_EnvelopeContract(t *testing.T) {
	w := run(func(c *gin.Context) { response.Success(c, gin.H{"id": "100001"}) })

	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %d, want 200", w.Code)
	}
	got := decode(t, w)
	if got.Code != 200 {
		t.Errorf("body.code = %d, want 200", got.Code)
	}
	if got.Message != "success" {
		t.Errorf("body.message = %q, want %q", got.Message, "success")
	}
	if string(got.Data) != `{"id":"100001"}` {
		t.Errorf("body.data = %s, want {\"id\":\"100001\"}", got.Data)
	}
}

// TC-RESP-02
// Contract: Login responds with message "true", which the web shell treats as the
// success sentinel, so SuccessWithMsg must not rewrite it.
func TestSuccessWithMsg_PreservesCustomSentinel(t *testing.T) {
	w := run(func(c *gin.Context) { response.SuccessWithMsg(c, nil, "true") })

	got := decode(t, w)
	if got.Code != 200 {
		t.Errorf("body.code = %d, want 200", got.Code)
	}
	if got.Message != "true" {
		t.Errorf("body.message = %q, want %q (login sentinel)", got.Message, "true")
	}
}

// TC-RESP-03
// Contract: business failures are reported with HTTP 200 and a non-200 body code.
// The frontend reads body.code exclusively; any change to the HTTP status would
// silently bypass its error interceptor.
func TestFail_ReturnsHTTP200WithBusinessCode(t *testing.T) {
	w := run(func(c *gin.Context) { response.Fail(c, 400, "参数错误") })

	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %d, want 200 (errors travel in the body)", w.Code)
	}
	got := decode(t, w)
	if got.Code != 400 {
		t.Errorf("body.code = %d, want 400", got.Code)
	}
	if got.Message != "参数错误" {
		t.Errorf("body.message = %q, want %q", got.Message, "参数错误")
	}
	if string(got.Data) != "null" {
		t.Errorf("body.data = %s, want null", got.Data)
	}
}

// TC-RESP-04
// Contract: the only path that may use a real 401 status. The web shell keys its
// re-login redirect off HTTP 401 + code 100401.
func TestUnauthorized_ReturnsHTTP401AndCode100401(t *testing.T) {
	w := run(func(c *gin.Context) { response.Unauthorized(c, "未登录") })

	if w.Code != http.StatusUnauthorized {
		t.Errorf("HTTP status = %d, want 401", w.Code)
	}
	got := decode(t, w)
	if got.Code != 100401 {
		t.Errorf("body.code = %d, want 100401", got.Code)
	}
}

// TC-RESP-05
// Contract: totalPages = ceil(total/pageSize). Boundaries matter because the
// frontend paginator renders exactly totalPages pages.
func TestSuccessPage_TotalPagesMath(t *testing.T) {
	cases := []struct {
		name      string
		total     int64
		current   int
		pageSize  int
		wantPages int64
	}{
		{"empty result set", 0, 1, 10, 0},
		{"single item", 1, 1, 10, 1},
		{"exact single page", 10, 1, 10, 1},
		{"one past the boundary", 11, 1, 10, 2},
		{"uneven divisor", 100, 1, 7, 15},
		{"pageSize zero must not divide by zero", 5, 1, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := run(func(c *gin.Context) {
				response.SuccessPage(c, []interface{}{}, tc.total, tc.current, tc.pageSize)
			})

			var got pageEnvelope
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if got.Data.Total != tc.total {
				t.Errorf("total = %d, want %d", got.Data.Total, tc.total)
			}
			if got.Data.TotalPages != tc.wantPages {
				t.Errorf("totalPages = %d, want %d (total=%d pageSize=%d)",
					got.Data.TotalPages, tc.wantPages, tc.total, tc.pageSize)
			}
		})
	}
}

// TC-RESP-06
// Contract risk: a nil slice in the list field marshals to JSON null, not [].
// Callers doing `resp.data.list.map(...)` throw on null. This test pins the
// current behaviour so the defect is visible rather than accidental.
func TestSuccessPage_NilListMarshalsToNull(t *testing.T) {
	w := run(func(c *gin.Context) { response.SuccessPage(c, nil, 0, 1, 10) })

	var got pageEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if string(got.Data.List) != "null" {
		t.Fatalf("list = %s, want null for a nil slice (documents the frontend crash risk)", got.Data.List)
	}
}

// TC-RESP-07
// Contract: SuccessPage's optional fields use omitempty, so a pageSize of 0 drops
// current/pageSize/totalPages from the payload entirely.
func TestSuccessPage_ZeroPageSizeOmitsOptionalKeys(t *testing.T) {
	w := run(func(c *gin.Context) { response.SuccessPage(c, []interface{}{}, 0, 0, 0) })

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw["data"], &data); err != nil {
		t.Fatalf("invalid data object: %v", err)
	}

	for _, key := range []string{"current", "pageSize", "totalPages"} {
		if _, present := data[key]; present {
			t.Errorf("data.%s present with zero value; omitempty should drop it", key)
		}
	}
	if _, present := data["list"]; !present {
		t.Error("data.list must always be present, even when empty")
	}
	if _, present := data["total"]; !present {
		t.Error("data.total must always be present")
	}
}
