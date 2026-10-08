package integration

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/testsupport"
)

// --- Response shapes for the Living Test Plan diff matrix ---

type diffMatrix struct {
	WorkspaceID  string        `json:"workspaceId"`
	RepoURL      string        `json:"repoUrl"`
	GitBranch    string        `json:"gitBranch"`
	CommitSHA    string        `json:"commitSha"`
	PrdPath      string        `json:"prdPath"`
	LastSyncedAt int64         `json:"lastSyncedAt"`
	Sections     []diffSection `json:"sections"`
}

type diffSection struct {
	ID            string       `json:"id"`
	SectionNumber string       `json:"sectionNumber"`
	LineStart     int          `json:"lineStart"`
	LineEnd       int          `json:"lineEnd"`
	Title         string       `json:"title"`
	Paragraphs    []string     `json:"paragraphs"`
	DiffStatus    string       `json:"diffStatus"`
	Analysis      diffAnalysis `json:"analysis"`
}

type diffAnalysis struct {
	ID                    string     `json:"id"`
	Title                 string     `json:"title"`
	RiskLevel             string     `json:"riskLevel"`
	RiskTag               string     `json:"riskTag"`
	ImpactScope           []string   `json:"impactScope"`
	VerificationChecklist []string   `json:"verificationChecklist"`
	Cases                 []diffCase `json:"cases"`
}

type diffCase struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Lang        string `json:"lang"`
	FilePath    string `json:"filePath"`
	LineNo      int    `json:"lineNo"`
	Status      string `json:"status"`
	DurationMs  int    `json:"durationMs"`
	CodeSnippet string `json:"codeSnippet"`
	Evidence    struct {
		Assertion   string `json:"assertion"`
		DbStateDiff string `json:"dbStateDiff"`
		TraceLog    string `json:"traceLog"`
	} `json:"evidence"`
}

func findSection(m *diffMatrix, number string) *diffSection {
	for i := range m.Sections {
		if m.Sections[i].SectionNumber == number {
			return &m.Sections[i]
		}
	}
	return nil
}

func sampleCase(id, status string) map[string]interface{} {
	return map[string]interface{}{
		"id": id, "name": "test_" + id, "lang": "go",
		"filePath": "tests/integration/fixture_test.go", "lineNo": 1,
		"status": status, "durationMs": 5,
		"codeSnippet": "func Test" + id + "(t *testing.T) {}",
		"evidence": map[string]interface{}{
			"assertion":   "fixture assertion",
			"dbStateDiff": "fixture diff",
			"traceLog":    "fixture log",
		},
	}
}

func sampleSection(number, diffStatus string, cases []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":            "sec-" + number,
		"sectionNumber": number,
		"lineStart":     1,
		"lineEnd":       20,
		"title":         "fixture section " + number,
		"paragraphs":    []string{"fixture paragraph"},
		"diffStatus":    diffStatus,
		"analysis": map[string]interface{}{
			"id": "ana-" + number, "title": "fixture analysis",
			"riskLevel": "P1", "riskTag": "契约破坏",
			"riskDescription":       "fixture",
			"impactScope":           []string{"fixture"},
			"strategy":              "fixture",
			"verificationChecklist": []string{"fixture check"},
			"cases":                 cases,
		},
	}
}

func syncPayload(sections []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"repoUrl":   "git@github.com:trueone/anubis.git",
		"gitBranch": "main",
		"commitSha": "abc1234",
		"prdPath":   "docs/iterations/sprint/prd.md",
		"sections":  sections,
	}
}

