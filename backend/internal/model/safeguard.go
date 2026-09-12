package model

import "time"

type Safeguard struct {
	ID                 uint              `gorm:"primaryKey" json:"id"`
	Name               string            `gorm:"size:180;not null" json:"name"`
	SafeguardType      string            `gorm:"size:40;not null;index" json:"safeguard_type"`
	TargetScenarioID   uint              `gorm:"not null;index:idx_safeguard_target_independence" json:"target_scenario_id"`
	TargetScenario     DeviationScenario `gorm:"foreignKey:TargetScenarioID" json:"target_scenario,omitempty"`
	IndependenceKey    string            `gorm:"size:100;not null;index:idx_safeguard_target_independence" json:"independence_key"`
	Effectiveness      float64           `gorm:"not null" json:"effectiveness"`
	TestIntervalDays   int               `gorm:"not null" json:"test_interval_days"`
	LastVerifiedAt     *time.Time        `json:"last_verified_at,omitempty"`
	LifecycleState     string            `gorm:"size:24;not null;index" json:"lifecycle_state"`
	EvidenceNote       string            `gorm:"type:text;not null" json:"evidence_note"`
	CreatedAt          time.Time         `gorm:"not null" json:"created_at"`
	UpdatedAt          time.Time         `gorm:"not null" json:"updated_at"`
	LastVerificationBy *uint             `json:"last_verification_by,omitempty"`
}

func (Safeguard) TableName() string { return "safeguards" }

func (s Safeguard) VerificationExpiresAt() *time.Time {
	if s.LastVerifiedAt == nil || s.TestIntervalDays <= 0 {
		return nil
	}
	expires := s.LastVerifiedAt.AddDate(0, 0, s.TestIntervalDays)
	return &expires
}

func (s Safeguard) IsEffectiveAt(at time.Time) bool {
	if s.LifecycleState != "active" || s.Effectiveness <= 0 || s.LastVerifiedAt == nil {
		return false
	}
	expires := s.VerificationExpiresAt()
	return expires != nil && !at.After(*expires)
}
