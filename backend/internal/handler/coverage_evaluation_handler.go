package handler
import (
	"context"
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/service"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
)
type CoverageEvaluationHandler struct {
	service service.CoverageEvaluationService
}
func NewCoverageEvaluationHandler(value service.CoverageEvaluationService) *CoverageEvaluationHandler {
	return &CoverageEvaluationHandler{service: value}
}
func (h *CoverageEvaluationHandler) List(c *gin.Context) {
	scenarioID, ok := optionalUint(c, "scenario_id")
	if !ok {
		return
	}
	evaluatorID, ok := optionalUint(c, "evaluated_by")
	if !ok {
		return
	}
	page, size := util.Pagination(c)
	result, err := h.service.List(c.Request.Context(), dto.CoverageEvaluationQuery{
		ScenarioID: scenarioID, State: c.Query("state"), Evaluator: evaluatorID, Page: page, PageSize: size,
	})
	respond(c, http.StatusOK, result, err)
}
func (h *CoverageEvaluationHandler) Get(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}
func (h *CoverageEvaluationHandler) Run(c *gin.Context) {
	var request dto.RunCoverageEvaluationRequest
	if !bindJSON(c, &request) {
		return
	}
	result, duplicate, err := h.service.Run(
		c.Request.Context(), request, c.GetHeader("Idempotency-Key"), mustActor(c),
	)
	status := http.StatusCreated
	if duplicate {
		status = http.StatusOK
	}
	respond(c, status, result, err)
}
func (h *CoverageEvaluationHandler) Confirm(c *gin.Context) { h.stateAction(c, h.service.Confirm) }
func (h *CoverageEvaluationHandler) Void(c *gin.Context)    { h.stateAction(c, h.service.Void) }
func (h *CoverageEvaluationHandler) Replay(c *gin.Context)  { h.stateAction(c, h.service.Replay) }
func (h *CoverageEvaluationHandler) stateAction(c *gin.Context, operation func(
	context.Context, uint, util.Actor,
) (dto.CoverageEvaluationResponse, error)) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, err := operation(c.Request.Context(), id, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
func (h *CoverageEvaluationHandler) Compare(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	otherID, err := util.ParseUintParam(c, "other_id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, err := h.service.Compare(c.Request.Context(), id, otherID)
	respond(c, http.StatusOK, result, err)
}