// readStoredMatrix loads the persisted matrix straight out of the workspace row.
func readStoredMatrix(t *testing.T, workspaceID string) map[string]interface{} {
	t.Helper()

	var ws model.QualityWorkspace
	if err := config.DB.Where("workspace_id = ?", workspaceID).First(&ws).Error; err != nil {
		t.Fatalf("reload workspace %s: %v", workspaceID, err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(ws.Metadata), &metadata); err != nil {
		t.Fatalf("workspace metadata is not valid JSON: %v; raw=%s", err, ws.Metadata)
	}

	stored, ok := metadata["diff_matrix"].(map[string]interface{})
	if !ok {
		t.Fatalf("workspace metadata has no diff_matrix key; keys=%v", keysOf(metadata))
	}
	return stored
}

func keysOf(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TC-IT-DM-01
// Contract: syncing a matrix persists it into quality_workspace.metadata under the
// diff_matrix key, and a subsequent read returns the same asset.
func TestSyncDiffMatrix_PersistsAndReadsBack(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "dm-sync")
	payload := syncPayload([]map[string]interface{}{
		sampleSection("1.0", "COVERED", []map[string]interface{}{sampleCase("tc-1", "PASSED")}),
	})

	resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/sync", payload, authHeader())
	if resp.Code != 200 {
		t.Fatalf("sync failed: code=%d message=%s", resp.Code, resp.Message)
	}

	var synced diffMatrix
	decode(t, resp, &synced)
	if synced.CommitSHA != "abc1234" {
		t.Errorf("commitSha = %q, want abc1234", synced.CommitSHA)
	}
	if synced.WorkspaceID != wsID {
		t.Errorf("workspaceId = %q, want %q", synced.WorkspaceID, wsID)
	}
	if synced.LastSyncedAt <= 0 {
		t.Error("lastSyncedAt was not populated; the freshness indicator would render as 1970")
	}
	if len(synced.Sections) != 1 {
		t.Fatalf("sections = %d, want 1", len(synced.Sections))
	}

	// Evidence: the row-level state change.
	stored := readStoredMatrix(t, wsID)
	if got := stored["commitSha"]; got != "abc1234" {
		t.Errorf("persisted metadata diff_matrix.commitSha = %v, want abc1234", got)
	}
	storedSections, _ := stored["sections"].([]interface{})
	if len(storedSections) != 1 {
		t.Errorf("persisted sections = %d, want 1", len(storedSections))
	}

	// Round trip through the read endpoint.
	reload := request(t, http.MethodGet, "/api/quality-workspace/"+wsID+"/diff-matrix", nil, authHeader())
	if reload.Code != 200 {
		t.Fatalf("read back failed: code=%d message=%s", reload.Code, reload.Message)
	}

	var reread diffMatrix
	decode(t, reload, &reread)
	if reread.CommitSHA != "abc1234" {
		t.Errorf("read-back commitSha = %q, want abc1234", reread.CommitSHA)
	}
	if sec := findSection(&reread, "1.0"); sec == nil {
		t.Error("read-back matrix lost section 1.0")
	}
}

// TC-IT-DM-02
// Contract risk (P0): a section whose only case is FAILED keeps reporting COVERED.
// ReportCaseExecution recomputes the section status but leaves it untouched when a
// case failed without any case being missing, so the matrix can show a failed
// requirement as covered.
func TestReportCaseExecution_FailedCaseMustNotRemainCovered(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "dm-failed")
	payload := syncPayload([]map[string]interface{}{
		sampleSection("2.1", "COVERED", []map[string]interface{}{sampleCase("tc-fail", "PASSED")}),
	})

	if resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/sync", payload, authHeader()); resp.Code != 200 {
		t.Fatalf("sync failed: code=%d message=%s", resp.Code, resp.Message)
	}

	report := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/case/report",
		map[string]interface{}{
			"caseId":        "tc-fail",
			"sectionNumber": "2.1",
			"status":        "FAILED",
			"durationMs":    37,
			"assertion":     "assert result.delta == 0.00 [FAIL] got 0.01",
			"dbStateDiff":   "[BLOCKED] transaction rolled back",
			"traceLog":      "imbalance intercepted after 37ms",
		}, authHeader())
	if report.Code != 200 {
		t.Fatalf("case report failed: code=%d message=%s", report.Code, report.Message)
	}

	var matrix diffMatrix
	decode(t, report, &matrix)

	section := findSection(&matrix, "2.1")
	if section == nil {
		t.Fatal("section 2.1 missing from the reported matrix")
	}
	if len(section.Analysis.Cases) != 1 {
		t.Fatalf("cases = %d, want 1", len(section.Analysis.Cases))
	}

	tc := section.Analysis.Cases[0]
	if tc.Status != "FAILED" {
		t.Errorf("case status = %q, want FAILED", tc.Status)
	}
	if tc.DurationMs != 37 {
		t.Errorf("case durationMs = %d, want 37", tc.DurationMs)
	}
	if tc.Evidence.Assertion == "" || tc.Evidence.DbStateDiff == "" || tc.Evidence.TraceLog == "" {
		t.Errorf("evidence was not persisted: %+v", tc.Evidence)
	}

	t.Logf("observed: caseStatus=%q sectionDiffStatus=%q", tc.Status, section.DiffStatus)

	testsupport.Contract(t, section.DiffStatus != "COVERED",
		"a section whose only case is FAILED still reports diffStatus=%q; the recomputation in ReportCaseExecution never runs the failed-only branch",
		section.DiffStatus)
}

