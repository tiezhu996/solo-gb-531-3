package dto
import (
	"hazop-safeguard-coverage/backend/internal/model"
	"strings"
	"time"
)
type CreateDeviationScenarioRequest struct {
	ProcessNodeID uint   `json:"process_node_id" binding:"required"`
	Guideword     string `json:"guideword" binding:"required,oneof=no more less reverse other"`
	Parameter     string `json:"parameter" binding:"required,min=1,max=120"`
	Cause         string `json:"cause" binding:"required,min=3,max=4000"`
	Consequence   string `json:"consequence" binding:"required,min=3,max=4000"`
	Likelihood    int    `json:"likelihood" binding:"required,min=1,max=5"`
	Severity      int    `json:"severity" binding:"required,min=1,max=5"`
}
func (r *CreateDeviationScenarioRequest) Normalize() {
	r.Guideword = strings.ToLower(strings.TrimSpace(r.Guideword))
	r.Parameter = strings.TrimSpace(r.Parameter)
	r.Cause = strings.TrimSpace(r.Cause)
	r.Consequence = strings.TrimSpace(r.Consequence)
}
type UpdateDeviationScenarioRequest struct {
	Guideword   *string `json:"guideword" binding:"omitempty,oneof=no more less reverse other"`
	Parameter   *string `json:"parameter" binding:"omitempty,min=1,max=120"`
	Cause       *string `json:"cause" binding:"omitempty,min=3,max=4000"`
	Consequence *string `json:"consequence" binding:"omitempty,min=3,max=4000"`
	Likelihood  *int    `json:"likelihood" binding:"omitempty,min=1,max=5"`
	Severity    *int    `json:"severity" binding:"omitempty,min=1,max=5"`
	Version     int     `json:"version" binding:"required,min=1"`
}
func (r *UpdateDeviationScenarioRequest) Normalize() {
	if r.Guideword != nil {
		value := strings.ToLower(strings.TrimSpace(*r.Guideword))
		r.Guideword = &value
	}
	r.Parameter = trimPointer(r.Parameter)
	r.Cause = trimPointer(r.Cause)
	r.Consequence = trimPointer(r.Consequence)
}
type TransitionScenarioRequest struct {
	ToState string `json:"to_state" binding:"required,oneof=analyzed verified accepted rework"`
	Comment string `json:"comment" binding:"omitempty,max=1000"`
	Version int    `json:"version" binding:"required,min=1"`
}
type DeviationScenarioQuery struct {
	ProcessNodeID uint
	Guideword     string
	State         string
	Risk          string
	Search        string
	Page          int
	PageSize      int
}
type StateEvent struct {
	State     string    `json:"state"`
	ActorName string    `json:"actor_name"`
	Comment   string    `json:"comment,omitempty"`
	At        time.Time `json:"at"`
}
type DeviationScenarioResponse struct {
	ID             uint                 `json:"id"`
	ProcessNodeID  uint                 `json:"process_node_id"`
	ProcessNode    *ProcessNodeResponse `json:"process_node,omitempty"`
	Guideword      string               `json:"guideword"`
	Parameter      string               `json:"parameter"`
	Cause          string               `json:"cause"`
	Consequence    string               `json:"consequence"`
	Likelihood     int                  `json:"likelihood"`
	Severity       int                  `json:"severity"`
	RiskScore      int                  `json:"risk_score"`
	RiskRank       string               `json:"risk_rank"`
	ScenarioState  string               `json:"scenario_state"`
	Version        int                  `json:"version"`
	CreatedBy      uint                 `json:"created_by"`
	CreatedByName  string               `json:"created_by_name"`
	ReviewedBy     *uint                `json:"reviewed_by,omitempty"`
	ReviewedByName string               `json:"reviewed_by_name,omitempty"`
	Safeguards     []SafeguardResponse  `json:"safeguards,omitempty"`
	Timeline       []StateEvent         `json:"timeline,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}
type DeviationScenarioListResponse struct {
	Items []DeviationScenarioResponse `json:"items"`
	Total int64                       `json:"total"`
	Page  int                         `json:"page"`
	Size  int                         `json:"page_size"`
}
func NewDeviationScenarioResponse(s model.DeviationScenario) DeviationScenarioResponse {
	response := DeviationScenarioResponse{
		ID: s.ID, ProcessNodeID: s.ProcessNodeID, Guideword: s.Guideword, Parameter: s.Parameter,
		Cause: s.Cause, Consequence: s.Consequence, Likelihood: s.Likelihood, Severity: s.Severity,
		RiskScore: s.InitialRisk(), RiskRank: RiskRank(s.InitialRisk()), ScenarioState: s.ScenarioState,
		Version: s.Version, CreatedBy: s.CreatedBy, CreatedByName: s.CreatedByName,
		ReviewedBy: s.ReviewedBy, ReviewedByName: s.ReviewedByName,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	for _, item := range s.Safeguards {
		response.Safeguards = append(response.Safeguards, NewSafeguardResponse(item, time.Now().UTC()))
	}
	if s.ProcessNode.ID != 0 {
		node := NewProcessNodeResponse(s.ProcessNode, model.ProcessNodeSummary{})
		response.ProcessNode = &node
	}
	return response
}
func RiskRank(score int) string {
	switch {
	case score >= 20:
		return "critical"
	case score >= 12:
		return "high"
	case score >= 6:
		return "medium"
	default:
		return "low"
	}
}
