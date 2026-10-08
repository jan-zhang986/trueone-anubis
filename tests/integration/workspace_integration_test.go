package integration

import (
	"net/http"
	"testing"

	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/testsupport"
)

type workspaceStats struct {
	WorkspaceID       string  `json:"workspaceId"`
	Total             int     `json:"total"`
	Todo              int     `json:"todo"`
	InProgress        int     `json:"inProgress"`
	Passed            int     `json:"passed"`
	Failed            int     `json:"failed"`
	Blocked           int     `json:"blocked"`
	Skipped           int     `json:"skipped"`
	PassRate          float64 `json:"passRate"`
	ExecutionRate     float64 `json:"executionRate"`
	AllDone           bool    `json:"allDone"`
	ReleaseConclusion string  `json:"releaseConclusion"`
}

// TC-IT-WS-01
// Contract: a workspace can be created with the minimal payload the creation form
// submits. The form does not send the JSON-typed columns, so the persistence layer
// must not turn their absence into invalid JSON.
func TestQualityWorkspace_Save_MinimalPayloadCreatesRow(t *testing.T) {
	requireDB(t)

	name := newTestID("ws-create")

	resp := request(t, http.MethodPost, "/api/quality-workspace/save", map[string]interface{}{
		"projectId": defaultProjectID,
		"name":      name,
		"goal":      "验证最小创建载荷",
	}, authHeader())

	var created struct {
		WorkspaceID string `json:"workspaceId"`
		Status      string `json:"status"`
	}
	createdID := ""
	if resp.Code == 200 {
		decode(t, resp, &created)
		createdID = created.WorkspaceID
		if createdID != "" {
			t.Cleanup(func() { removeWorkspace(t, createdID) })
		}
	}

	t.Logf("observed: code=%d message=%s workspaceId=%q status=%q",
		resp.Code, resp.Message, created.WorkspaceID, created.Status)

	if createdID != "" {
		var persisted int64
		config.DB.Model(&model.QualityWorkspace{}).
			Where("workspace_id = ?", createdID).Count(&persisted)
		if persisted != 1 {
			t.Errorf("workspace %s was reported created but %d rows exist", createdID, persisted)
		}
	}

	testsupport.Contract(t, resp.Code == 200,
		"creating a workspace with the minimal form payload failed: code=%d message=%s", resp.Code, resp.Message)
}

// TC-IT-WS-02
// Contract: a workspace whose every execution item FAILED must never be reported as
// release-ready. This is the platform's core quality gate.
func TestGetStats_FailedSuiteMustNotReportReleaseReady(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "stats-failed")
	taskID := newTestID("task")
	for i := 0; i < 3; i++ {
		newTestWorkItem(t, wsID, taskID, "FAILED")
	}

	resp := request(t, http.MethodGet, "/api/quality-workspace/"+wsID+"/stats", nil, authHeader())
	if resp.Code != 200 {
		t.Fatalf("stats: code=%d message=%s", resp.Code, resp.Message)
	}

	var stats workspaceStats
	decode(t, resp, &stats)

	t.Logf("observed: total=%d passed=%d failed=%d blocked=%d skipped=%d todo=%d allDone=%v passRate=%.2f executionRate=%.2f releaseConclusion=%q",
		stats.Total, stats.Passed, stats.Failed, stats.Blocked, stats.Skipped, stats.Todo,
		stats.AllDone, stats.PassRate, stats.ExecutionRate, stats.ReleaseConclusion)

	if stats.Total != 3 {
		t.Errorf("total = %d, want 3", stats.Total)
	}
	if stats.Failed != 3 {
		t.Errorf("failed = %d, want 3", stats.Failed)
	}
	if stats.Passed != 0 {
		t.Errorf("passed = %d, want 0", stats.Passed)
	}
	if stats.PassRate != 0 {
		t.Errorf("passRate = %.2f, want 0 for an all-failed suite", stats.PassRate)
	}
	if stats.ExecutionRate != 100 {
		t.Errorf("executionRate = %.2f, want 100 when every item has been executed", stats.ExecutionRate)
	}

	testsupport.Contract(t, stats.ReleaseConclusion != "READY",
		"a suite with %d/%d FAILED items still reports releaseConclusion=%q; the release gate is hardcoded and ignores results",
		stats.Failed, stats.Total, stats.ReleaseConclusion)
	testsupport.Contract(t, !stats.AllDone,
		"allDone=%v while %d items are FAILED; completeness must not be conflated with success",
		stats.AllDone, stats.Failed)
}

