package model

import "time"

type CoverageEvaluation struct {
	ID                      uint              `gorm:"primaryKey" json:"id"`
	ScenarioID              uint              `gorm:"not null;index" json:"scenario_id"`
	Scenario                DeviationScenario `gorm:"foreignKey:ScenarioID" json:"scenario,omitempty"`
	AlgorithmVersion        string            `gorm:"size:40;not null" json:"algorithm_version"`
	InputSnapshot           string            `gorm:"type:text;not null" json:"input_snapshot"`
	InputHash               string            `gorm:"size:64;not null;index" json:"input_hash"`
	CoverageScore           float64           `gorm:"not null" json:"coverage_score"`
	UncoveredPaths          string            `gorm:"type:text;not null" json:"uncovered_paths"`
	DeduplicatedSafeguards  string            `gorm:"type:text;not null" json:"deduplicated_safeguards"`
	RiskRankBefore          string            `gorm:"size:24;not null" json:"risk_rank_before"`
	RiskRankAfter           string            `gorm:"size:24;not null" json:"risk_rank_after"`
	EvaluationState         string            `gorm:"size:24;not null;index" json:"evaluation_state"`
	Explanation             string            `gorm:"type:text;not null" json:"explanation"`
	EvaluatedBy             uint              `gorm:"not null;index" json:"evaluated_by"`
	EvaluatedByName         string            `gorm:"size:80;not null" json:"evaluated_by_name"`
	EvaluatedAt             time.Time         `gorm:"not null" json:"evaluated_at"`
	CreatedAt               time.Time         `gorm:"not null" json:"created_at"`
	UpdatedAt               time.Time         `gorm:"not null" json:"updated_at"`
	IdempotencyKey          string            `gorm:"size:128;not null;uniqueIndex" json:"idempotency_key"`
	DurationMilliseconds    int64             `gorm:"not null" json:"duration_milliseconds"`
	FailureReason           string            `gorm:"type:text" json:"failure_reason,omitempty"`
	ConfirmedBy             *uint             `json:"confirmed_by,omitempty"`
	ConfirmedAt             *time.Time        `json:"confirmed_at,omitempty"`
	SupersededEvaluationID  *uint             `json:"superseded_evaluation_id,omitempty"`
	DeterminismReplayPassed *bool             `json:"determinism_replay_passed,omitempty"`
}

func (CoverageEvaluation) TableName() string { return "coverage_evaluations" }

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:80;not null;uniqueIndex" json:"username"`
	DisplayName  string    `gorm:"size:120;not null" json:"display_name"`
	PasswordHash string    `gorm:"size:100;not null" json:"-"`
	Role         string    `gorm:"size:40;not null;index" json:"role"`
	Active       bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null" json:"updated_at"`
}

func (User) TableName() string { return "users" }

type AuditLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	RequestID      string    `gorm:"size:80;not null;index" json:"request_id"`
	ActorID        uint      `gorm:"not null;index" json:"actor_id"`
	ActorName      string    `gorm:"size:80;not null" json:"actor_name"`
	ActorRole      string    `gorm:"size:40;not null" json:"actor_role"`
	EntityType     string    `gorm:"size:60;not null;index" json:"entity_type"`
	EntityID       uint      `gorm:"not null;index" json:"entity_id"`
	Action         string    `gorm:"size:80;not null;index" json:"action"`
	BeforeSnapshot string    `gorm:"type:text;not null" json:"before_snapshot"`
	AfterSnapshot  string    `gorm:"type:text;not null" json:"after_snapshot"`
	InputHash      string    `gorm:"size:64;index" json:"input_hash,omitempty"`
	Algorithm      string    `gorm:"size:40" json:"algorithm_version,omitempty"`
	DurationMS     int64     `json:"duration_ms,omitempty"`
	ResultSummary  string    `gorm:"type:text" json:"result_summary,omitempty"`
	CreatedAt      time.Time `gorm:"not null;index" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
