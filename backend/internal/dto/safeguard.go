package dto
import (
	"hazop-safeguard-coverage/backend/internal/model"
	"strings"
	"time"
)
type CreateSafeguardRequest struct {
	Name             string     `json:"name" binding:"required,min=2,max=180"`
	SafeguardType    string     `json:"safeguard_type" binding:"required,oneof=alarm interlock relief procedural containment detection"`
	TargetScenarioID uint       `json:"target_scenario_id" binding:"required"`
	IndependenceKey  string     `json:"independence_key" binding:"required,min=2,max=100"`
	Effectiveness    float64    `json:"effectiveness" binding:"required,gt=0,lte=1"`
	TestIntervalDays int        `json:"test_interval_days" binding:"required,min=1,max=3650"`
	LastVerifiedAt   *time.Time `json:"last_verified_at"`
	EvidenceNote     string     `json:"evidence_note" binding:"required,min=3,max=4000"`
}
func (r *CreateSafeguardRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.SafeguardType = strings.ToLower(strings.TrimSpace(r.SafeguardType))
	r.IndependenceKey = strings.ToUpper(strings.TrimSpace(r.IndependenceKey))
	r.EvidenceNote = strings.TrimSpace(r.EvidenceNote)
}
type UpdateSafeguardRequest struct {
	Name             *string  `json:"name" binding:"omitempty,min=2,max=180"`
	SafeguardType    *string  `json:"safeguard_type" binding:"omitempty,oneof=alarm interlock relief procedural containment detection"`
	IndependenceKey  *string  `json:"independence_key" binding:"omitempty,min=2,max=100"`
	Effectiveness    *float64 `json:"effectiveness" binding:"omitempty,gt=0,lte=1"`
	TestIntervalDays *int     `json:"test_interval_days" binding:"omitempty,min=1,max=3650"`
	EvidenceNote     *string  `json:"evidence_note" binding:"omitempty,min=3,max=4000"`
}
func (r *UpdateSafeguardRequest) Normalize() {
	r.Name = trimPointer(r.Name)
	if r.SafeguardType != nil {
		value := strings.ToLower(strings.TrimSpace(*r.SafeguardType))
		r.SafeguardType = &value
	}
	if r.IndependenceKey != nil {
		value := strings.ToUpper(strings.TrimSpace(*r.IndependenceKey))
		r.IndependenceKey = &value
	}
	r.EvidenceNote = trimPointer(r.EvidenceNote)
}
type VerifySafeguardRequest struct {
	VerifiedAt   time.Time `json:"verified_at" binding:"required"`
	EvidenceNote string    `json:"evidence_note" binding:"required,min=3,max=4000"`
}
type SafeguardActionRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=1000"`
}
type SafeguardQuery struct {
	ScenarioID     uint
	Type           string
	LifecycleState string
	ExpiredOnly    bool
	Search         string
	Page           int
	PageSize       int
}
type SafeguardResponse struct {
	ID                  uint       `json:"id"`
	Name                string     `json:"name"`
	SafeguardType       string     `json:"safeguard_type"`
	TargetScenarioID    uint       `json:"target_scenario_id"`
	IndependenceKey     string     `json:"independence_key"`
	Effectiveness       float64    `json:"effectiveness"`
	TestIntervalDays    int        `json:"test_interval_days"`
	LastVerifiedAt      *time.Time `json:"last_verified_at,omitempty"`
	VerificationExpires *time.Time `json:"verification_expires_at,omitempty"`
	VerificationExpired bool       `json:"verification_expired"`
	LifecycleState      string     `json:"lifecycle_state"`
	EvidenceNote        string     `json:"evidence_note"`
	LastVerificationBy  *uint      `json:"last_verification_by,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
type SafeguardListResponse struct {
	Items []SafeguardResponse `json:"items"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"page_size"`
}
func NewSafeguardResponse(s model.Safeguard, now time.Time) SafeguardResponse {
	expires := s.VerificationExpiresAt()
	expired := expires == nil || now.After(*expires)
	return SafeguardResponse{
		ID: s.ID, Name: s.Name, SafeguardType: s.SafeguardType, TargetScenarioID: s.TargetScenarioID,
		IndependenceKey: s.IndependenceKey, Effectiveness: s.Effectiveness,
		TestIntervalDays: s.TestIntervalDays, LastVerifiedAt: s.LastVerifiedAt,
		VerificationExpires: expires, VerificationExpired: expired,
		LifecycleState: s.LifecycleState, EvidenceNote: s.EvidenceNote,
		LastVerificationBy: s.LastVerificationBy, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}