// TC-IT-DM-03
// Contract: a section containing a MISSING case must be downgraded to WARNING,
// because an unimplemented test is not evidence of correctness.
func TestReportCaseExecution_SectionWithMissingCaseBecomesWarning(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "dm-warning")
	payload := syncPayload([]map[string]interface{}{
		sampleSection("3.1", "COVERED", []map[string]interface{}{
			sampleCase("tc-ok", "PASSED"),
			sampleCase("tc-missing", "MISSING"),
		}),
	})

	if resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/sync", payload, authHeader()); resp.Code != 200 {
		t.Fatalf("sync failed: code=%d message=%s", resp.Code, resp.Message)
	}

	report := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/case/report",
		map[string]interface{}{
			"caseId": "tc-ok", "sectionNumber": "3.1", "status": "PASSED", "durationMs": 4,
			"assertion": "assert true", "traceLog": "ok",
		}, authHeader())
	if report.Code != 200 {
		t.Fatalf("case report failed: code=%d message=%s", report.Code, report.Message)
	}

	var matrix diffMatrix
	decode(t, report, &matrix)

	section := findSection(&matrix, "3.1")
	if section == nil {
		t.Fatal("section 3.1 missing")
	}
	if section.DiffStatus != "WARNING" {
		t.Errorf("diffStatus = %q, want WARNING while a case is MISSING", section.DiffStatus)
	}
}

// TC-IT-DM-04
// Contract risk (P1): a section with zero cases can never be derived as GAP. The
// GAP branch sits inside the caseFound guard, and caseFound is only true when a
// case exists, so the branch is unreachable and the status stays whatever the
// client last sent.
func TestReportCaseExecution_SectionWithoutCasesMustBecomeGap(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "dm-gap")
	payload := syncPayload([]map[string]interface{}{
		sampleSection("4.1", "COVERED", []map[string]interface{}{}),
	})

	if resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/sync", payload, authHeader()); resp.Code != 200 {
		t.Fatalf("sync failed: code=%d message=%s", resp.Code, resp.Message)
	}

	report := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/case/report",
		map[string]interface{}{
			"caseId": "tc-anything", "sectionNumber": "4.1", "status": "PASSED", "durationMs": 1,
		}, authHeader())
	if report.Code != 200 {
		t.Fatalf("case report failed: code=%d message=%s", report.Code, report.Message)
	}

	var matrix diffMatrix
	decode(t, report, &matrix)

	section := findSection(&matrix, "4.1")
	if section == nil {
		t.Fatal("section 4.1 missing")
	}

	t.Logf("observed: cases=%d diffStatus=%q", len(section.Analysis.Cases), section.DiffStatus)

	testsupport.Contract(t, section.DiffStatus == "GAP",
		"a requirement section with zero test cases still reports diffStatus=%q; the GAP branch is unreachable because caseFound requires an existing case",
		section.DiffStatus)
}