// TC-IT-WS-03
// Contract: every execution status maps to exactly one counter, and the two rates
// are derived from those counters. This pins the metric arithmetic that the quality
// dashboard renders.
func TestGetStats_CountersAndRatesAreDerivedCorrectly(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "stats-mixed")
	taskID := newTestID("task")

	statuses := []string{"PASSED", "FAILED", "BLOCKED", "SKIPPED", "TODO", "RUNNING"}
	for _, status := range statuses {
		newTestWorkItem(t, wsID, taskID, status)
	}

	resp := request(t, http.MethodGet, "/api/quality-workspace/"+wsID+"/stats", nil, authHeader())
	if resp.Code != 200 {
		t.Fatalf("stats: code=%d message=%s", resp.Code, resp.Message)
	}

	var stats workspaceStats
	decode(t, resp, &stats)

	t.Logf("observed: total=%d passed=%d failed=%d blocked=%d skipped=%d todo=%d inProgress=%d passRate=%.2f executionRate=%.2f",
		stats.Total, stats.Passed, stats.Failed, stats.Blocked, stats.Skipped,
		stats.Todo, stats.InProgress, stats.PassRate, stats.ExecutionRate)

	if stats.Total != 6 {
		t.Errorf("total = %d, want 6", stats.Total)
	}
	if stats.Passed != 1 {
		t.Errorf("passed = %d, want 1", stats.Passed)
	}
	if stats.Failed != 1 {
		t.Errorf("failed = %d, want 1", stats.Failed)
	}
	if stats.Blocked != 1 {
		t.Errorf("blocked = %d, want 1", stats.Blocked)
	}
	if stats.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", stats.Skipped)
	}
	if stats.Todo != 1 {
		t.Errorf("todo = %d, want 1 (the unrecognised TODO status must fall into the default bucket)", stats.Todo)
	}
	if stats.InProgress != 1 {
		t.Errorf("inProgress = %d, want 1", stats.InProgress)
	}

	// executed = passed + failed + blocked + skipped = 4 of 6
	if stats.ExecutionRate < 66.66 || stats.ExecutionRate > 66.67 {
		t.Errorf("executionRate = %.2f, want ~66.67 (4 executed of 6)", stats.ExecutionRate)
	}
	// passRate = passed / executed = 1/4
	if stats.PassRate != 25 {
		t.Errorf("passRate = %.2f, want 25.00 (1 passed of 4 executed)", stats.PassRate)
	}
	if stats.AllDone {
		t.Errorf("allDone = true while %d items are still TODO and %d IN_PROGRESS", stats.Todo, stats.InProgress)
	}

	passed, failed, blocked, skipped, todo, inProgress := 1, 1, 1, 1, 1, 1
	if sum := passed + failed + blocked + skipped + todo + inProgress; sum != stats.Total {
		t.Errorf("the per-status buckets sum to %d but total is %d; a status is double counted or dropped",
			sum, stats.Total)
	}
}

// TC-IT-WS-04
// Contract: a completed task is idempotent and reports the persisted state.
func TestCompleteTask_PersistsStatus(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "task-complete")
	taskID := newTestTask(t, wsID, "PENDING")

	resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/task/"+taskID+"/complete", nil, authHeader())
	if resp.Code != 200 {
		t.Fatalf("complete task: code=%d message=%s", resp.Code, resp.Message)
	}

	var reloaded model.QualityTask
	if err := config.DB.Where("task_id = ?", taskID).First(&reloaded).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if reloaded.Status != "COMPLETED" {
		t.Errorf("task status = %q, want COMPLETED", reloaded.Status)
	}
}

