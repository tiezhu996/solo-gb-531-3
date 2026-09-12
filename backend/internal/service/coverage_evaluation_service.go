package service
import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"hazop-safeguard-coverage/backend/internal/algorithm"
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
	"strings"
	"time"
)
type CoverageEvaluationService interface {
	Run(context.Context, dto.RunCoverageEvaluationRequest, string, util.Actor) (dto.CoverageEvaluationResponse, bool, error)
	Get(context.Context, uint) (dto.CoverageEvaluationResponse, error)
	List(context.Context, dto.CoverageEvaluationQuery) (dto.CoverageEvaluationListResponse, error)
	Confirm(context.Context, uint, util.Actor) (dto.CoverageEvaluationResponse, error)
	Void(context.Context, uint, util.Actor) (dto.CoverageEvaluationResponse, error)
	Replay(context.Context, uint, util.Actor) (dto.CoverageEvaluationResponse, error)
	Compare(context.Context, uint, uint) (dto.EvaluationComparisonResponse, error)
}
type coverageEvaluationService struct {
	evaluations repository.CoverageEvaluationRepository
	scenarios   repository.DeviationScenarioRepository
	nodes       repository.ProcessNodeRepository
	safeguards  repository.SafeguardRepository
	audits      repository.AuditRepository
	evaluator   *algorithm.Evaluator
	now         func() time.Time
}
func NewCoverageEvaluationService(
	evaluations repository.CoverageEvaluationRepository,
	scenarios repository.DeviationScenarioRepository,
	nodes repository.ProcessNodeRepository,
	safeguards repository.SafeguardRepository,
	audits repository.AuditRepository,
	evaluator *algorithm.Evaluator,
) CoverageEvaluationService {
	return &coverageEvaluationService{
		evaluations: evaluations, scenarios: scenarios, nodes: nodes,
		safeguards: safeguards, audits: audits, evaluator: evaluator,
		now: func() time.Time { return time.Now().UTC() },
	}
}
func (s *coverageEvaluationService) Run(
	ctx context.Context,
	request dto.RunCoverageEvaluationRequest,
	idempotencyKey string,
	actor util.Actor,
) (dto.CoverageEvaluationResponse, bool, error) {
	key := strings.TrimSpace(idempotencyKey)
	if len(key) < 8 || len(key) > 128 {
		return dto.CoverageEvaluationResponse{}, false, util.NewError(
			http.StatusBadRequest, util.CodeIdempotency,
			"Idempotency-Key header must contain 8 to 128 characters",
		)
	}
	existing, err := s.evaluations.FindByIdempotencyKey(ctx, key)
	if err == nil {
		return dto.NewCoverageEvaluationResponse(existing), true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to check idempotency key", err)
	}
	scenario, err := s.scenarios.GetByID(ctx, request.ScenarioID, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CoverageEvaluationResponse{}, false, util.NotFound("deviation scenario")
		}
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load deviation scenario", err)
	}
	node, err := s.nodes.GetByID(ctx, scenario.ProcessNodeID)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load process node", err)
	}
	safeguards, err := s.safeguards.ListByScenario(ctx, scenario.ID)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load scenario safeguards", err)
	}
	referenceTime := s.now().Truncate(time.Second)
	snapshot := algorithm.NewSnapshot(node, scenario, safeguards, referenceTime)
	snapshotJSON, err := util.CanonicalJSON(snapshot)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to freeze evaluation input", err)
	}
	evaluation := model.CoverageEvaluation{
		ScenarioID: scenario.ID, AlgorithmVersion: algorithm.Version,
		InputSnapshot: snapshotJSON, InputHash: util.HashString(snapshotJSON),
		UncoveredPaths: "[]", DeduplicatedSafeguards: "[]", Explanation: "{}",
		RiskRankBefore: dto.RiskRank(scenario.InitialRisk()), RiskRankAfter: dto.RiskRank(scenario.InitialRisk()),
		EvaluationState: string(constants.CoverageQueued),
		EvaluatedBy:     actor.UserID, EvaluatedByName: actor.Username,
		EvaluatedAt: referenceTime, IdempotencyKey: key,
		CreatedAt: referenceTime, UpdatedAt: referenceTime,
	}
	if err := s.evaluations.Create(ctx, &evaluation); err != nil {
		if uniqueViolation(err) {
			existing, findErr := s.evaluations.FindByIdempotencyKey(ctx, key)
			if findErr == nil {
				return dto.NewCoverageEvaluationResponse(existing), true, nil
			}
		}
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to queue coverage evaluation", err)
	}
	changed, err := s.evaluations.Transition(
		ctx, evaluation.ID, []string{string(constants.CoverageQueued)},
		string(constants.CoverageRunning), map[string]any{},
	)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to start coverage evaluation", err)
	}
	if !changed {
		return dto.CoverageEvaluationResponse{}, false, util.NewError(http.StatusConflict, util.CodeConflict, "evaluation state changed before it could start")
	}
	evaluation.EvaluationState = string(constants.CoverageRunning)
	started := time.Now()
	result, evaluationErr := s.evaluator.Evaluate(snapshot)
	duration := time.Since(started).Milliseconds()
	if evaluationErr != nil {
		_, transitionErr := s.evaluations.Transition(
			ctx, evaluation.ID, []string{string(constants.CoverageRunning)},
			string(constants.CoverageFailed), map[string]any{
				"failure_reason":        util.CompactText(evaluationErr.Error(), 2000),
				"duration_milliseconds": duration,
			},
		)
		if transitionErr != nil {
			return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "evaluation failed and failure state could not be stored", transitionErr)
		}
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "coverage evaluation failed", evaluationErr)
	}
	updates := map[string]any{
		"input_snapshot": result.SnapshotJSON, "input_hash": result.InputHash,
		"coverage_score": result.CoverageScore, "uncovered_paths": result.UncoveredJSON,
		"deduplicated_safeguards": result.DeduplicatedJSON, "explanation": result.ExplanationJSON,
		"risk_rank_before": result.RiskBefore, "risk_rank_after": result.RiskAfter,
		"duration_milliseconds": duration, "failure_reason": "",
	}
	changed, err = s.evaluations.Complete(ctx, evaluation.ID, string(constants.CoverageRunning), updates)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to store coverage result", err)
	}
	if !changed {
		return dto.CoverageEvaluationResponse{}, false, util.NewError(http.StatusConflict, util.CodeConflict, "evaluation state changed while the result was being stored")
	}
	evaluation, err = s.evaluations.GetByID(ctx, evaluation.ID)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload coverage evaluation", err)
	}
	summary, _ := util.CanonicalJSON(map[string]any{
		"coverage_score":   evaluation.CoverageScore,
		"risk_rank_before": evaluation.RiskRankBefore,
		"risk_rank_after":  evaluation.RiskRankAfter,
		"uncovered_paths":  len(result.Explanation.Paths) - countCovered(result.Explanation.Paths),
	})
	log := model.AuditLog{
		RequestID: actor.RequestID, ActorID: actor.UserID, ActorName: actor.Username,
		ActorRole: actor.Role, EntityType: "coverage_evaluation", EntityID: evaluation.ID,
		Action: "run", BeforeSnapshot: "{}", AfterSnapshot: result.ExplanationJSON,
		InputHash: result.InputHash, Algorithm: algorithm.Version, DurationMS: duration,
		ResultSummary: summary, CreatedAt: s.now(),
	}
	if err := s.audits.Record(ctx, log); err != nil {
		return dto.CoverageEvaluationResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record algorithm audit", err)
	}
	return dto.NewCoverageEvaluationResponse(evaluation), false, nil
}
func (s *coverageEvaluationService) Get(ctx context.Context, id uint) (dto.CoverageEvaluationResponse, error) {
	evaluation, err := s.evaluations.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CoverageEvaluationResponse{}, util.NotFound("coverage evaluation")
		}
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load coverage evaluation", err)
	}
	return dto.NewCoverageEvaluationResponse(evaluation), nil
}
func (s *coverageEvaluationService) List(
	ctx context.Context,
	query dto.CoverageEvaluationQuery,
) (dto.CoverageEvaluationListResponse, error) {
	evaluations, total, err := s.evaluations.List(ctx, query)
	if err != nil {
		return dto.CoverageEvaluationListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list coverage evaluations", err)
	}
	response := dto.CoverageEvaluationListResponse{
		Items: make([]dto.CoverageEvaluationResponse, 0, len(evaluations)),
		Total: total, Page: query.Page, Size: query.PageSize,
	}
	for _, evaluation := range evaluations {
		response.Items = append(response.Items, dto.NewCoverageEvaluationResponse(evaluation))
	}
	return response, nil
}
func (s *coverageEvaluationService) Confirm(
	ctx context.Context,
	id uint,
	actor util.Actor,
) (dto.CoverageEvaluationResponse, error) {
	evaluation, err := s.evaluations.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CoverageEvaluationResponse{}, util.NotFound("coverage evaluation")
		}
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load coverage evaluation", err)
	}
	scenario, err := s.scenarios.GetByID(ctx, evaluation.ScenarioID, false)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load evaluated scenario", err)
	}
	if scenario.CreatedBy == actor.UserID {
		return dto.CoverageEvaluationResponse{}, util.NewError(http.StatusConflict, util.CodeReviewerConflict, "evaluation confirmer must differ from scenario author")
	}
	now := s.now()
	changed, err := s.evaluations.Transition(
		ctx, id, []string{string(constants.CoverageCompleted)}, string(constants.CoverageConfirmed),
		map[string]any{"confirmed_by": actor.UserID, "confirmed_at": now},
	)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to confirm evaluation", err)
	}
	if !changed {
		return dto.CoverageEvaluationResponse{}, util.NewError(http.StatusConflict, util.CodeStateTransition, "only a completed evaluation may be confirmed")
	}
	after, err := s.evaluations.GetByID(ctx, id)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload evaluation", err)
	}
	if err := s.recordStateAudit(ctx, actor, "confirm", evaluation, after); err != nil {
		return dto.CoverageEvaluationResponse{}, err
	}
	return dto.NewCoverageEvaluationResponse(after), nil
}
func (s *coverageEvaluationService) Void(
	ctx context.Context,
	id uint,
	actor util.Actor,
) (dto.CoverageEvaluationResponse, error) {
	evaluation, err := s.evaluations.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CoverageEvaluationResponse{}, util.NotFound("coverage evaluation")
		}
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load coverage evaluation", err)
	}
	changed, err := s.evaluations.Transition(
		ctx, id,
		[]string{string(constants.CoverageCompleted), string(constants.CoverageFailed), string(constants.CoverageConfirmed)},
		string(constants.CoverageVoided), map[string]any{},
	)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to void evaluation", err)
	}
	if !changed {
		return dto.CoverageEvaluationResponse{}, util.NewError(http.StatusConflict, util.CodeStateTransition, "evaluation cannot be voided from its current state")
	}
	after, err := s.evaluations.GetByID(ctx, id)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload evaluation", err)
	}
	if err := s.recordStateAudit(ctx, actor, "void", evaluation, after); err != nil {
		return dto.CoverageEvaluationResponse{}, err
	}
	return dto.NewCoverageEvaluationResponse(after), nil
}
func (s *coverageEvaluationService) Replay(
	ctx context.Context,
	id uint,
	actor util.Actor,
) (dto.CoverageEvaluationResponse, error) {
	evaluation, err := s.evaluations.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CoverageEvaluationResponse{}, util.NotFound("coverage evaluation")
		}
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load coverage evaluation", err)
	}
	passed, _, err := s.evaluator.Replay(evaluation.InputSnapshot, evaluation.InputHash, evaluation.CoverageScore)
	if err != nil {
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "evaluation could not be replayed", err)
	}
	if err := s.evaluations.SetReplayResult(ctx, id, passed); err != nil {
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to store replay status", err)
	}
	log := model.AuditLog{
		RequestID: actor.RequestID, ActorID: actor.UserID, ActorName: actor.Username,
		ActorRole: actor.Role, EntityType: "coverage_evaluation", EntityID: id,
		Action: "replay", BeforeSnapshot: "{}", AfterSnapshot: fmt.Sprintf("{\"passed\":%t}", passed),
		InputHash: evaluation.InputHash, Algorithm: evaluation.AlgorithmVersion,
		ResultSummary: fmt.Sprintf("deterministic replay passed=%t", passed), CreatedAt: s.now(),
	}
	if err := s.audits.Record(ctx, log); err != nil {
		return dto.CoverageEvaluationResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record replay audit", err)
	}
	if !passed {
		return dto.CoverageEvaluationResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "deterministic replay did not match the stored result")
	}
	return s.Get(ctx, id)
}
func (s *coverageEvaluationService) Compare(
	ctx context.Context,
	id uint,
	otherID uint,
) (dto.EvaluationComparisonResponse, error) {
	base, err := s.evaluations.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.EvaluationComparisonResponse{}, util.NotFound("base coverage evaluation")
		}
		return dto.EvaluationComparisonResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load base evaluation", err)
	}
	other, err := s.evaluations.GetByID(ctx, otherID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.EvaluationComparisonResponse{}, util.NotFound("compared coverage evaluation")
		}
		return dto.EvaluationComparisonResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load compared evaluation", err)
	}
	if base.ScenarioID != other.ScenarioID {
		return dto.EvaluationComparisonResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "only evaluations of the same scenario can be compared")
	}
	baseResponse := dto.NewCoverageEvaluationResponse(base)
	otherResponse := dto.NewCoverageEvaluationResponse(other)
	return dto.EvaluationComparisonResponse{
		BaseID: base.ID, ComparedID: other.ID,
		ScoreDelta:         other.CoverageScore - base.CoverageScore,
		UncoveredPathDelta: len(otherResponse.UncoveredPaths) - len(baseResponse.UncoveredPaths),
		RiskRankChanged:    base.RiskRankAfter != other.RiskRankAfter,
		InputChanged:       base.InputHash != other.InputHash,
	}, nil
}
func (s *coverageEvaluationService) recordStateAudit(
	ctx context.Context,
	actor util.Actor,
	action string,
	before model.CoverageEvaluation,
	after model.CoverageEvaluation,
) error {
	beforeJSON, err := snapshotJSON(before)
	if err != nil {
		return util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to serialize audit snapshot", err)
	}
	afterJSON, err := snapshotJSON(after)
	if err != nil {
		return util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to serialize audit snapshot", err)
	}
	log := model.AuditLog{
		RequestID: actor.RequestID, ActorID: actor.UserID, ActorName: actor.Username,
		ActorRole: actor.Role, EntityType: "coverage_evaluation", EntityID: before.ID,
		Action: action, BeforeSnapshot: beforeJSON, AfterSnapshot: afterJSON,
		InputHash: before.InputHash, Algorithm: before.AlgorithmVersion, CreatedAt: s.now(),
	}
	if err := s.audits.Record(ctx, log); err != nil {
		return util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record write audit", err)
	}
	return nil
}
func countCovered(paths []dto.CoveragePathResponse) int {
	count := 0
	for _, path := range paths {
		if path.Covered {
			count++
		}
	}
	return count
}
