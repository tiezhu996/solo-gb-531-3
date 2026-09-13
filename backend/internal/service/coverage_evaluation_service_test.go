package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"hazop-safeguard-coverage/backend/internal/algorithm"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"
)

func evidenceService(t *testing.T) (CoverageEvaluationService, repository.CoverageEvaluationRepository, model.ProcessNode, model.DeviationScenario) {
	t.Helper()
	db := testDB(t)
	nodeRepo := repository.NewProcessNodeRepository(db)
	scenarioRepo := repository.NewDeviationScenarioRepository(db)
	safeguardRepo := repository.NewSafeguardRepository(db)
	evaluationRepo := repository.NewCoverageEvaluationRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	node := model.ProcessNode{
		NodeCode: "R-900", Name: "Evidence Reactor", UnitName: "Evidence Unit", Medium: "propylene",
		DesignPressure: 2.5, DesignTemperature: 180, OwnerTeam: "pss", Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := nodeRepo.Create(ctx, &node); err != nil {
		t.Fatalf("create node: %v", err)
	}
	scenario := model.DeviationScenario{
		ProcessNodeID: node.ID, Guideword: "more", Parameter: "temperature",
		Cause: "cooling loss", Consequence: "overpressure",
		Likelihood: 4, Severity: 5, ScenarioState: "analyzed", Version: 1,
		CreatedBy: 7, CreatedByName: "author", CreatedAt: now, UpdatedAt: now,
	}
	if err := scenarioRepo.Create(ctx, &scenario); err != nil {
		t.Fatalf("create scenario: %v", err)
	}
	verified := now.AddDate(0, 0, -5)
	safeguards := []model.Safeguard{
		{
			Name: "High temperature trip A", SafeguardType: "sis", TargetScenarioID: scenario.ID,
			IndependenceKey: "SIS-TEMP", Effectiveness: 0.9, TestIntervalDays: 365,
			LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "proof test record A",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			Name: "Duplicate temperature trip", SafeguardType: "sis", TargetScenarioID: scenario.ID,
			IndependenceKey: "SIS-TEMP", Effectiveness: 0.6, TestIntervalDays: 365,
			LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "shares the same sensor logic solver",
			CreatedAt: now, UpdatedAt: now,
		},
	}
	for index := range safeguards {
		if err := safeguardRepo.Create(ctx, &safeguards[index]); err != nil {
			t.Fatalf("create safeguard: %v", err)
		}
	}
	svc := NewCoverageEvaluationService(
		evaluationRepo, scenarioRepo, nodeRepo, safeguardRepo, auditRepo, algorithm.NewEvaluator(),
	)
	return svc, evaluationRepo, node, scenario
}

func runEvaluationForExport(t *testing.T, svc CoverageEvaluationService, scenarioID uint) dto.CoverageEvaluationResponse {
	t.Helper()
	actor := util.Actor{UserID: 7, Username: "author", Role: "process_engineer", RequestID: "evidence-run"}
	result, duplicate, err := svc.Run(context.Background(), dto.RunCoverageEvaluationRequest{ScenarioID: scenarioID}, "evidence-key-completed-001", actor)
	if err != nil || duplicate {
		t.Fatalf("run evaluation: duplicate=%t err=%v", duplicate, err)
	}
	if result.EvaluationState != "completed" {
		t.Fatalf("unexpected evaluation state %s", result.EvaluationState)
	}
	return result
}

func TestExportEvidencePackContainsSnapshotStepsUncoveredAndDeduplication(t *testing.T) {
	svc, _, _, scenario := evidenceService(t)
	evaluation := runEvaluationForExport(t, svc, scenario.ID)

	pack, err := svc.ExportEvidencePack(context.Background(), evaluation.ID)
	if err != nil {
		t.Fatalf("export evidence pack: %v", err)
	}
	if pack.PackVersion != "evidence-pack-v1" || pack.ExportedAt.IsZero() {
		t.Fatalf("pack metadata missing: %#v", pack)
	}
	if pack.EvaluationID != evaluation.ID || pack.ScenarioID != scenario.ID {
		t.Fatalf("pack identity mismatch: %#v", pack)
	}
	if pack.State.Code != "completed" || pack.State.ReadableSummary == "" {
		t.Fatalf("completed state must be readable: %#v", pack.State)
	}
	if pack.InputHash == "" || pack.InputHash != evaluation.InputHash {
		t.Fatalf("input hash mismatch: pack=%q evaluation=%q", pack.InputHash, evaluation.InputHash)
	}
	var snapshot algorithm.Snapshot
	if err := json.Unmarshal(pack.InputSnapshot, &snapshot); err != nil {
		t.Fatalf("pack input snapshot must be the frozen JSON snapshot: %v", err)
	}
	if snapshot.Node.NodeCode != "R-900" || snapshot.Scenario.ID != scenario.ID || len(snapshot.Safeguards) != 2 {
		t.Fatalf("snapshot content mismatch: %#v", snapshot)
	}
	if len(pack.ScoreSteps) == 0 {
		t.Fatal("completed pack must carry scoring steps")
	}
	foundCombination := false
	for _, step := range pack.ScoreSteps {
		if step.Rule == "independent-layer-combination" {
			foundCombination = true
		}
	}
	if !foundCombination {
		t.Fatalf("scoring steps must explain layer combination: %#v", pack.ScoreSteps)
	}
	if len(pack.UncoveredPaths) != 0 {
		t.Fatalf("90%% protection covers the single path, got %d uncovered", len(pack.UncoveredPaths))
	}
	if len(pack.DeduplicatedSafeguards) != 1 || pack.DeduplicatedSafeguards[0].IndependenceKey != "SIS-TEMP" {
		t.Fatalf("pack must explain independence deduplication: %#v", pack.DeduplicatedSafeguards)
	}
	deduped := pack.DeduplicatedSafeguards[0]
	if deduped.KeptID == 0 || len(deduped.IgnoredIDs) != 1 || deduped.Reason == "" {
		t.Fatalf("deduplication detail incomplete: %#v", deduped)
	}
	if pack.RiskRankBefore == "" || pack.RiskRankAfter == "" || pack.BoundaryNote == "" {
		t.Fatal("risk ranks and offline boundary note must be exported")
	}
}

func TestExportEvidencePackDescribesFailedEvaluation(t *testing.T) {
	svc, repo, _, scenario := evidenceService(t)
	now := time.Now().UTC()
	snapshot := algorithm.NewSnapshot(
		model.ProcessNode{ID: 1, NodeCode: "R-900"},
		model.DeviationScenario{ID: scenario.ID, Guideword: "more", Parameter: "temperature", Cause: "x", Consequence: "y", Likelihood: 4, Severity: 5},
		nil, now,
	)
	snapshotJSON, err := util.CanonicalJSON(snapshot)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	// Simulate the defect scenario: a failed row still carries stale, even malformed,
	// conclusion columns. The failed export path must neither surface them nor fail parsing.
	staleExplanationJSON := `{broken-json`
	staleUncovered := `{broken-json`
	staleDedup := `{broken-json`
	failed := model.CoverageEvaluation{
		ScenarioID: scenario.ID, AlgorithmVersion: algorithm.Version,
		InputSnapshot: snapshotJSON, InputHash: util.HashString(snapshotJSON),
		UncoveredPaths: staleUncovered, DeduplicatedSafeguards: staleDedup, Explanation: staleExplanationJSON,
		CoverageScore: 70, RiskRankBefore: "high", RiskRankAfter: "low", EvaluationState: "failed",
		EvaluatedBy: 7, EvaluatedByName: "author", EvaluatedAt: now,
		CreatedAt: now, UpdatedAt: now, IdempotencyKey: "evidence-key-failed-0001",
		FailureReason: "simulated determinism failure",
	}
	if err := repo.Create(context.Background(), &failed); err != nil {
		t.Fatalf("create failed evaluation: %v", err)
	}
	pack, err := svc.ExportEvidencePack(context.Background(), failed.ID)
	if err != nil {
		t.Fatalf("export failed evidence pack: %v", err)
	}
	if pack.State.Code != "failed" || pack.State.FailureReason != "simulated determinism failure" {
		t.Fatalf("failed state detail mismatch: %#v", pack.State)
	}
	if !strings.Contains(pack.State.ReadableSummary, "结论字段") {
		t.Fatalf("failed summary must state conclusions are cleared, got %q", pack.State.ReadableSummary)
	}
	if pack.CoverageScore != 0 {
		t.Fatalf("failed pack coverage score must be 0, got %v", pack.CoverageScore)
	}
	if len(pack.ScoreSteps) != 0 || len(pack.UncoveredPaths) != 0 || len(pack.DeduplicatedSafeguards) != 0 {
		t.Fatalf("failed pack must suppress stale conclusions: steps=%d uncovered=%d dedup=%d",
			len(pack.ScoreSteps), len(pack.UncoveredPaths), len(pack.DeduplicatedSafeguards))
	}
	if pack.RiskRankAfter != "" {
		t.Fatalf("failed pack risk rank after must be empty, got %q", pack.RiskRankAfter)
	}
	if pack.RiskRankBefore != "high" || !json.Valid(pack.InputSnapshot) || pack.InputHash == "" || pack.BoundaryNote == "" {
		t.Fatal("failed pack must keep frozen input, hash, before-risk and offline boundary note")
	}
}

func TestExportEvidencePackDescribesVoidedEvaluation(t *testing.T) {
	svc, _, _, scenario := evidenceService(t)
	evaluation := runEvaluationForExport(t, svc, scenario.ID)
	reviewer := util.Actor{UserID: 42, Username: "reviewer", Role: "safety_reviewer", RequestID: "evidence-void"}
	voided, err := svc.Void(context.Background(), evaluation.ID, reviewer)
	if err != nil || voided.EvaluationState != "voided" {
		t.Fatalf("void evaluation: state=%s err=%v", voided.EvaluationState, err)
	}
	pack, err := svc.ExportEvidencePack(context.Background(), evaluation.ID)
	if err != nil {
		t.Fatalf("export voided evidence pack: %v", err)
	}
	if pack.State.Code != "voided" || pack.State.ReadableSummary == "" {
		t.Fatalf("voided state must be readable: %#v", pack.State)
	}
	if len(pack.ScoreSteps) == 0 {
		t.Fatal("voided pack retains the historical scoring trace for audit")
	}
}

func TestExportEvidencePackErrors(t *testing.T) {
	svc, repo, _, scenario := evidenceService(t)

	_, err := svc.ExportEvidencePack(context.Background(), 99999)
	var notFound *util.AppError
	if !errors.As(err, &notFound) || notFound.Status != 404 {
		t.Fatalf("missing evaluation must return 404, got %v", err)
	}

	now := time.Now().UTC()
	validBase := func(key, state string) model.CoverageEvaluation {
		return model.CoverageEvaluation{
			ScenarioID: scenario.ID, AlgorithmVersion: algorithm.Version,
			InputSnapshot: `{"algorithm_version":"hazop-cover-v1.0.0"}`, InputHash: "broken",
			UncoveredPaths: "[]", DeduplicatedSafeguards: "[]", Explanation: "{}",
			RiskRankBefore: "high", RiskRankAfter: "high", EvaluationState: state,
			EvaluatedBy: 7, EvaluatedByName: "author", EvaluatedAt: now,
			CreatedAt: now, UpdatedAt: now, IdempotencyKey: key,
		}
	}
	corruptCases := []struct {
		name      string
		state     string
		key       string
		mutate    func(*model.CoverageEvaluation)
		wantError string
	}{
		// Failed packs still export the frozen input, so a corrupt snapshot must be rejected.
		{name: "snapshot", state: "failed", key: "evidence-corrupt-snapshot-1", mutate: func(e *model.CoverageEvaluation) { e.InputSnapshot = "{not-json" }, wantError: "frozen input snapshot"},
		// Conclusion columns are only parsed for non-failed states; completed export must name the broken artifact.
		{name: "explanation", state: "completed", key: "evidence-corrupt-explain-01", mutate: func(e *model.CoverageEvaluation) { e.Explanation = "{not-json" }, wantError: "scoring steps explanation"},
		{name: "uncovered", state: "completed", key: "evidence-corrupt-paths-001", mutate: func(e *model.CoverageEvaluation) { e.UncoveredPaths = "{not-json" }, wantError: "uncovered paths record"},
		{name: "dedup", state: "completed", key: "evidence-corrupt-dedup-001", mutate: func(e *model.CoverageEvaluation) { e.DeduplicatedSafeguards = "{not-json" }, wantError: "independence dedup record"},
	}
	for _, tc := range corruptCases {
		t.Run(tc.name, func(t *testing.T) {
			evaluation := validBase(tc.key, tc.state)
			tc.mutate(&evaluation)
			if err := repo.Create(context.Background(), &evaluation); err != nil {
				t.Fatalf("create corrupt evaluation: %v", err)
			}
			_, err = svc.ExportEvidencePack(context.Background(), evaluation.ID)
			var invalid *util.AppError
			if !errors.As(err, &invalid) || invalid.Status != 422 || !strings.Contains(invalid.Message, tc.wantError) {
				t.Fatalf("corrupt %s must surface a 422 naming %q, got %v", tc.name, tc.wantError, err)
			}
		})
	}
}

// TestEvidencePackExportRealServiceRegression drives the real Run algorithm to produce
// genuine scoring steps, uncovered paths and dedup notes, then verifies that a failed
// record carrying those residual conclusions exports only the frozen input, hash and
// failure reason. It also regresses completed, voided and corrupt-snapshot exports.
func TestEvidencePackExportRealServiceRegression(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	nodeRepo := repository.NewProcessNodeRepository(db)
	scenarioRepo := repository.NewDeviationScenarioRepository(db)
	safeguardRepo := repository.NewSafeguardRepository(db)
	evalRepo := repository.NewCoverageEvaluationRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	svc := NewCoverageEvaluationService(evalRepo, scenarioRepo, nodeRepo, safeguardRepo, auditRepo, algorithm.NewEvaluator())

	now := time.Now().UTC()
	node := model.ProcessNode{
		NodeCode: "R-910", Name: "Regression Reactor", UnitName: "Regression Unit", Medium: "propylene",
		DesignPressure: 3, DesignTemperature: 210, OwnerTeam: "pss", Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := nodeRepo.Create(ctx, &node); err != nil {
		t.Fatalf("create node: %v", err)
	}
	// Scenario A has two safeguards sharing one independence key: yields real score steps + dedup.
	protected := model.DeviationScenario{
		ProcessNodeID: node.ID, Guideword: "more", Parameter: "temperature",
		Cause: "cooling loss", Consequence: "overpressure", Likelihood: 4, Severity: 5,
		ScenarioState: "analyzed", Version: 1, CreatedBy: 7, CreatedByName: "engineer", CreatedAt: now, UpdatedAt: now,
	}
	if err := scenarioRepo.Create(ctx, &protected); err != nil {
		t.Fatalf("create protected scenario: %v", err)
	}
	verified := now.AddDate(0, 0, -5)
	for _, item := range []model.Safeguard{
		{Name: "SIS trip A", SafeguardType: "sis", TargetScenarioID: protected.ID, IndependenceKey: "SIS-910", Effectiveness: 0.9, TestIntervalDays: 365, LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "a", CreatedAt: now, UpdatedAt: now},
		{Name: "SIS trip duplicate", SafeguardType: "sis", TargetScenarioID: protected.ID, IndependenceKey: "SIS-910", Effectiveness: 0.6, TestIntervalDays: 365, LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "b", CreatedAt: now, UpdatedAt: now},
	} {
		guard := item
		if err := safeguardRepo.Create(ctx, &guard); err != nil {
			t.Fatalf("create safeguard: %v", err)
		}
	}
	// Scenario B has no safeguards: yields a real uncovered path and zero coverage.
	unprotected := model.DeviationScenario{
		ProcessNodeID: node.ID, Guideword: "less", Parameter: "flow",
		Cause: "pump trip", Consequence: "dry run", Likelihood: 3, Severity: 4,
		ScenarioState: "analyzed", Version: 1, CreatedBy: 7, CreatedByName: "engineer", CreatedAt: now, UpdatedAt: now,
	}
	if err := scenarioRepo.Create(ctx, &unprotected); err != nil {
		t.Fatalf("create unprotected scenario: %v", err)
	}

	author := util.Actor{UserID: 7, Username: "engineer", Role: "process_engineer", RequestID: "reg-run-protected"}
	completed, duplicate, err := svc.Run(ctx, dto.RunCoverageEvaluationRequest{ScenarioID: protected.ID}, "reg-completed-0001", author)
	if err != nil || duplicate || completed.EvaluationState != "completed" {
		t.Fatalf("protected run: state=%s duplicate=%t err=%v", completed.EvaluationState, duplicate, err)
	}
	_, _, err = svc.Run(ctx, dto.RunCoverageEvaluationRequest{ScenarioID: unprotected.ID}, "reg-uncovered-0001",
		util.Actor{UserID: 7, Username: "engineer", Role: "process_engineer", RequestID: "reg-run-uncovered"})
	if err != nil {
		t.Fatalf("unprotected run: %v", err)
	}
	completedRow, err := evalRepo.FindByIdempotencyKey(ctx, "reg-completed-0001")
	if err != nil {
		t.Fatalf("load completed row: %v", err)
	}
	uncoveredRow, err := evalRepo.FindByIdempotencyKey(ctx, "reg-uncovered-0001")
	if err != nil {
		t.Fatalf("load uncovered row: %v", err)
	}
	// The real algorithm must have populated all three conclusion artifacts across the runs.
	if len(completed.Explanation.ScoreSteps) == 0 || len(completed.DeduplicatedSafeguards) == 0 {
		t.Fatalf("protected run must yield real steps/dedup: steps=%d dedup=%d",
			len(completed.Explanation.ScoreSteps), len(completed.DeduplicatedSafeguards))
	}
	if len(uncoveredRow.UncoveredPaths) <= len("[]") {
		t.Fatalf("unprotected run must persist a real uncovered path: %s", uncoveredRow.UncoveredPaths)
	}

	// Build a failed row whose residual conclusion columns merge the genuine protected and
	// unprotected artifacts, plus a positive stale score and residual risk.
	staleUncovered := uncoveredRow.UncoveredPaths
	staleDedup := completedRow.DeduplicatedSafeguards
	staleExplanation := completedRow.Explanation
	failed := model.CoverageEvaluation{
		ScenarioID: protected.ID, AlgorithmVersion: algorithm.Version,
		InputSnapshot: completedRow.InputSnapshot, InputHash: completedRow.InputHash,
		UncoveredPaths: staleUncovered, DeduplicatedSafeguards: staleDedup, Explanation: staleExplanation,
		CoverageScore: completedRow.CoverageScore, RiskRankBefore: completedRow.RiskRankBefore,
		RiskRankAfter: completedRow.RiskRankAfter, EvaluationState: "failed",
		EvaluatedBy: 7, EvaluatedByName: "engineer", EvaluatedAt: now,
		CreatedAt: now, UpdatedAt: now, IdempotencyKey: "reg-failed-residual-1",
		FailureReason: "residual failure: deterministic check failed after partial write",
	}
	if err := evalRepo.Create(ctx, &failed); err != nil {
		t.Fatalf("create residual failed row: %v", err)
	}

	// Waiting rows may carry the same residual conclusions (and malformed conclusion JSON).
	queued := model.CoverageEvaluation{
		ScenarioID: protected.ID, AlgorithmVersion: algorithm.Version,
		InputSnapshot: completedRow.InputSnapshot, InputHash: completedRow.InputHash,
		UncoveredPaths: "{broken", DeduplicatedSafeguards: "{broken", Explanation: "{broken",
		CoverageScore: completedRow.CoverageScore, RiskRankBefore: completedRow.RiskRankBefore,
		RiskRankAfter: completedRow.RiskRankAfter, EvaluationState: "queued",
		EvaluatedBy: 7, EvaluatedByName: "engineer", EvaluatedAt: now,
		CreatedAt: now, UpdatedAt: now, IdempotencyKey: "reg-queued-residual-1",
	}
	running := model.CoverageEvaluation{
		ScenarioID: protected.ID, AlgorithmVersion: algorithm.Version,
		InputSnapshot: completedRow.InputSnapshot, InputHash: completedRow.InputHash,
		UncoveredPaths: staleUncovered, DeduplicatedSafeguards: staleDedup, Explanation: staleExplanation,
		CoverageScore: completedRow.CoverageScore, RiskRankBefore: completedRow.RiskRankBefore,
		RiskRankAfter: completedRow.RiskRankAfter, EvaluationState: "running",
		EvaluatedBy: 7, EvaluatedByName: "engineer", EvaluatedAt: now,
		CreatedAt: now, UpdatedAt: now, IdempotencyKey: "reg-running-residual-1",
	}
	for _, waiting := range []*model.CoverageEvaluation{&queued, &running} {
		if err := evalRepo.Create(ctx, waiting); err != nil {
			t.Fatalf("create residual %s row: %v", waiting.EvaluationState, err)
		}
	}

	assertWaitingPack := func(t *testing.T, id uint, state string) {
		t.Helper()
		pack, err := svc.ExportEvidencePack(ctx, id)
		if err != nil {
			t.Fatalf("export %s pack: %v", state, err)
		}
		if pack.State.Code != state {
			t.Fatalf("%s state code = %q", state, pack.State.Code)
		}
		if string(pack.InputSnapshot) != completedRow.InputSnapshot || pack.InputHash != completedRow.InputHash {
			t.Fatalf("%s pack must keep frozen input and hash", state)
		}
		if !strings.Contains(pack.State.ReadableSummary, "结论字段") {
			t.Fatalf("%s summary must state conclusion fields are cleared: %q", state, pack.State.ReadableSummary)
		}
		if pack.CoverageScore != 0 || pack.RiskRankAfter != "" ||
			len(pack.ScoreSteps) != 0 || len(pack.UncoveredPaths) != 0 || len(pack.DeduplicatedSafeguards) != 0 {
			t.Fatalf("%s pack leaked residual conclusions: score=%v riskAfter=%q steps=%d uncovered=%d dedup=%d",
				state, pack.CoverageScore, pack.RiskRankAfter, len(pack.ScoreSteps), len(pack.UncoveredPaths), len(pack.DeduplicatedSafeguards))
		}
		if pack.RiskRankBefore == "" || pack.BoundaryNote == "" {
			t.Fatalf("%s pack must retain frozen input metadata", state)
		}
	}

	t.Run("queued suppresses residual conclusions", func(t *testing.T) {
		assertWaitingPack(t, queued.ID, "queued")
	})

	t.Run("running suppresses residual conclusions", func(t *testing.T) {
		assertWaitingPack(t, running.ID, "running")
	})

	t.Run("failed suppresses residual conclusions", func(t *testing.T) {
		pack, err := svc.ExportEvidencePack(ctx, failed.ID)
		if err != nil {
			t.Fatalf("export failed pack: %v", err)
		}
		// Frozen input, hash and failure reason survive.
		if string(pack.InputSnapshot) != completedRow.InputSnapshot {
			t.Fatal("failed pack must keep the frozen input snapshot byte-for-byte")
		}
		if pack.InputHash != completedRow.InputHash {
			t.Fatalf("failed pack must keep input hash %q, got %q", completedRow.InputHash, pack.InputHash)
		}
		if pack.State.Code != "failed" || pack.State.FailureReason != failed.FailureReason {
			t.Fatalf("failed state must carry failure reason: %#v", pack.State)
		}
		// Every conclusion artifact is cleared even though the row held real results.
		if pack.CoverageScore != 0 {
			t.Fatalf("failed pack score must be 0, got %v", pack.CoverageScore)
		}
		if len(pack.ScoreSteps) != 0 {
			t.Fatalf("failed pack must clear %d residual score steps", len(pack.ScoreSteps))
		}
		if len(pack.UncoveredPaths) != 0 {
			t.Fatalf("failed pack must clear %d residual uncovered paths", len(pack.UncoveredPaths))
		}
		if len(pack.DeduplicatedSafeguards) != 0 {
			t.Fatalf("failed pack must clear %d residual dedup notes", len(pack.DeduplicatedSafeguards))
		}
		if pack.RiskRankAfter != "" {
			t.Fatalf("failed pack must clear residual risk rank, got %q", pack.RiskRankAfter)
		}
		if !strings.Contains(pack.State.ReadableSummary, "结论字段") {
			t.Fatalf("failed summary must explain conclusions are cleared: %q", pack.State.ReadableSummary)
		}
		// Input-side metadata is still evidence.
		if pack.RiskRankBefore == "" || pack.BoundaryNote == "" || pack.AlgorithmVersion == "" {
			t.Fatal("failed pack must retain before-risk, boundary note and algorithm version")
		}
		// Repeatable: a second export produces the same frozen input and cleared conclusions.
		again, err := svc.ExportEvidencePack(ctx, failed.ID)
		if err != nil {
			t.Fatalf("repeat export: %v", err)
		}
		if string(again.InputSnapshot) != string(pack.InputSnapshot) || again.InputHash != pack.InputHash ||
			len(again.ScoreSteps) != 0 || len(again.UncoveredPaths) != 0 || len(again.DeduplicatedSafeguards) != 0 ||
			again.CoverageScore != 0 || again.RiskRankAfter != "" {
			t.Fatal("repeated failed export is inconsistent")
		}
	})

	t.Run("completed keeps real conclusions", func(t *testing.T) {
		pack, err := svc.ExportEvidencePack(ctx, completedRow.ID)
		if err != nil {
			t.Fatalf("export completed pack: %v", err)
		}
		if pack.State.Code != "completed" || pack.CoverageScore != completedRow.CoverageScore {
			t.Fatalf("completed pack must keep score %v, got %v state=%s",
				completedRow.CoverageScore, pack.CoverageScore, pack.State.Code)
		}
		if len(pack.ScoreSteps) == 0 || len(pack.DeduplicatedSafeguards) == 0 {
			t.Fatalf("completed pack must keep real steps and dedup: steps=%d dedup=%d",
				len(pack.ScoreSteps), len(pack.DeduplicatedSafeguards))
		}
		if pack.RiskRankAfter == "" || string(pack.InputSnapshot) != completedRow.InputSnapshot {
			t.Fatal("completed pack must keep residual risk and frozen snapshot")
		}
	})

	t.Run("voided keeps historical conclusions", func(t *testing.T) {
		reviewer := util.Actor{UserID: 20, Username: "reviewer", Role: "safety_reviewer", RequestID: "reg-void"}
		voided, err := svc.Void(ctx, completedRow.ID, reviewer)
		if err != nil || voided.EvaluationState != "voided" {
			t.Fatalf("void: state=%s err=%v", voided.EvaluationState, err)
		}
		pack, err := svc.ExportEvidencePack(ctx, completedRow.ID)
		if err != nil {
			t.Fatalf("export voided pack: %v", err)
		}
		if pack.State.Code != "voided" {
			t.Fatalf("state = %s, want voided", pack.State.Code)
		}
		if len(pack.ScoreSteps) == 0 || len(pack.DeduplicatedSafeguards) == 0 || pack.CoverageScore == 0 {
			t.Fatalf("voided pack must retain historical conclusions: steps=%d dedup=%d score=%v",
				len(pack.ScoreSteps), len(pack.DeduplicatedSafeguards), pack.CoverageScore)
		}
	})

	t.Run("corrupt snapshot names the broken business artifact", func(t *testing.T) {
		corrupt := failed
		corrupt.ID = 0
		corrupt.IdempotencyKey = "reg-failed-corrupt-01"
		corrupt.InputSnapshot = "{broken-json"
		if err := evalRepo.Create(ctx, &corrupt); err != nil {
			t.Fatalf("create corrupt failed row: %v", err)
		}
		_, err := svc.ExportEvidencePack(ctx, corrupt.ID)
		var appErr *util.AppError
		if !errors.As(err, &appErr) || appErr.Status != 422 ||
			appErr.Code != util.CodeValidation || !strings.Contains(appErr.Message, "frozen input snapshot") {
			t.Fatalf("corrupt snapshot must return 422 naming the frozen input snapshot rule, got %v", err)
		}
	})
}
