package repository
import (
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"strings"
	"time"
)
type CoverageEvaluationRepository interface {
	Create(context.Context, *model.CoverageEvaluation) error
	GetByID(context.Context, uint) (model.CoverageEvaluation, error)
	FindByIdempotencyKey(context.Context, string) (model.CoverageEvaluation, error)
	List(context.Context, dto.CoverageEvaluationQuery) ([]model.CoverageEvaluation, int64, error)
	Transition(context.Context, uint, []string, string, map[string]any) (bool, error)
	Complete(context.Context, uint, string, map[string]any) (bool, error)
	SetReplayResult(context.Context, uint, bool) error
}
type AuditRepository interface {
	Record(context.Context, model.AuditLog) error
	List(context.Context, AuditQuery) ([]model.AuditLog, int64, error)
}
type UserRepository interface {
	FindByUsername(context.Context, string) (model.User, error)
	GetByID(context.Context, uint) (model.User, error)
}
type AuditQuery struct {
	EntityType string
	ActorID    uint
	RequestID  string
	Action     string
	From       *time.Time
	To         *time.Time
	Page       int
	PageSize   int
}
type coverageEvaluationRepository struct{ db *gorm.DB }
type auditRepository struct{ db *gorm.DB }
type userRepository struct{ db *gorm.DB }
func NewCoverageEvaluationRepository(db *gorm.DB) CoverageEvaluationRepository {
	return &coverageEvaluationRepository{db: db}
}
func NewAuditRepository(db *gorm.DB) AuditRepository { return &auditRepository{db: db} }
func NewUserRepository(db *gorm.DB) UserRepository   { return &userRepository{db: db} }
func (r *coverageEvaluationRepository) Create(ctx context.Context, evaluation *model.CoverageEvaluation) error {
	if err := r.db.WithContext(ctx).Create(evaluation).Error; err != nil {
		return fmt.Errorf("create coverage evaluation: %w", err)
	}
	return nil
}
func (r *coverageEvaluationRepository) GetByID(ctx context.Context, id uint) (model.CoverageEvaluation, error) {
	var evaluation model.CoverageEvaluation
	if err := r.db.WithContext(ctx).Preload("Scenario").First(&evaluation, id).Error; err != nil {
		return model.CoverageEvaluation{}, fmt.Errorf("find coverage evaluation %d: %w", id, err)
	}
	return evaluation, nil
}
func (r *coverageEvaluationRepository) FindByIdempotencyKey(ctx context.Context, key string) (model.CoverageEvaluation, error) {
	var evaluation model.CoverageEvaluation
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&evaluation).Error; err != nil {
		return model.CoverageEvaluation{}, fmt.Errorf("find evaluation by idempotency key: %w", err)
	}
	return evaluation, nil
}
func (r *coverageEvaluationRepository) List(ctx context.Context, query dto.CoverageEvaluationQuery) ([]model.CoverageEvaluation, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.CoverageEvaluation{})
	if query.ScenarioID != 0 {
		base = base.Where("scenario_id = ?", query.ScenarioID)
	}
	if query.State != "" {
		base = base.Where("evaluation_state = ?", query.State)
	}
	if query.Evaluator != 0 {
		base = base.Where("evaluated_by = ?", query.Evaluator)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count coverage evaluations: %w", err)
	}
	var evaluations []model.CoverageEvaluation
	offset := (query.Page - 1) * query.PageSize
	if err := base.Order("evaluated_at DESC, id DESC").Limit(query.PageSize).Offset(offset).
		Find(&evaluations).Error; err != nil {
		return nil, 0, fmt.Errorf("list coverage evaluations: %w", err)
	}
	return evaluations, total, nil
}
func (r *coverageEvaluationRepository) Transition(
	ctx context.Context,
	id uint,
	from []string,
	to string,
	updates map[string]any,
) (bool, error) {
	values := make(map[string]any, len(updates)+2)
	for key, value := range updates {
		values[key] = value
	}
	values["evaluation_state"] = to
	values["updated_at"] = time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&model.CoverageEvaluation{}).
		Where("id = ? AND evaluation_state IN ?", id, from).Updates(values)
	if result.Error != nil {
		return false, fmt.Errorf("transition coverage evaluation %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *coverageEvaluationRepository) Complete(ctx context.Context, id uint, expectedState string, updates map[string]any) (bool, error) {
	values := make(map[string]any, len(updates)+2)
	for key, value := range updates {
		values[key] = value
	}
	values["evaluation_state"] = "completed"
	values["updated_at"] = time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&model.CoverageEvaluation{}).
		Where("id = ? AND evaluation_state = ?", id, expectedState).Updates(values)
	if result.Error != nil {
		return false, fmt.Errorf("complete coverage evaluation %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}
func (r *coverageEvaluationRepository) SetReplayResult(ctx context.Context, id uint, passed bool) error {
	result := r.db.WithContext(ctx).Model(&model.CoverageEvaluation{}).
		Where("id = ?", id).Update("determinism_replay_passed", passed)
	if result.Error != nil {
		return fmt.Errorf("store replay result for evaluation %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *auditRepository) Record(ctx context.Context, log model.AuditLog) error {
	if log.BeforeSnapshot == "" {
		log.BeforeSnapshot = "{}"
	}
	if log.AfterSnapshot == "" {
		log.AfterSnapshot = "{}"
	}
	if err := r.db.WithContext(ctx).Create(&log).Error; err != nil {
		return fmt.Errorf("record audit log: %w", err)
	}
	return nil
}
func (r *auditRepository) List(ctx context.Context, query AuditQuery) ([]model.AuditLog, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.AuditLog{})
	if query.EntityType != "" {
		base = base.Where("entity_type = ?", query.EntityType)
	}
	if query.ActorID != 0 {
		base = base.Where("actor_id = ?", query.ActorID)
	}
	if requestID := strings.TrimSpace(query.RequestID); requestID != "" {
		base = base.Where("request_id = ?", requestID)
	}
	if query.Action != "" {
		base = base.Where("action = ?", query.Action)
	}
	if query.From != nil {
		base = base.Where("created_at >= ?", *query.From)
	}
	if query.To != nil {
		base = base.Where("created_at <= ?", *query.To)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	var logs []model.AuditLog
	offset := (query.Page - 1) * query.PageSize
	if err := base.Order("created_at DESC, id DESC").Limit(query.PageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, total, nil
}
func (r *userRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("LOWER(username) = ?", strings.ToLower(strings.TrimSpace(username))).
		First(&user).Error; err != nil {
		return model.User{}, fmt.Errorf("find user by username: %w", err)
	}
	return user, nil
}
func (r *userRepository) GetByID(ctx context.Context, id uint) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return model.User{}, fmt.Errorf("find user %d: %w", id, err)
	}
	return user, nil
}
func decodeJSON(value string, target any) error {
	if value == "" {
		value = "[]"
	}
	if err := json.Unmarshal([]byte(value), target); err != nil {
		return fmt.Errorf("decode stored JSON: %w", err)
	}
	return nil
}
