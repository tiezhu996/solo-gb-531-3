package service
import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
	"time"
)
type DeviationScenarioService interface {
	Create(context.Context, dto.CreateDeviationScenarioRequest, util.Actor) (dto.DeviationScenarioResponse, error)
	Get(context.Context, uint) (dto.DeviationScenarioResponse, error)
	List(context.Context, dto.DeviationScenarioQuery) (dto.DeviationScenarioListResponse, error)
	Update(context.Context, uint, dto.UpdateDeviationScenarioRequest, util.Actor) (dto.DeviationScenarioResponse, error)
	Transition(context.Context, uint, dto.TransitionScenarioRequest, util.Actor) (dto.DeviationScenarioResponse, error)
}
type deviationScenarioService struct {
	scenarios repository.DeviationScenarioRepository
	nodes     repository.ProcessNodeRepository
	audits    repository.AuditRepository
	now       func() time.Time
}
func NewDeviationScenarioService(
	scenarios repository.DeviationScenarioRepository,
	nodes repository.ProcessNodeRepository,
	audits repository.AuditRepository,
) DeviationScenarioService {
	return &deviationScenarioService{
		scenarios: scenarios, nodes: nodes, audits: audits,
		now: func() time.Time { return time.Now().UTC() },
	}
}
func (s *deviationScenarioService) Create(
	ctx context.Context,
	request dto.CreateDeviationScenarioRequest,
	actor util.Actor,
) (dto.DeviationScenarioResponse, error) {
	request.Normalize()
	if !constants.DeviationGuideword(request.Guideword).Valid() {
		return dto.DeviationScenarioResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "guideword is not supported")
	}
	node, err := s.nodes.GetByID(ctx, request.ProcessNodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.DeviationScenarioResponse{}, util.NotFound("process node")
		}
		return dto.DeviationScenarioResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load process node", err)
	}
	if !node.Active() {
		return dto.DeviationScenarioResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "cannot add a scenario to an inactive process node")
	}
	now := s.now()
	scenario := model.DeviationScenario{
		ProcessNodeID: request.ProcessNodeID, Guideword: request.Guideword,
		Parameter: request.Parameter, Cause: request.Cause, Consequence: request.Consequence,
		Likelihood: request.Likelihood, Severity: request.Severity,
		ScenarioState: string(constants.ScenarioDraft), Version: 1,
		CreatedBy: actor.UserID, CreatedByName: actor.Username,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.scenarios.Create(ctx, &scenario); err != nil {
		return dto.DeviationScenarioResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to create deviation scenario", err)
	}
	if err := s.recordAudit(ctx, actor, scenario.ID, "create", nil, scenario, ""); err != nil {
		return dto.DeviationScenarioResponse{}, err
	}
	scenario.ProcessNode = node
	return dto.NewDeviationScenarioResponse(scenario), nil
}
func (s *deviationScenarioService) Get(ctx context.Context, id uint) (dto.DeviationScenarioResponse, error) {
	scenario, err := s.scenarios.GetByID(ctx, id, true)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.DeviationScenarioResponse{}, util.NotFound("deviation scenario")
		}
		return dto.DeviationScenarioResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load deviation scenario", err)
	}
	response := dto.NewDeviationScenarioResponse(scenario)
	logs, _, auditErr := s.audits.List(ctx, repository.AuditQuery{
		EntityType: "deviation_scenario", Page: 1, PageSize: 100,
	})
	if auditErr == nil {
		for index := len(logs) - 1; index >= 0; index-- {
			log := logs[index]
			if log.EntityID != scenario.ID {
				continue
			}
			if log.Action != "transition" && log.Action != "create" {
				continue
			}
			state := scenario.ScenarioState
			var projected map[string]any
			if json.Unmarshal([]byte(log.AfterSnapshot), &projected) == nil {
				if value, ok := projected["scenario_state"].(string); ok {
					state = value
				}
			}
			response.Timeline = append(response.Timeline, dto.StateEvent{
				State: state, ActorName: log.ActorName, Comment: log.ResultSummary, At: log.CreatedAt,
			})
		}
	}
	return response, nil
}
func (s *deviationScenarioService) List(
	ctx context.Context,
	query dto.DeviationScenarioQuery,
) (dto.DeviationScenarioListResponse, error) {
	scenarios, total, err := s.scenarios.List(ctx, query)
	if err != nil {
		return dto.DeviationScenarioListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list deviation scenarios", err)
	}
	response := dto.DeviationScenarioListResponse{
		Items: make([]dto.DeviationScenarioResponse, 0, len(scenarios)),
		Total: total, Page: query.Page, Size: query.PageSize,
	}
	for _, scenario := range scenarios {
		response.Items = append(response.Items, dto.NewDeviationScenarioResponse(scenario))
	}
	return response, nil
}
func (s *deviationScenarioService) Update(
	ctx context.Context,
	id uint,
	request dto.UpdateDeviationScenarioRequest,
	actor util.Actor,
) (dto.DeviationScenarioResponse, error) {
	request.Normalize()
	scenario, err := s.scenarios.GetByID(ctx, id, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.DeviationScenarioResponse{}, util.NotFound("deviation scenario")
		}
		return dto.DeviationScenarioResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load deviation scenario", err)
	}
	if scenario.ScenarioState != string(constants.ScenarioDraft) && scenario.ScenarioState != string(constants.ScenarioRework) {
		return dto.DeviationScenarioResponse{}, util.NewError(http.StatusConflict, util.CodeStateTransition, "only draft or rework scenarios may be edited")
	}
	before := scenario
	if request.Guideword != nil {
		if !constants.DeviationGuideword(*request.Guideword).Valid() {
			return dto.DeviationScenarioResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "guideword is not supported")
		}
		scenario.Guideword = *request.Guideword
	}
	if request.Parameter != nil {
		scenario.Parameter = *request.Parameter
	}
	if request.Cause != nil {
		scenario.Cause = *request.Cause
	}
	if request.Consequence != nil {
		scenario.Consequence = *request.Consequence
	}
	if request.Likelihood != nil {
		scenario.Likelihood = *request.Likelihood
	}
	if request.Severity != nil {
		scenario.Severity = *request.Severity
	}
	scenario.UpdatedAt = s.now()
	changed, err := s.scenarios.UpdateWithVersion(ctx, &scenario, request.Version)
	if err != nil {
		return dto.DeviationScenarioResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to update deviation scenario", err)
	}
	if !changed {
		return dto.DeviationScenarioResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "scenario version or state changed concurrently")
	}
	scenario.Version = request.Version + 1
	if err := s.recordAudit(ctx, actor, scenario.ID, "update", before, scenario, ""); err != nil {
		return dto.DeviationScenarioResponse{}, err
	}
	return s.Get(ctx, id)
}
func (s *deviationScenarioService) Transition(
	ctx context.Context,
	id uint,
	request dto.TransitionScenarioRequest,
	actor util.Actor,
) (dto.DeviationScenarioResponse, error) {
	scenario, err := s.scenarios.GetByID(ctx, id, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.DeviationScenarioResponse{}, util.NotFound("deviation scenario")
		}
		return dto.DeviationScenarioResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load deviation scenario", err)
	}
	from := constants.ScenarioState(scenario.ScenarioState)
	to := constants.ScenarioState(request.ToState)
	role := constants.Role(actor.Role)
	if to == constants.ScenarioAnalyzed && role != constants.RoleAdmin && role != constants.RoleProcessEngineer {
		return dto.DeviationScenarioResponse{}, util.NewError(http.StatusForbidden, util.CodeForbidden, "only a process engineer may submit analysis")
	}
	if (to == constants.ScenarioVerified || to == constants.ScenarioAccepted || to == constants.ScenarioRework) &&
		role != constants.RoleAdmin && role != constants.RoleSafetyReviewer {
		return dto.DeviationScenarioResponse{}, util.NewError(http.StatusForbidden, util.CodeForbidden, "only a safety reviewer may perform this transition")
	}
	if !constants.CanTransitionScenario(from, to) {
		return dto.DeviationScenarioResponse{}, util.NewError(
			http.StatusConflict, util.CodeStateTransition,
			"illegal scenario transition from "+scenario.ScenarioState+" to "+request.ToState,
		)
	}
	reviewAction := to == constants.ScenarioVerified || to == constants.ScenarioAccepted
	if reviewAction && !scenario.ReviewerSeparated(actor.UserID) {
		return dto.DeviationScenarioResponse{}, util.NewError(
			http.StatusConflict, util.CodeReviewerConflict,
			"scenario reviewer must differ from scenario author",
		)
	}
	var reviewerID *uint
	reviewerName := ""
	if reviewAction {
		value := actor.UserID
		reviewerID = &value
		reviewerName = actor.Username
	}
	before := scenario
	changed, err := s.scenarios.Transition(
		ctx, scenario.ID, scenario.ScenarioState, request.ToState,
		request.Version, reviewerID, reviewerName,
	)
	if err != nil {
		return dto.DeviationScenarioResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to transition deviation scenario", err)
	}
	if !changed {
		return dto.DeviationScenarioResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "scenario version or state changed concurrently")
	}
	scenario.ScenarioState = request.ToState
	scenario.Version = request.Version + 1
	scenario.UpdatedAt = s.now()
	if reviewerID != nil {
		scenario.ReviewedBy = reviewerID
		scenario.ReviewedByName = reviewerName
	}
	if err := s.recordAudit(ctx, actor, scenario.ID, "transition", before, scenario, request.Comment); err != nil {
		return dto.DeviationScenarioResponse{}, err
	}
	return s.Get(ctx, id)
}
func (s *deviationScenarioService) recordAudit(
	ctx context.Context,
	actor util.Actor,
	entityID uint,
	action string,
	before any,
	after any,
	comment string,
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
		ActorRole: actor.Role, EntityType: "deviation_scenario", EntityID: entityID, Action: action,
		BeforeSnapshot: beforeJSON, AfterSnapshot: afterJSON,
		ResultSummary: util.CompactText(comment, 1000), CreatedAt: s.now(),
	}
	if err := s.audits.Record(ctx, log); err != nil {
		return util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record write audit", err)
	}
	return nil
}