// TC-IT-DM-05
// Contract risk (P0): reading the matrix of a workspace that has never been synced
// invents a hardcoded asset - including cases already reported as PASSED - for an
// unrelated repository, and then persists it. Fabricated green evidence is the most
// damaging possible defect in a quality platform.
func TestGetDiffMatrix_MustNotFabricateEvidenceForAnUnsyncedWorkspace(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "dm-fabricated")

	resp := request(t, http.MethodGet, "/api/quality-workspace/"+wsID+"/diff-matrix", nil, authHeader())
	if resp.Code != 200 {
		t.Fatalf("read: code=%d message=%s", resp.Code, resp.Message)
	}

	var matrix diffMatrix
	decode(t, resp, &matrix)

	passedCases := 0
	for _, section := range matrix.Sections {
		for _, tc := range section.Analysis.Cases {
			if tc.Status == "PASSED" {
				passedCases++
			}
		}
	}

	t.Logf("observed: repoUrl=%q prdPath=%q sections=%d fabricatedPassedCases=%d",
		matrix.RepoURL, matrix.PrdPath, len(matrix.Sections), passedCases)

	testsupport.Contract(t, matrix.RepoURL != "git@github.com:vanguard/trade-payment-service.git",
		"GET diff-matrix invented an asset for the unrelated repository %q", matrix.RepoURL)
	testsupport.Contract(t, passedCases == 0,
		"GET diff-matrix invented %d test cases already marked PASSED for a workspace that was never analysed", passedCases)

	// The invented asset must not have been written to the database either.
	stored := readStoredMatrix(t, wsID)
	testsupport.Contract(t, stored["repoUrl"] != "git@github.com:vanguard/trade-payment-service.git",
		"the fabricated asset was persisted into quality_workspace.metadata for %s", wsID)
}

// TC-IT-DM-06
// Contract: report generation persists a report row carrying a snapshot of the
// workspace statistics.
func TestGenerateReport_PersistsReportWithSnapshot(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "report-gen")
	taskID := newTestID("task")
	for i := 0; i < 2; i++ {
		newTestWorkItem(t, wsID, taskID, "PASSED")
	}

	resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/report/generate",
		map[string]interface{}{"reportType": "OVERVIEW"}, authHeader())

	var reportID string
	if resp.Code == 200 {
		var report struct {
			ReportID string `json:"reportId"`
			Status   string `json:"status"`
			Snapshot string `json:"snapshotJson"`
		}
		decode(t, resp, &report)
		reportID = report.ReportID

		var persisted int64
		config.DB.Model(&model.QualityReport{}).Where("report_id = ?", report.ReportID).Count(&persisted)
		if persisted != 1 {
			t.Errorf("report %s was returned but %d rows exist", report.ReportID, persisted)
		}
		if report.Snapshot == "" {
			t.Error("snapshotJson is empty; the report cannot be reproduced later")
		}
	}

	t.Logf("observed: code=%d message=%s reportId=%q", resp.Code, resp.Message, reportID)

	testsupport.Contract(t, resp.Code == 200,
		"report generation failed: code=%d message=%s", resp.Code, resp.Message)
}

// TC-IT-DM-07
// Contract: generation from Markdown rejects an empty document before spending a
// model call.
func TestGenerateDiffMatrixFromMarkdown_RejectsEmptyDocument(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "dm-generate")

	empty := request(t, http.MethodPost, "/api/quality-workspace/"+wsID+"/diff-matrix/generate",
		map[string]interface{}{"markdownContent": "   "}, authHeader())
	if empty.Code != 400 {
		t.Errorf("empty markdown: code = %d, want 400; the model must not be called for an empty document", empty.Code)
	}
}

// TC-IT-DM-07b
// External dependency: the generation path calls the DeepSeek HTTP API, a paid
// third-party service whose output is non-deterministic. It therefore cannot back a
// deterministic assertion, so the live call is opt-in via ANUBIS_LIVE_LLM=1. The
// deterministic parts - input validation and the response contract - are covered by
// the surrounding tests.
func TestGenerateDiffMatrixFromMarkdown_LiveModelCall(t *testing.T) {
	requireDB(t)

	if os.Getenv("ANUBIS_LIVE_LLM") != "1" {
		t.Skip("live model call is opt-in; set ANUBIS_LIVE_LLM=1 to exercise the DeepSeek API")
	}

	wsID := newTestWorkspace(t, "dm-live")

	resp := request(t, http.MethodPost, "/api/quality-workspace/"+wsID+"/diff-matrix/generate",
		map[string]interface{}{
			"markdownContent": "# 需求\n\n1.0 系统必须校验借贷平衡。\n",
			"prdTitle":        "fixture-prd.md",
		}, authHeader())

	if resp.Code != 200 {
		t.Fatalf("generation failed: code=%d message=%s", resp.Code, resp.Message)
	}

	var matrix diffMatrix
	decode(t, resp, &matrix)
	if len(matrix.Sections) == 0 {
		t.Fatal("generation returned no sections")
	}

	testsupport.Contract(t, matrix.RepoURL != "git@github.com:vanguard/trade-payment-service.git",
		"the generated matrix attributes its analysis to the hardcoded unrelated repository %q", matrix.RepoURL)
}

