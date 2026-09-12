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
type SafeguardRepository interface {
	Create(context.Context, *model.Safeguard) error
	GetByID(context.Context, uint) (model.Safeguard, error)
	List(context.Context, dto.SafeguardQuery, time.Time) ([]model.Safeguard, int64, error)
	ListByScenario(context.Context, uint) ([]model.Safeguard, error)
	Update(context.Context, *model.Safeguard) error
	SetLifecycle(context.Context, uint, []string, string, map[string]any) (bool, error)
}
type safeguardRepository struct{ db *gorm.DB }
func NewSafeguardRepository(db *gorm.DB) SafeguardRepository {
	return &safeguardRepository{db: db}
}
func (r *safeguardRepository) Create(ctx context.Context, safeguard *model.Safeguard) error {
	if err := r.db.WithContext(ctx).Create(safeguard).Error; err != nil {
		return fmt.Errorf("create safeguard: %w", err)
	}
	return nil
}
func (r *safeguardRepository) GetByID(ctx context.Context, id uint) (model.Safeguard, error) {
	var safeguard model.Safeguard
	if err := r.db.WithContext(ctx).Preload("TargetScenario").First(&safeguard, id).Error; err != nil {
		return model.Safeguard{}, fmt.Errorf("find safeguard %d: %w", id, err)
	}
	return safeguard, nil
}
func (r *safeguardRepository) List(ctx context.Context, query dto.SafeguardQuery, now time.Time) ([]model.Safeguard, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.Safeguard{})
	if query.ScenarioID != 0 {
		base = base.Where("target_scenario_id = ?", query.ScenarioID)
	}
	if query.Type != "" {
		base = base.Where("safeguard_type = ?", query.Type)
	}
	if query.LifecycleState != "" {
		base = base.Where("lifecycle_state = ?", query.LifecycleState)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		base = base.Where("LOWER(name) LIKE ? OR LOWER(independence_key) LIKE ? OR LOWER(evidence_note) LIKE ?", pattern, pattern, pattern)
	}
	if query.ExpiredOnly {
		expiry := "datetime(last_verified_at, '+' || test_interval_days || ' days')"
		if r.db.Dialector.Name() == "postgres" {
			expiry = "last_verified_at + (test_interval_days * INTERVAL '1 day')"
		}
		base = base.Where("last_verified_at IS NULL OR test_interval_days <= 0 OR "+expiry+" < ?", now)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count safeguards: %w", err)
	}
	var safeguards []model.Safeguard
	offset := (query.Page - 1) * query.PageSize
	if err := base.Preload("TargetScenario").Order("updated_at DESC, id DESC").
		Limit(query.PageSize).Offset(offset).Find(&safeguards).Error; err != nil {
		return nil, 0, fmt.Errorf("list safeguards: %w", err)
	}
	return safeguards, total, nil
}
func (r *safeguardRepository) ListByScenario(ctx context.Context, scenarioID uint) ([]model.Safeguard, error) {
	var safeguards []model.Safeguard
	if err := r.db.WithContext(ctx).Where("target_scenario_id = ?", scenarioID).
		Order("independence_key ASC, id ASC").Find(&safeguards).Error; err != nil {
		return nil, fmt.Errorf("list safeguards for scenario %d: %w", scenarioID, err)
	}
	return safeguards, nil
}
func (r *safeguardRepository) Update(ctx context.Context, safeguard *model.Safeguard) error {
	result := r.db.WithContext(ctx).Model(&model.Safeguard{}).Where("id = ?", safeguard.ID).Updates(map[string]any{
		"name": safeguard.Name, "safeguard_type": safeguard.SafeguardType,
		"independence_key": safeguard.IndependenceKey, "effectiveness": safeguard.Effectiveness,
		"test_interval_days": safeguard.TestIntervalDays, "evidence_note": safeguard.EvidenceNote,
		"updated_at": safeguard.UpdatedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("update safeguard %d: %w", safeguard.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *safeguardRepository) SetLifecycle(
	ctx context.Context,
	id uint,
	fromStates []string,
	toState string,
	updates map[string]any,
) (bool, error) {
	values := make(map[string]any, len(updates)+2)
	for key, value := range updates {
		values[key] = value
	}
	values["lifecycle_state"] = toState
	values["updated_at"] = time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&model.Safeguard{}).
		Where("id = ? AND lifecycle_state IN ?", id, fromStates).Updates(values)
	if result.Error != nil {
		return false, fmt.Errorf("change safeguard lifecycle %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}