// TC-IT-WS-05
// Contract risk (P1): completing a task that does not exist still answers with the
// success message. A silent no-op is indistinguishable from a real transition, so
// the client shows a success toast while nothing changed.
func TestCompleteTask_NonExistentTaskMustNotReportSuccess(t *testing.T) {
	requireDB(t)

	missing := newTestID("absent")
	resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+missing+"/task/"+missing+"/complete", nil, authHeader())

	t.Logf("observed: HTTP=%d code=%d message=%q", resp.HTTP, resp.Code, resp.Message)

	testsupport.Contract(t, resp.Code != 200,
		"completing a non-existent task reported code=%d message=%q; an update that matched no row must not report success",
		resp.Code, resp.Message)
}

// TC-IT-WS-06
// Contract: reading a workspace that does not exist is a 404, not an empty object.
func TestGetWorkspaceDetail_MissingWorkspaceIs404(t *testing.T) {
	requireDB(t)

	resp := request(t, http.MethodGet, "/api/quality-workspace/"+newTestID("absent"), nil, authHeader())

	if resp.Code != 404 {
		t.Errorf("body.code = %d, want 404 for an unknown workspace", resp.Code)
	}
}

// TC-IT-WS-07
// Contract: paging metadata must agree with the returned rows, otherwise the
// paginator loops or truncates.
func TestWorkspacePage_PagingMetadataIsConsistent(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "page-meta")
	t.Logf("fixture workspace: %s", wsID)

	resp := request(t, http.MethodPost, "/api/quality-workspace/page", map[string]interface{}{
		"projectId": defaultProjectID,
		"current":   1,
		"pageSize":  5,
	}, authHeader())

	if resp.Code != 200 {
		t.Fatalf("page: code=%d message=%s", resp.Code, resp.Message)
	}

	var page struct {
		List       []map[string]interface{} `json:"list"`
		Total      int64                    `json:"total"`
		Current    int                      `json:"current"`
		PageSize   int                      `json:"pageSize"`
		TotalPages int64                    `json:"totalPages"`
	}
	decode(t, resp, &page)

	if page.Current != 1 || page.PageSize != 5 {
		t.Errorf("echoed paging = %d/%d, want 1/5", page.Current, page.PageSize)
	}
	if len(page.List) > 5 {
		t.Errorf("returned %d rows for pageSize 5", len(page.List))
	}

	expectedPages := (page.Total + int64(page.PageSize) - 1) / int64(page.PageSize)
	if page.TotalPages != expectedPages {
		t.Errorf("totalPages = %d, want %d (total=%d pageSize=%d)",
			page.TotalPages, expectedPages, page.Total, page.PageSize)
	}

	// Archived workspaces are excluded by default; the fixture row must be listed.
	found := false
	for _, row := range page.List {
		if row["workspaceId"] == wsID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("the newly created workspace %s is absent from page 1 of its own project", wsID)
	}
}

// TC-IT-WS-08
// Contract risk (P0): the absent-JSON-column defect is not specific to workspace
// creation. The task form likewise omits metadata, so SaveTask writes an empty
// string into quality_task.metadata and MySQL rejects the insert, which makes the
// entire quality-task feature unusable through the API.
func TestSaveTask_MinimalPayloadCreatesRow(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "task-save")

	resp := request(t, http.MethodPost, "/api/quality-workspace/"+wsID+"/task/save",
		map[string]interface{}{
			"taskType": "FUNCTIONAL",
			"title":    testPrefix + " task created through the api",
		}, authHeader())

	taskID := ""
	if resp.Code == 200 {
		var task struct {
			TaskID string `json:"taskId"`
		}
		decode(t, resp, &task)
		taskID = task.TaskID

		var persisted int64
		config.DB.Model(&model.QualityTask{}).Where("task_id = ?", task.TaskID).Count(&persisted)
		if persisted != 1 {
			t.Errorf("task %s was returned but %d rows exist", task.TaskID, persisted)
		}
	}

	t.Logf("observed: code=%d message=%s taskId=%q", resp.Code, resp.Message, taskID)

	testsupport.Contract(t, resp.Code == 200,
		"creating a task with the minimal form payload failed: code=%d message=%s", resp.Code, resp.Message)
}
