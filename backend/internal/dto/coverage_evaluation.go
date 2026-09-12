package dto
import (
	"encoding/json"
	"hazop-safeguard-coverage/backend/internal/model"
	"time"
)
type RunCoverageEvaluationRequest struct {
	ScenarioID uint `json:"scenario_id" binding:"required"`
}
type CoverageEvaluationQuery struct {
	ScenarioID uint
	State      string
	Evaluator  uint
	Page       int
	PageSize   int
}
type CoveragePathResponse struct {
	PathID             string   `json:"path_id"`
	NodeCode           string   `json:"node_code"`
	Cause              string   `json:"cause"`
	Consequence        string   `json:"consequence"`
	SafeguardIDs       []uint   `json:"safeguard_ids"`
	IndependenceKeys   []string `json:"independence_keys"`
	CombinedProtection float64  `json:"combined_protection"`
	Covered            bool     `json:"covered"`
	Reason             string   `json:"reason"`
}
type ScoreStepResponse struct {
	Step         int     `json:"step"`
	Rule         string  `json:"rule"`
	Input        string  `json:"input"`
	Contribution float64 `json:"contribution"`
	RunningScore float64 `json:"running_score"`
	Explanation  string  `json:"explanation"`
}
type DeduplicatedSafeguardResponse struct {
	IndependenceKey string `json:"independence_key"`
	KeptID          uint   `json:"kept_id"`
	IgnoredIDs      []uint `json:"ignored_ids"`
	Reason          string `json:"reason"`
}
type EvaluationExplanation struct {
	Summary       string                          `json:"summary"`
	Paths         []CoveragePathResponse          `json:"paths"`
	ScoreSteps    []ScoreStepResponse             `json:"score_steps"`
	Deduplicated  []DeduplicatedSafeguardResponse `json:"deduplicated_safeguards"`
	BoundaryNote  string                          `json:"boundary_note"`
	ReferenceTime time.Time                       `json:"reference_time"`
}
type CoverageEvaluationResponse struct {
	ID                     uint                            `json:"id"`
	ScenarioID             uint                            `json:"scenario_id"`
	AlgorithmVersion       string                          `json:"algorithm_version"`
	InputSnapshot          json.RawMessage                 `json:"input_snapshot"`
	InputHash              string                          `json:"input_hash"`
	CoverageScore          float64                         `json:"coverage_score"`
	UncoveredPaths         []CoveragePathResponse          `json:"uncovered_paths"`
	DeduplicatedSafeguards []DeduplicatedSafeguardResponse `json:"deduplicated_safeguards"`
	RiskRankBefore         string                          `json:"risk_rank_before"`
	RiskRankAfter          string                          `json:"risk_rank_after"`
	EvaluationState        string                          `json:"evaluation_state"`
	Explanation            EvaluationExplanation           `json:"explanation"`
	EvaluatedBy            uint                            `json:"evaluated_by"`
	EvaluatedByName        string                          `json:"evaluated_by_name"`
	EvaluatedAt            time.Time                       `json:"evaluated_at"`
	IdempotencyKey         string                          `json:"idempotency_key"`
	DurationMilliseconds   int64                           `json:"duration_milliseconds"`
	FailureReason          string                          `json:"failure_reason,omitempty"`
	ConfirmedBy            *uint                           `json:"confirmed_by,omitempty"`
	ConfirmedAt            *time.Time                      `json:"confirmed_at,omitempty"`
	ReplayPassed           *bool                           `json:"determinism_replay_passed,omitempty"`
	CreatedAt              time.Time                       `json:"created_at"`
}
type CoverageEvaluationListResponse struct {
	Items []CoverageEvaluationResponse `json:"items"`
	Total int64                        `json:"total"`
	Page  int                          `json:"page"`
	Size  int                          `json:"page_size"`
}
type EvaluationComparisonResponse struct {
	BaseID             uint    `json:"base_id"`
	ComparedID         uint    `json:"compared_id"`
	ScoreDelta         float64 `json:"score_delta"`
	UncoveredPathDelta int     `json:"uncovered_path_delta"`
	RiskRankChanged    bool    `json:"risk_rank_changed"`
	InputChanged       bool    `json:"input_changed"`
}
func NewCoverageEvaluationResponse(e model.CoverageEvaluation) CoverageEvaluationResponse {
	response := CoverageEvaluationResponse{
		ID: e.ID, ScenarioID: e.ScenarioID, AlgorithmVersion: e.AlgorithmVersion,
		InputSnapshot: rawJSON(e.InputSnapshot), InputHash: e.InputHash, CoverageScore: e.CoverageScore,
		RiskRankBefore: e.RiskRankBefore, RiskRankAfter: e.RiskRankAfter,
		EvaluationState: e.EvaluationState, EvaluatedBy: e.EvaluatedBy,
		EvaluatedByName: e.EvaluatedByName, EvaluatedAt: e.EvaluatedAt,
		IdempotencyKey: e.IdempotencyKey, DurationMilliseconds: e.DurationMilliseconds,
		FailureReason: e.FailureReason, ConfirmedBy: e.ConfirmedBy, ConfirmedAt: e.ConfirmedAt,
		ReplayPassed: e.DeterminismReplayPassed, CreatedAt: e.CreatedAt,
	}
	_ = json.Unmarshal([]byte(e.UncoveredPaths), &response.UncoveredPaths)
	_ = json.Unmarshal([]byte(e.DeduplicatedSafeguards), &response.DeduplicatedSafeguards)
	_ = json.Unmarshal([]byte(e.Explanation), &response.Explanation)
	return response
}
func rawJSON(value string) json.RawMessage {
	if !json.Valid([]byte(value)) {
		return json.RawMessage("null")
	}
	return json.RawMessage(value)
}
