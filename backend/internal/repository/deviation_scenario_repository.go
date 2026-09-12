package repository
import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"strings"
	"time"
)
type DeviationScenarioRepository interface {
	Create(context.Context, *model.DeviationScenario) error
	GetByID(context.Context, uint, bool) (model.DeviationScenario, error)
	List(context.Context, dto.DeviationScenarioQuery) ([]model.DeviationScenario, int64, error)
	UpdateWithVersion(context.Context, *model.DeviationScenario, int) (bool, error)
	Transition(context.Context, uint, string, string, int, *uint, string) (bool, error)
	CountByNode(context.Context, uint) (int64, error)
}
type deviationScenarioRepository struct{ db *gorm.DB }
func NewDeviationScenarioRepository(db *gorm.DB) DeviationScenarioRepository {
	return &deviationScenarioRepository{db: db}
}
func (r *deviationScenarioRepository) Create(ctx context.Context, scenario *model.DeviationScenario) error {
	if err := r.db.WithContext(ctx).Create(scenario).Error; err != nil {
		return fmt.Errorf("create deviation scenario: %w", err)
	}
	return nil
}
func (r *deviationScenarioRepository) GetByID(ctx context.Context, id uint, preload bool) (model.DeviationScenario, error) {
	query := r.db.WithContext(ctx)
	if preload {
		query = query.Preload("ProcessNode").Preload("Safeguards", func(db *gorm.DB) *gorm.DB {
			return db.Order("independence_key ASC, id ASC")
		})
	}
	var scenario model.DeviationScenario
	if err := query.First(&scenario, id).Error; err != nil {
		return model.DeviationScenario{}, fmt.Errorf("find deviation scenario %d: %w", id, err)
	}
	return scenario, nil
}
func (r *deviationScenarioRepository) List(ctx context.Context, query dto.DeviationScenarioQuery) ([]model.DeviationScenario, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.DeviationScenario{})
	if query.ProcessNodeID != 0 {
		base = base.Where("process_node_id = ?", query.ProcessNodeID)
	}
	if query.Guideword != "" {
		base = base.Where("guideword = ?", query.Guideword)
	}
	if query.State != "" {
		base = base.Where("scenario_state = ?", query.State)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		base = base.Where("LOWER(parameter) LIKE ? OR LOWER(cause) LIKE ? OR LOWER(consequence) LIKE ?", pattern, pattern, pattern)
	}
	switch query.Risk {
	case "critical":
		base = base.Where("likelihood * severity >= 20")
	case "high":
		base = base.Where("likelihood * severity BETWEEN 12 AND 19")
	case "medium":
		base = base.Where("likelihood * severity BETWEEN 6 AND 11")
	case "low":
		base = base.Where("likelihood * severity <= 5")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count deviation scenarios: %w", err)
	}
	var scenarios []model.DeviationScenario
	offset := (query.Page - 1) * query.PageSize
	err := base.Preload("ProcessNode").Preload("Safeguards").
		Order("updated_at DESC, id DESC").Limit(query.PageSize).Offset(offset).Find(&scenarios).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list deviation scenarios: %w", err)
	}
	return scenarios, total, nil
}
func (r *deviationScenarioRepository) UpdateWithVersion(ctx context.Context, scenario *model.DeviationScenario, expectedVersion int) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.DeviationScenario{}).
		Where("id = ? AND version = ? AND scenario_state IN ?", scenario.ID, expectedVersion, []string{"draft", "rework"}).
		Updates(map[string]any{
			"guideword": scenario.Guideword, "parameter": scenario.Parameter,
			"cause": scenario.Cause, "consequence": scenario.Consequence,
			"likelihood": scenario.Likelihood, "severity": scenario.Severity,
			"version": expectedVersion + 1, "updated_at": scenario.UpdatedAt,
		})
	if result.Error != nil {
		return false, fmt.Errorf("update deviation scenario %d: %w", scenario.ID, result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *deviationScenarioRepository) Transition(
	ctx context.Context,
	id uint,
	fromState string,
	toState string,
	expectedVersion int,
	reviewerID *uint,
	reviewerName string,
) (bool, error) {
	updates := map[string]any{
		"scenario_state": toState,
		"version":        expectedVersion + 1,
		"updated_at":     time.Now().UTC(),
	}
	if reviewerID != nil {
		updates["reviewed_by"] = *reviewerID
		updates["reviewed_by_name"] = reviewerName
	}
	result := r.db.WithContext(ctx).Model(&model.DeviationScenario{}).
		Where("id = ? AND scenario_state = ? AND version = ?", id, fromState, expectedVersion).
		Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("transition deviation scenario %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *deviationScenarioRepository) CountByNode(ctx context.Context, nodeID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.DeviationScenario{}).
		Where("process_node_id = ?", nodeID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count scenarios for node %d: %w", nodeID, err)
	}
	return count, nil
}
