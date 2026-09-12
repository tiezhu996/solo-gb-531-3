package dto
import (
	"hazop-safeguard-coverage/backend/internal/model"
	"strings"
	"time"
)
type CreateProcessNodeRequest struct {
	NodeCode          string  `json:"node_code" binding:"required,min=2,max=40"`
	Name              string  `json:"name" binding:"required,min=2,max=160"`
	UnitName          string  `json:"unit_name" binding:"required,min=2,max=160"`
	Medium            string  `json:"medium" binding:"required,min=1,max=120"`
	DesignPressure    float64 `json:"design_pressure" binding:"required,gte=0,lte=10000"`
	DesignTemperature float64 `json:"design_temperature" binding:"gte=-273.15,lte=5000"`
	OwnerTeam         string  `json:"owner_team" binding:"required,min=2,max=120"`
}
func (r *CreateProcessNodeRequest) Normalize() {
	r.NodeCode = strings.ToUpper(strings.TrimSpace(r.NodeCode))
	r.Name = strings.TrimSpace(r.Name)
	r.UnitName = strings.TrimSpace(r.UnitName)
	r.Medium = strings.TrimSpace(r.Medium)
	r.OwnerTeam = strings.TrimSpace(r.OwnerTeam)
}
type UpdateProcessNodeRequest struct {
	Name              *string  `json:"name" binding:"omitempty,min=2,max=160"`
	UnitName          *string  `json:"unit_name" binding:"omitempty,min=2,max=160"`
	Medium            *string  `json:"medium" binding:"omitempty,min=1,max=120"`
	DesignPressure    *float64 `json:"design_pressure" binding:"omitempty,gte=0,lte=10000"`
	DesignTemperature *float64 `json:"design_temperature" binding:"omitempty,gte=-273.15,lte=5000"`
	OwnerTeam         *string  `json:"owner_team" binding:"omitempty,min=2,max=120"`
	Status            *string  `json:"status" binding:"omitempty,oneof=active inactive"`
}
func (r *UpdateProcessNodeRequest) Normalize() {
	r.Name = trimPointer(r.Name)
	r.UnitName = trimPointer(r.UnitName)
	r.Medium = trimPointer(r.Medium)
	r.OwnerTeam = trimPointer(r.OwnerTeam)
}
type ProcessNodeQuery struct {
	Search    string
	UnitName  string
	OwnerTeam string
	Status    string
	Page      int
	PageSize  int
}
type ProcessNodeResponse struct {
	ID                uint                     `json:"id"`
	NodeCode          string                   `json:"node_code"`
	Name              string                   `json:"name"`
	UnitName          string                   `json:"unit_name"`
	Medium            string                   `json:"medium"`
	DesignPressure    float64                  `json:"design_pressure"`
	DesignTemperature float64                  `json:"design_temperature"`
	OwnerTeam         string                   `json:"owner_team"`
	Status            string                   `json:"status"`
	Summary           model.ProcessNodeSummary `json:"coverage_summary"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
}
type ProcessNodeListResponse struct {
	Items []ProcessNodeResponse `json:"items"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"page_size"`
}
func NewProcessNodeResponse(node model.ProcessNode, summary model.ProcessNodeSummary) ProcessNodeResponse {
	return ProcessNodeResponse{
		ID: node.ID, NodeCode: node.NodeCode, Name: node.Name, UnitName: node.UnitName,
		Medium: node.Medium, DesignPressure: node.DesignPressure, DesignTemperature: node.DesignTemperature,
		OwnerTeam: node.OwnerTeam, Status: node.Status, Summary: summary,
		CreatedAt: node.CreatedAt, UpdatedAt: node.UpdatedAt,
	}
}
func trimPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
