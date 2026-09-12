package model
import "time"
type ProcessNode struct {
	ID                uint                `gorm:"primaryKey" json:"id"`
	NodeCode          string              `gorm:"size:40;not null;uniqueIndex" json:"node_code"`
	Name              string              `gorm:"size:160;not null" json:"name"`
	UnitName          string              `gorm:"size:160;not null;index" json:"unit_name"`
	Medium            string              `gorm:"size:120;not null" json:"medium"`
	DesignPressure    float64             `gorm:"not null" json:"design_pressure"`
	DesignTemperature float64             `gorm:"not null" json:"design_temperature"`
	OwnerTeam         string              `gorm:"size:120;not null;index" json:"owner_team"`
	Status            string              `gorm:"size:24;not null;index" json:"status"`
	CreatedAt         time.Time           `gorm:"not null" json:"created_at"`
	UpdatedAt         time.Time           `gorm:"not null" json:"updated_at"`
	Scenarios         []DeviationScenario `gorm:"foreignKey:ProcessNodeID" json:"-"`
}
func (ProcessNode) TableName() string { return "process_nodes" }
func (n ProcessNode) Active() bool { return n.Status == "active" }
type ProcessNodeSummary struct {
	ScenarioCount      int64   `json:"scenario_count"`
	ActiveSafeguards   int64   `json:"active_safeguards"`
	OpenHighRisk       int64   `json:"open_high_risk"`
	LatestCoverage     float64 `json:"latest_coverage"`
	UncoveredPathCount int     `json:"uncovered_path_count"`
}
