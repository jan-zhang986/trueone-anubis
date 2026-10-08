package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/router"
)

// testPrefix namespaces every row this suite creates so that cleanup is exact and
// a crashed run can be identified and swept by hand.
const testPrefix = "TESTARCH"

var engine *gin.Engine

func api() *gin.Engine {
	if engine == nil {
		engine = router.SetupRouter()
	}
	return engine
}

// apiResponse is the decoded {code,data,message} envelope plus the HTTP status.
type apiResponse struct {
	HTTP    int
	Code    int
	Message string
	Data    json.RawMessage
}

// request drives one HTTP call through the fully wired router.
func request(t *testing.T, method, path string, body interface{}, headers map[string]string) apiResponse {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	api().ServeHTTP(w, req)

	var parsed apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("%s %s: body is not a JSON envelope: %v; raw=%s", method, path, err, w.Body.String())
	}
	parsed.HTTP = w.Code
	return parsed
}

// authHeader returns the credential headers the web shell sends.
func authHeader() map[string]string {
	return map[string]string{
		"X-AUTH-TOKEN": adminUserID,
		"ORGANIZATION": defaultOrgID,
		"PROJECT":      defaultProjectID,
	}
}

// decode unmarshals an envelope's data payload into target.
func decode(t *testing.T, resp apiResponse, target interface{}) {
	t.Helper()
	if len(resp.Data) == 0 {
		t.Fatalf("response has no data payload: code=%d message=%s", resp.Code, resp.Message)
	}
	if err := json.Unmarshal(resp.Data, target); err != nil {
		t.Fatalf("cannot decode data payload: %v; raw=%s", err, resp.Data)
	}
}

func newTestID(kind string) string {
	return fmt.Sprintf("%s-%s-%d", testPrefix, kind, time.Now().UnixNano())
}

// newTestWorkspace inserts an isolated quality workspace and registers cleanup.
//
// The JSON-typed columns are seeded with valid JSON literals: GORM sends the Go
// zero value ("") for absent string fields, and MySQL rejects an empty string for
// a json column. That behaviour is itself asserted by
// TestQualityWorkspace_Save_WithoutJSONColumns.
func newTestWorkspace(t *testing.T, label string) string {
	t.Helper()
	requireDB(t)

	id := newTestID("ws")
	now := time.Now().UnixMilli()

	ws := model.QualityWorkspace{
		WorkspaceID:     id,
		ProjectID:       defaultProjectID,
		Name:            testPrefix + " " + label,
		Status:          "DRAFT",
		Tags:            "[]",
		ScopeDefinition: "{}",
		Metadata:        "{}",
		CreateUser:      adminUserID,
		UpdateUser:      adminUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Archived:        model.BitBool(false),
	}
	if err := config.DB.Create(&ws).Error; err != nil {
		t.Fatalf("cannot create fixture workspace: %v", err)
	}

	t.Cleanup(func() { removeWorkspace(t, id) })
	return id
}

func removeWorkspace(t *testing.T, workspaceID string) {
	t.Helper()
	for _, statement := range []string{
		"DELETE FROM quality_work_item WHERE workspace_id = ?",
		"DELETE FROM quality_task WHERE workspace_id = ?",
		"DELETE FROM quality_report WHERE workspace_id = ?",
		"DELETE FROM quality_workspace WHERE workspace_id = ?",
	} {
		if err := config.DB.Exec(statement, workspaceID).Error; err != nil {
			t.Errorf("cleanup %q for %s: %v", statement, workspaceID, err)
		}
	}
}

// newTestWorkItem inserts one execution item into a workspace.
func newTestWorkItem(t *testing.T, workspaceID, taskID, status string) string {
	t.Helper()

	id := newTestID("wi")
	now := time.Now().UnixMilli()

	item := model.QualityWorkItem{
		WorkItemID:      id,
		WorkspaceID:     workspaceID,
		TaskID:          taskID,
		ProjectID:       defaultProjectID,
		Title:           testPrefix + " item",
		Status:          status,
		Result:          status,
		RuntimeSnapshot: "{}",
		Metadata:        "{}",
		CreateUser:      adminUserID,
		UpdateUser:      adminUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := config.DB.Create(&item).Error; err != nil {
		t.Fatalf("cannot create fixture work item: %v", err)
	}
	return id
}

// newTestTask inserts one quality task into a workspace.
//
// Metadata is seeded with a JSON literal because MySQL rejects an empty string for
// a json column while GORM sends the Go zero value when the field is unset. The
// API-level consequence of that mismatch is asserted by
// TestQualityWorkspace_Save_MinimalPayloadCreatesRow.
func newTestTask(t *testing.T, workspaceID, status string) string {
	t.Helper()

	id := newTestID("task")
	now := time.Now().UnixMilli()

	task := model.QualityTask{
		TaskID:      id,
		WorkspaceID: workspaceID,
		ProjectID:   defaultProjectID,
		TaskType:    "FUNCTIONAL",
		Title:       testPrefix + " task",
		Status:      status,
		Metadata:    "{}",
		CreateUser:  adminUserID,
		UpdateUser:  adminUserID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := config.DB.Create(&task).Error; err != nil {
		t.Fatalf("cannot create fixture task: %v", err)
	}
	return id
}

// newTestUserInOrg inserts a disposable user belonging to a specific organisation,
// so that tenant scoping can be asserted without touching the admin account.
func newTestUserInOrg(t *testing.T, organizationID string) string {
	t.Helper()
	requireDB(t)

	id := newTestID("user")
	now := time.Now().UnixMilli()

	user := model.User{
		ID:                 id,
		Name:               "Test Arch Fixture",
		Email:              id + "@example.com",
		Password:           "fixture-password",
		Enable:             model.BitBool(true),
		Deleted:            model.BitBool(false),
		CreatedAt:          now,
		UpdatedAt:          now,
		Source:             "LOCAL",
		LastOrganizationID: organizationID,
		LastProjectID:      defaultProjectID,
		CreateUser:         adminUserID,
		UpdateUser:         adminUserID,
	}
	if err := config.DB.Create(&user).Error; err != nil {
		t.Fatalf("cannot create fixture user: %v", err)
	}

	t.Cleanup(func() {
		if err := config.DB.Exec("DELETE FROM user WHERE id = ?", id).Error; err != nil {
			t.Errorf("cleanup fixture user %s: %v", id, err)
		}
	})
	return id
}

func newTestUser(t *testing.T) string {
	t.Helper()
	return newTestUserInOrg(t, defaultOrgID)
}

// countAuditLogs counts operation_log rows matching a predicate.
func countAuditLogs(t *testing.T, where string, args ...interface{}) int64 {
	t.Helper()

	var n int64
	if err := config.DB.Model(&model.OperationLog{}).Where(where, args...).Count(&n).Error; err != nil {
		t.Fatalf("count operation_log: %v", err)
	}
	return n
}

// waitForAuditLog polls for an audit row, because RecordAuditLog writes from a
// detached goroutine and is therefore not observable the moment the HTTP call
// returns.
func waitForAuditLog(t *testing.T, where string, args ...interface{}) bool {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if countAuditLogs(t, where, args...) > 0 {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