// TC-IT-DM-06
// Contract: the execution report must persist the complete evidence triple - the
// assertion, the database state diff and the trace log - because those three fields
// are what make a matrix entry reproducible rather than a bare status label.
func TestReportCaseExecution_PersistsEvidence(t *testing.T) {
	requireDB(t)

	wsID := newTestWorkspace(t, "dm-evidence")
	payload := syncPayload([]map[string]interface{}{
		sampleSection("5.1", "COVERED", []map[string]interface{}{sampleCase("tc-evidence", "PASSED")}),
	})

	if resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/sync", payload, authHeader()); resp.Code != 200 {
		t.Fatalf("sync failed: code=%d message=%s", resp.Code, resp.Message)
	}

	const (
		wantAssertion = "assert verdict.delta == 0.00 [FAIL] got 0.01"
		wantDbDiff    = "[BLOCKED] transaction rolled back, no row inserted"
		wantTrace     = "imbalance intercepted after 37ms"
	)

	report := request(t, http.MethodPost,
		"/api/quality-workspace/"+wsID+"/diff-matrix/case/report",
		map[string]interface{}{
			"caseId":        "tc-evidence",
			"sectionNumber": "5.1",
			"status":        "FAILED",
			"durationMs":    37,
			"assertion":     wantAssertion,
			"dbStateDiff":   wantDbDiff,
			"traceLog":      wantTrace,
		}, authHeader())
	if report.Code != 200 {
		t.Fatalf("case report failed: code=%d message=%s", report.Code, report.Message)
	}

	var matrix diffMatrix
	decode(t, report, &matrix)

	section := findSection(&matrix, "5.1")
	if section == nil || len(section.Analysis.Cases) != 1 {
		t.Fatal("section 5.1 or its single case is missing from the reported matrix")
	}
	tc := section.Analysis.Cases[0]

	if tc.Status != "FAILED" {
		t.Errorf("status = %q, want FAILED", tc.Status)
	}
	if tc.DurationMs != 37 {
		t.Errorf("durationMs = %d, want 37", tc.DurationMs)
	}
	if tc.Evidence.Assertion != wantAssertion {
		t.Errorf("assertion = %q, want %q", tc.Evidence.Assertion, wantAssertion)
	}
	if tc.Evidence.DbStateDiff != wantDbDiff {
		t.Errorf("dbStateDiff = %q, want %q", tc.Evidence.DbStateDiff, wantDbDiff)
	}
	if tc.Evidence.TraceLog != wantTrace {
		t.Errorf("traceLog = %q, want %q", tc.Evidence.TraceLog, wantTrace)
	}
}

// TC-IT-DM-08
// Contract: reporting execution against a workspace that does not exist must be
// rejected. The read path currently fabricates a default matrix instead of
// surfacing the not-found error, so the report lands against invented state.
func TestReportCaseExecution_UnknownWorkspaceFails(t *testing.T) {
	requireDB(t)

	resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+newTestID("absent")+"/diff-matrix/case/report",
		map[string]interface{}{"caseId": "tc-1", "status": "PASSED"}, authHeader())

	t.Logf("observed: HTTP=%d code=%d message=%s", resp.HTTP, resp.Code, resp.Message)

	testsupport.Contract(t, resp.Code != 200,
		"reporting case execution against a non-existent workspace returned code=%d; the not-found error is swallowed by the fabricated default matrix",
		resp.Code)
}
