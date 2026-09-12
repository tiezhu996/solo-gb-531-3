package handler
import (
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/service"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
)
type DeviationScenarioHandler struct {
	service service.DeviationScenarioService
}
func NewDeviationScenarioHandler(value service.DeviationScenarioService) *DeviationScenarioHandler {
	return &DeviationScenarioHandler{service: value}
}
func (h *DeviationScenarioHandler) List(c *gin.Context) {
	nodeID, ok := optionalUint(c, "process_node_id")
	if !ok {
		return
	}
	page, size := util.Pagination(c)
	result, err := h.service.List(c.Request.Context(), dto.DeviationScenarioQuery{
		ProcessNodeID: nodeID, Guideword: c.Query("guideword"), State: c.Query("state"),
		Risk: c.Query("risk"), Search: c.Query("search"), Page: page, PageSize: size,
	})
	respond(c, http.StatusOK, result, err)
}
func (h *DeviationScenarioHandler) Get(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}
func (h *DeviationScenarioHandler) Create(c *gin.Context) {
	var request dto.CreateDeviationScenarioRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Create(c.Request.Context(), request, mustActor(c))
	respond(c, http.StatusCreated, result, err)
}
func (h *DeviationScenarioHandler) Update(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.UpdateDeviationScenarioRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Update(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
func (h *DeviationScenarioHandler) Transition(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.TransitionScenarioRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Transition(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
