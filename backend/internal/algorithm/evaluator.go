package algorithm
import (
	"encoding/json"
	"fmt"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/util"
)
const SafetyBoundary = "Offline decision support only. Results cannot replace a licensed process-safety professional's judgment and cannot issue equipment control commands."
type EvaluationResult struct {
	SnapshotJSON     string
	InputHash        string
	CoverageScore    float64
	UncoveredJSON    string
	DeduplicatedJSON string
	ExplanationJSON  string
	RiskBefore       string
	RiskAfter        string
	Explanation      dto.EvaluationExplanation
}
type Evaluator struct{}
func NewEvaluator() *Evaluator { return &Evaluator{} }
func (e *Evaluator) Evaluate(snapshot Snapshot) (EvaluationResult, error) {
	if snapshot.AlgorithmVersion != Version {
		return EvaluationResult{}, fmt.Errorf("algorithm version mismatch: got %q want %q", snapshot.AlgorithmVersion, Version)
	}
	if snapshot.Node.ID == 0 || snapshot.Scenario.ID == 0 {
		return EvaluationResult{}, fmt.Errorf("snapshot requires persisted node and scenario")
	}
	if snapshot.ReferenceTime.IsZero() {
		return EvaluationResult{}, fmt.Errorf("snapshot reference time is required")
	}
	snapshotJSON, err := util.CanonicalJSON(snapshot)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("serialize evaluation snapshot: %w", err)
	}
	graph := BuildGraph(snapshot)
	independence := ResolveIndependence(snapshot.Safeguards, snapshot.ReferenceTime)
	score := CalculateScore(graph, snapshot.Scenario, independence.Retained, independence.Rejected)
	explanation := dto.EvaluationExplanation{
		Summary: fmt.Sprintf("%d cause-to-consequence paths evaluated; %d paths are below the protection threshold.", len(score.Paths), len(score.UncoveredPaths)),
		Paths:   score.Paths, ScoreSteps: score.Steps, Deduplicated: independence.Deduplicated,
		BoundaryNote: SafetyBoundary, ReferenceTime: snapshot.ReferenceTime,
	}
	uncoveredJSON, err := util.CanonicalJSON(score.UncoveredPaths)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("serialize uncovered paths: %w", err)
	}
	deduplicatedJSON, err := util.CanonicalJSON(independence.Deduplicated)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("serialize deduplicated safeguards: %w", err)
	}
	explanationJSON, err := util.CanonicalJSON(explanation)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("serialize scoring explanation: %w", err)
	}
	return EvaluationResult{
		SnapshotJSON: snapshotJSON, InputHash: util.HashString(snapshotJSON), CoverageScore: score.CoverageScore,
		UncoveredJSON: uncoveredJSON, DeduplicatedJSON: deduplicatedJSON,
		ExplanationJSON: explanationJSON, RiskBefore: score.RiskBefore, RiskAfter: score.RiskAfter,
		Explanation: explanation,
	}, nil
}
func (e *Evaluator) Replay(snapshotJSON string, expectedHash string, expectedScore float64) (bool, EvaluationResult, error) {
	if util.HashString(snapshotJSON) != expectedHash {
		return false, EvaluationResult{}, fmt.Errorf("stored snapshot hash does not match stored input hash")
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
		return false, EvaluationResult{}, fmt.Errorf("decode frozen snapshot: %w", err)
	}
	result, err := e.Evaluate(snapshot)
	if err != nil {
		return false, EvaluationResult{}, fmt.Errorf("replay evaluation: %w", err)
	}
	passed := result.InputHash == expectedHash && result.CoverageScore == expectedScore
	return passed, result, nil
}
