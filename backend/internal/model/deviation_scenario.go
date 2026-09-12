package model

import "time"

type DeviationScenario struct {
	ID             uint        `gorm:"primaryKey" json:"id"`
	ProcessNodeID  uint        `gorm:"not null;index" json:"process_node_id"`
	ProcessNode    ProcessNode `gorm:"foreignKey:ProcessNodeID" json:"process_node,omitempty"`
	Guideword      string      `gorm:"size:24;not null;index" json:"guideword"`
	Parameter      string      `gorm:"size:120;not null" json:"parameter"`
	Cause          string      `gorm:"type:text;not null" json:"cause"`
	Consequence    string      `gorm:"type:text;not null" json:"consequence"`
	Likelihood     int         `gorm:"not null" json:"likelihood"`
	Severity       int         `gorm:"not null" json:"severity"`
	ScenarioState  string      `gorm:"size:24;not null;index" json:"scenario_state"`
	Version        int         `gorm:"not null;default:1" json:"version"`
	CreatedBy      uint        `gorm:"not null;index" json:"created_by"`
	CreatedByName  string      `gorm:"size:80;not null" json:"created_by_name"`
	ReviewedBy     *uint       `gorm:"index" json:"reviewed_by,omitempty"`
	ReviewedByName string      `gorm:"size:80" json:"reviewed_by_name,omitempty"`
	CreatedAt      time.Time   `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time   `gorm:"not null" json:"updated_at"`
	Safeguards     []Safeguard `gorm:"foreignKey:TargetScenarioID" json:"safeguards,omitempty"`
}

func (DeviationScenario) TableName() string { return "deviation_scenarios" }

func (s DeviationScenario) InitialRisk() int { return s.Likelihood * s.Severity }

func (s DeviationScenario) ReviewerSeparated(userID uint) bool {
	return s.CreatedBy != userID
}
