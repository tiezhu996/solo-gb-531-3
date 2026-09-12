package service

import (
	"context"
	"encoding/json"
	"errors"
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
	failed := model.CoverageEvaluation{
		ScenarioID: scenario.ID, AlgorithmVersion: algorithm.Version,
		InputSnapshot: snapshotJSON, InputHash: util.HashString(snapshotJSON),
		UncoveredPaths: "[]", DeduplicatedSafeguards: "[]", Explanation: "{}",
		RiskRankBefore: "high", RiskRankAfter: "high", EvaluationState: "failed",
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
	if pack.State.ReadableSummary == "" {
		t.Fatal("failed evaluation needs a readable status explanation")
	}
	if len(pack.ScoreSteps) != 0 || len(pack.UncoveredPaths) != 0 {
		t.Fatalf("failed pack must not fabricate scoring results: steps=%d uncovered=%d", len(pack.ScoreSteps), len(pack.UncoveredPaths))
	}
	if json.Valid(pack.InputSnapshot) == false {
		t.Fatal("failed pack must still carry the frozen input snapshot")
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
	corrupt := model.CoverageEvaluation{
		ScenarioID: scenario.ID, AlgorithmVersion: algorithm.Version,
		InputSnapshot: "{not-json", InputHash: "broken",
		UncoveredPaths: "[]", DeduplicatedSafeguards: "[]", Explanation: "{}",
		RiskRankBefore: "high", RiskRankAfter: "high", EvaluationState: "failed",
		EvaluatedBy: 7, EvaluatedByName: "author", EvaluatedAt: now,
		CreatedAt: now, UpdatedAt: now, IdempotencyKey: "evidence-key-corrupt-001",
	}
	if err := repo.Create(context.Background(), &corrupt); err != nil {
		t.Fatalf("create corrupt evaluation: %v", err)
	}
	_, err = svc.ExportEvidencePack(context.Background(), corrupt.ID)
	var invalid *util.AppError
	if !errors.As(err, &invalid) || invalid.Status != 422 {
		t.Fatalf("unreadable snapshot must surface a clear 422 export failure, got %v", err)
	}
}
