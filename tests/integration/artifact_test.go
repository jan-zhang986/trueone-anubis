package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"trueone-anubis/internal/model"
)

const (
	matrixArtifact = "../../docs/test-architecture/diff-matrix.json"
	baselineDoc    = "../../docs/test-architecture/requirements-baseline.md"
)

// loadMatrixArtifact reads the generated asset through the SAME Go type the HTTP
// endpoint binds to, so schema drift between the artifact and the platform contract
// is a compile-and-assert failure rather than a silent mismatch at push time.
func loadMatrixArtifact(t *testing.T) model.DiffMatrixSyncRequest {
	t.Helper()

	raw, err := os.ReadFile(matrixArtifact)
	if err != nil {
		t.Fatalf("cannot read %s: %v", matrixArtifact, err)
	}

	var request model.DiffMatrixSyncRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("%s does not satisfy the sync contract: %v", matrixArtifact, err)
	}
	return request
}

// TC-ART-01
// Contract: the artifact must bind to DiffMatrixSyncRequest with no field loss.
func TestArtifact_BindsToSyncContract(t *testing.T) {
	request := loadMatrixArtifact(t)

	if request.RepoURL == "" {
		t.Error("repoUrl is empty; the sync contract requires the analysed repository")
	}
	if request.CommitSHA == "" {
		t.Error("commitSha is empty; the matrix cannot be traced back to a revision")
	}
	if request.PrdPath == "" {
		t.Error("prdPath is empty; requirement anchors cannot be located")
	}
	if len(request.Sections) == 0 {
		t.Fatal("sections is empty")
	}

	t.Logf("artifact: repoUrl=%q commitSha=%q prdPath=%q sections=%d",
		request.RepoURL, request.CommitSHA, request.PrdPath, len(request.Sections))
}

// TC-ART-02
// Contract: every enumerated field must stay inside the vocabulary the platform
// renders. An unknown diffStatus or case status would fall through every frontend
// branch and render as a colourless entry.
func TestArtifact_EnumerationsAreInVocabulary(t *testing.T) {
	request := loadMatrixArtifact(t)

	validStatus := map[string]bool{"COVERED": true, "WARNING": true, "GAP": true}
	validRisk := map[string]bool{"P0": true, "P1": true, "P2": true}

	seenNumbers := map[string]bool{}
	cases, passed, failed, missing := 0, 0, 0, 0

	for _, section := range request.Sections {
		if !validStatus[section.DiffStatus] {
			t.Errorf("section %s: diffStatus %q is outside {COVERED, WARNING, GAP}",
				section.SectionNumber, section.DiffStatus)
		}
		if !validRisk[section.Analysis.RiskLevel] {
			t.Errorf("section %s: riskLevel %q is outside {P0, P1, P2}",
				section.SectionNumber, section.Analysis.RiskLevel)
		}
		if seenNumbers[section.SectionNumber] {
			t.Errorf("section number %s appears more than once", section.SectionNumber)
		}
		seenNumbers[section.SectionNumber] = true

		if len(section.Paragraphs) == 0 {
			t.Errorf("section %s: paragraphs is empty; the requirement text is missing",
				section.SectionNumber)
		}
		if len(section.Analysis.ImpactScope) == 0 {
			t.Errorf("section %s: impactScope is empty; the failure mode has no engineering blast radius",
				section.SectionNumber)
		}
		if len(section.Analysis.VerificationChecklist) == 0 {
			t.Errorf("section %s: verificationChecklist is empty", section.SectionNumber)
		}

		for _, tc := range section.Analysis.Cases {
			cases++
			switch tc.Status {
			case "PASSED":
				passed++
			case "FAILED":
				failed++
			case "MISSING":
				missing++
			default:
				t.Errorf("section %s case %s: status %q is outside {PASSED, FAILED, MISSING}",
					section.SectionNumber, tc.ID, tc.Status)
			}

			if strings.TrimSpace(tc.CodeSnippet) == "" {
				t.Errorf("section %s case %s: codeSnippet is empty; a case without code is not test-as-code",
					section.SectionNumber, tc.ID)
			}
			if tc.FilePath == "" || tc.LineNo <= 0 {
				t.Errorf("section %s case %s: filePath/lineNo missing (%q:%d)",
					section.SectionNumber, tc.ID, tc.FilePath, tc.LineNo)
			}
			if tc.Status != "MISSING" && strings.TrimSpace(tc.Evidence.Assertion) == "" {
				t.Errorf("section %s case %s: an executed case must carry an assertion",
					section.SectionNumber, tc.ID)
			}
		}
	}

	t.Logf("artifact: %d sections, %d cases (passed=%d failed=%d missing=%d)",
		len(request.Sections), cases, passed, failed, missing)
}

// TC-ART-03
// Contract: the line anchors must resolve to real lines of the requirement baseline,
// otherwise the matrix's "requirement -> risk" link is decorative.
func TestArtifact_LineAnchorsResolveAgainstBaseline(t *testing.T) {
	request := loadMatrixArtifact(t)

	raw, err := os.ReadFile(baselineDoc)
	if err != nil {
		t.Fatalf("cannot read %s: %v", baselineDoc, err)
	}
	baseline := strings.Split(string(raw), "\n")

	for _, section := range request.Sections {
		if section.LineStart < 1 || section.LineEnd > len(baseline) {
			t.Errorf("section %s: line range %d-%d falls outside the baseline (1-%d)",
				section.SectionNumber, section.LineStart, section.LineEnd, len(baseline))
			continue
		}
		if section.LineStart > section.LineEnd {
			t.Errorf("section %s: lineStart %d > lineEnd %d",
				section.SectionNumber, section.LineStart, section.LineEnd)
			continue
		}

		// The anchored block must actually name the section it claims to describe.
		block := strings.Join(baseline[section.LineStart-1:section.LineEnd], "\n")
		if !strings.Contains(block, section.SectionNumber) {
			t.Errorf("section %s: anchored lines %d-%d do not mention %q",
				section.SectionNumber, section.LineStart, section.LineEnd, section.SectionNumber)
		}
	}

	t.Logf("all %d line anchors resolve inside %s (%d lines)",
		len(request.Sections), filepath.Base(baselineDoc), len(baseline))
}

// TC-ART-04
// Contract: sections flagged GAP must explain themselves. A red status without a
// recorded reason or a production fallback is not actionable.
func TestArtifact_GapSectionsCarryNoticeOrFailures(t *testing.T) {
	request := loadMatrixArtifact(t)

	gaps := 0
	for _, section := range request.Sections {
		if section.DiffStatus != "GAP" {
			continue
		}
		gaps++

		hasFailed := false
		for _, tc := range section.Analysis.Cases {
			if tc.Status == "FAILED" {
				hasFailed = true
				break
			}
		}
		if !hasFailed && section.Analysis.UncoveredNotice == nil {
			t.Errorf("section %s is GAP with no FAILED case and no uncoveredNotice",
				section.SectionNumber)
		}
		if section.Analysis.UncoveredNotice != nil {
			if strings.TrimSpace(section.Analysis.UncoveredNotice.Reason) == "" {
				t.Errorf("section %s: uncoveredNotice.reason is empty", section.SectionNumber)
			}
		}
	}

	if gaps == 0 {
		t.Error("no GAP sections found; the audit would be reporting a green build")
	}
	t.Logf("artifact: %d GAP sections, all carrying justification", gaps)
}
