package repository
import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"strings"
)
type ProcessNodeRepository interface {
	Create(context.Context, *model.ProcessNode) error
	GetByID(context.Context, uint) (model.ProcessNode, error)
	GetByCode(context.Context, string) (model.ProcessNode, error)
	List(context.Context, dto.ProcessNodeQuery) ([]model.ProcessNode, int64, error)
	Update(context.Context, *model.ProcessNode) error
	Deactivate(context.Context, uint) (bool, error)
	Summary(context.Context, uint) (model.ProcessNodeSummary, error)
}
type processNodeRepository struct{ db *gorm.DB }
func NewProcessNodeRepository(db *gorm.DB) ProcessNodeRepository {
	return &processNodeRepository{db: db}
}
func (r *processNodeRepository) Create(ctx context.Context, node *model.ProcessNode) error {
	if err := r.db.WithContext(ctx).Create(node).Error; err != nil {
		return fmt.Errorf("create process node: %w", err)
	}
	return nil
}
func (r *processNodeRepository) GetByID(ctx context.Context, id uint) (model.ProcessNode, error) {
	var node model.ProcessNode
	err := r.db.WithContext(ctx).First(&node, id).Error
	if err != nil {
		return model.ProcessNode{}, fmt.Errorf("find process node %d: %w", id, err)
	}
	return node, nil
}
func (r *processNodeRepository) GetByCode(ctx context.Context, code string) (model.ProcessNode, error) {
	var node model.ProcessNode
	err := r.db.WithContext(ctx).Where("node_code = ?", code).First(&node).Error
	if err != nil {
		return model.ProcessNode{}, fmt.Errorf("find process node by code: %w", err)
	}
	return node, nil
}
func (r *processNodeRepository) List(ctx context.Context, query dto.ProcessNodeQuery) ([]model.ProcessNode, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.ProcessNode{})
	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		base = base.Where("LOWER(node_code) LIKE ? OR LOWER(name) LIKE ? OR LOWER(medium) LIKE ?", pattern, pattern, pattern)
	}
	if query.UnitName != "" {
		base = base.Where("unit_name = ?", query.UnitName)
	}
	if query.OwnerTeam != "" {
		base = base.Where("owner_team = ?", query.OwnerTeam)
	}
	if query.Status != "" {
		base = base.Where("status = ?", query.Status)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count process nodes: %w", err)
	}
	var nodes []model.ProcessNode
	offset := (query.Page - 1) * query.PageSize
	if err := base.Order("node_code ASC").Limit(query.PageSize).Offset(offset).Find(&nodes).Error; err != nil {
		return nil, 0, fmt.Errorf("list process nodes: %w", err)
	}
	return nodes, total, nil
}
func (r *processNodeRepository) Update(ctx context.Context, node *model.ProcessNode) error {
	result := r.db.WithContext(ctx).Model(&model.ProcessNode{}).Where("id = ?", node.ID).Updates(map[string]any{
		"name": node.Name, "unit_name": node.UnitName, "medium": node.Medium,
		"design_pressure": node.DesignPressure, "design_temperature": node.DesignTemperature,
		"owner_team": node.OwnerTeam, "status": node.Status, "updated_at": node.UpdatedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("update process node %d: %w", node.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *processNodeRepository) Deactivate(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.ProcessNode{}).
		Where("id = ? AND status = ?", id, "active").
		Updates(map[string]any{"status": "inactive", "updated_at": gorm.Expr("CURRENT_TIMESTAMP")})
	if result.Error != nil {
		return false, fmt.Errorf("deactivate process node %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *processNodeRepository) Summary(ctx context.Context, id uint) (model.ProcessNodeSummary, error) {
	var summary model.ProcessNodeSummary
	if err := r.db.WithContext(ctx).Model(&model.DeviationScenario{}).
		Where("process_node_id = ?", id).Count(&summary.ScenarioCount).Error; err != nil {
		return summary, fmt.Errorf("count node scenarios: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&model.DeviationScenario{}).
		Where("process_node_id = ? AND likelihood * severity >= ?", id, 12).
		Where("scenario_state <> ?", "accepted").Count(&summary.OpenHighRisk).Error; err != nil {
		return summary, fmt.Errorf("count high-risk scenarios: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&model.Safeguard{}).
		Joins("JOIN deviation_scenarios ON deviation_scenarios.id = safeguards.target_scenario_id").
		Where("deviation_scenarios.process_node_id = ? AND safeguards.lifecycle_state = ?", id, "active").
		Count(&summary.ActiveSafeguards).Error; err != nil {
		return summary, fmt.Errorf("count active safeguards: %w", err)
	}
	var latest model.CoverageEvaluation
	err := r.db.WithContext(ctx).Model(&model.CoverageEvaluation{}).
		Joins("JOIN deviation_scenarios ON deviation_scenarios.id = coverage_evaluations.scenario_id").
		Where("deviation_scenarios.process_node_id = ? AND coverage_evaluations.evaluation_state IN ?", id, []string{"completed", "confirmed"}).
		Order("coverage_evaluations.evaluated_at DESC").First(&latest).Error
	if err == nil {
		summary.LatestCoverage = latest.CoverageScore
		var paths []any
		if decodeJSON(latest.UncoveredPaths, &paths) == nil {
			summary.UncoveredPathCount = len(paths)
		}
	} else if err != gorm.ErrRecordNotFound {
		return summary, fmt.Errorf("load latest node evaluation: %w", err)
	}
	return summary, nil
}
