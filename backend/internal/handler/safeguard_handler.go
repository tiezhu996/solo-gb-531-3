package handler
import (
	"context"
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/service"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
	"strconv"
)
type SafeguardHandler struct{ service service.SafeguardService }
func NewSafeguardHandler(value service.SafeguardService) *SafeguardHandler {
	return &SafeguardHandler{service: value}
}
func (h *SafeguardHandler) List(c *gin.Context) {
	scenarioID, ok := optionalUint(c, "scenario_id")
	if !ok {
		return
	}
	expired, err := strconv.ParseBool(defaultString(c.Query("expired_only"), "false"))
	if err != nil {
		util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeBadRequest, "expired_only must be true or false"))
		return
	}
	page, size := util.Pagination(c)
	result, serviceErr := h.service.List(c.Request.Context(), dto.SafeguardQuery{
		ScenarioID: scenarioID, Type: c.Query("type"), LifecycleState: c.Query("lifecycle_state"),
		ExpiredOnly: expired, Search: c.Query("search"), Page: page, PageSize: size,
	})
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *SafeguardHandler) Get(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}
func (h *SafeguardHandler) Create(c *gin.Context) {
	var request dto.CreateSafeguardRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Create(c.Request.Context(), request, mustActor(c))
	respond(c, http.StatusCreated, result, err)
}
func (h *SafeguardHandler) Update(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.UpdateSafeguardRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Update(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
func (h *SafeguardHandler) Verify(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.VerifySafeguardRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Verify(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
func (h *SafeguardHandler) Invalidate(c *gin.Context) { h.action(c, h.service.Invalidate) }
func (h *SafeguardHandler) Restore(c *gin.Context)    { h.action(c, h.service.Restore) }
func (h *SafeguardHandler) action(c *gin.Context, operation func(
	context.Context, uint, dto.SafeguardActionRequest, util.Actor,
) (dto.SafeguardResponse, error)) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.SafeguardActionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := operation(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
