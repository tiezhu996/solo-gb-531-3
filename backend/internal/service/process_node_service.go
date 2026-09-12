package service
import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
	"strings"
	"time"
)
type ProcessNodeService interface {
	Create(context.Context, dto.CreateProcessNodeRequest, util.Actor) (dto.ProcessNodeResponse, error)
	Get(context.Context, uint) (dto.ProcessNodeResponse, error)
	List(context.Context, dto.ProcessNodeQuery) (dto.ProcessNodeListResponse, error)
	Update(context.Context, uint, dto.UpdateProcessNodeRequest, util.Actor) (dto.ProcessNodeResponse, error)
	Deactivate(context.Context, uint, util.Actor) (dto.ProcessNodeResponse, error)
}
type processNodeService struct {
	nodes  repository.ProcessNodeRepository
	audits repository.AuditRepository
	now    func() time.Time
}
func NewProcessNodeService(nodes repository.ProcessNodeRepository, audits repository.AuditRepository) ProcessNodeService {
	return &processNodeService{nodes: nodes, audits: audits, now: func() time.Time { return time.Now().UTC() }}
}
func (s *processNodeService) Create(
	ctx context.Context,
	request dto.CreateProcessNodeRequest,
	actor util.Actor,
) (dto.ProcessNodeResponse, error) {
	request.Normalize()
	if _, err := s.nodes.GetByCode(ctx, request.NodeCode); err == nil {
		return dto.ProcessNodeResponse{}, util.Conflict("node_code already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to check node code", err)
	}
	now := s.now()
	node := model.ProcessNode{
		NodeCode: request.NodeCode, Name: request.Name, UnitName: request.UnitName,
		Medium: request.Medium, DesignPressure: request.DesignPressure,
		DesignTemperature: request.DesignTemperature, OwnerTeam: request.OwnerTeam,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.nodes.Create(ctx, &node); err != nil {
		if uniqueViolation(err) {
			return dto.ProcessNodeResponse{}, util.Conflict("node_code already exists")
		}
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to create process node", err)
	}
	if err := s.recordAudit(ctx, actor, "process_node", node.ID, "create", nil, node); err != nil {
		return dto.ProcessNodeResponse{}, err
	}
	return dto.NewProcessNodeResponse(node, model.ProcessNodeSummary{}), nil
}
func (s *processNodeService) Get(ctx context.Context, id uint) (dto.ProcessNodeResponse, error) {
	node, err := s.nodes.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ProcessNodeResponse{}, util.NotFound("process node")
		}
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load process node", err)
	}
	summary, err := s.nodes.Summary(ctx, id)
	if err != nil {
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load node coverage summary", err)
	}
	return dto.NewProcessNodeResponse(node, summary), nil
}
func (s *processNodeService) List(ctx context.Context, query dto.ProcessNodeQuery) (dto.ProcessNodeListResponse, error) {
	nodes, total, err := s.nodes.List(ctx, query)
	if err != nil {
		return dto.ProcessNodeListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list process nodes", err)
	}
	response := dto.ProcessNodeListResponse{
		Items: make([]dto.ProcessNodeResponse, 0, len(nodes)),
		Total: total, Page: query.Page, Size: query.PageSize,
	}
	for _, node := range nodes {
		summary, summaryErr := s.nodes.Summary(ctx, node.ID)
		if summaryErr != nil {
			return dto.ProcessNodeListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load node coverage summary", summaryErr)
		}
		response.Items = append(response.Items, dto.NewProcessNodeResponse(node, summary))
	}
	return response, nil
}
func (s *processNodeService) Update(
	ctx context.Context,
	id uint,
	request dto.UpdateProcessNodeRequest,
	actor util.Actor,
) (dto.ProcessNodeResponse, error) {
	request.Normalize()
	node, err := s.nodes.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ProcessNodeResponse{}, util.NotFound("process node")
		}
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load process node", err)
	}
	before := node
	if request.Name != nil {
		node.Name = *request.Name
	}
	if request.UnitName != nil {
		node.UnitName = *request.UnitName
	}
	if request.Medium != nil {
		node.Medium = *request.Medium
	}
	if request.DesignPressure != nil {
		node.DesignPressure = *request.DesignPressure
	}
	if request.DesignTemperature != nil {
		node.DesignTemperature = *request.DesignTemperature
	}
	if request.OwnerTeam != nil {
		node.OwnerTeam = *request.OwnerTeam
	}
	if request.Status != nil {
		node.Status = *request.Status
	}
	node.UpdatedAt = s.now()
	if err := s.nodes.Update(ctx, &node); err != nil {
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to update process node", err)
	}
	if err := s.recordAudit(ctx, actor, "process_node", node.ID, "update", before, node); err != nil {
		return dto.ProcessNodeResponse{}, err
	}
	summary, err := s.nodes.Summary(ctx, id)
	if err != nil {
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load node coverage summary", err)
	}
	return dto.NewProcessNodeResponse(node, summary), nil
}
func (s *processNodeService) Deactivate(ctx context.Context, id uint, actor util.Actor) (dto.ProcessNodeResponse, error) {
	node, err := s.nodes.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ProcessNodeResponse{}, util.NotFound("process node")
		}
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load process node", err)
	}
	if node.Status == "inactive" {
		return dto.ProcessNodeResponse{}, util.NewError(http.StatusConflict, util.CodeStateTransition, "process node is already inactive")
	}
	before := node
	changed, err := s.nodes.Deactivate(ctx, id)
	if err != nil {
		return dto.ProcessNodeResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to deactivate process node", err)
	}
	if !changed {
		return dto.ProcessNodeResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "process node changed concurrently")
	}
	node.Status = "inactive"
	node.UpdatedAt = s.now()
	if err := s.recordAudit(ctx, actor, "process_node", node.ID, "deactivate", before, node); err != nil {
		return dto.ProcessNodeResponse{}, err
	}
	return s.Get(ctx, id)
}
func (s *processNodeService) recordAudit(
	ctx context.Context,
	actor util.Actor,
	entityType string,
	entityID uint,
	action string,
	before any,
	after any,
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
		ActorRole: actor.Role, EntityType: entityType, EntityID: entityID, Action: action,
		BeforeSnapshot: beforeJSON, AfterSnapshot: afterJSON, CreatedAt: s.now(),
	}
	if err := s.audits.Record(ctx, log); err != nil {
		return util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record write audit", err)
	}
	return nil
}
func snapshotJSON(value any) (string, error) {
	if value == nil {
		return "{}", nil
	}
	return util.CanonicalJSON(value)
}
func uniqueViolation(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}
func serviceError(operation string, err error) error {
	return fmt.Errorf("%s: %w", operation, err)
}
