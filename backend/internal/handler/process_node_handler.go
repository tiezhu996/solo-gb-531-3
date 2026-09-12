package handler
import (
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/middleware"
	"hazop-safeguard-coverage/backend/internal/service"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
	"strconv"
)
type ProcessNodeHandler struct{ service service.ProcessNodeService }
func NewProcessNodeHandler(value service.ProcessNodeService) *ProcessNodeHandler {
	return &ProcessNodeHandler{service: value}
}
func (h *ProcessNodeHandler) List(c *gin.Context) {
	page, size := util.Pagination(c)
	result, err := h.service.List(c.Request.Context(), dto.ProcessNodeQuery{
		Search: c.Query("search"), UnitName: c.Query("unit_name"), OwnerTeam: c.Query("owner_team"),
		Status: c.Query("status"), Page: page, PageSize: size,
	})
	respond(c, http.StatusOK, result, err)
}
func (h *ProcessNodeHandler) Get(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, err := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}
func (h *ProcessNodeHandler) Create(c *gin.Context) {
	var request dto.CreateProcessNodeRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Create(c.Request.Context(), request, mustActor(c))
	respond(c, http.StatusCreated, result, err)
}
func (h *ProcessNodeHandler) Update(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.UpdateProcessNodeRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Update(c.Request.Context(), id, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
func (h *ProcessNodeHandler) Deactivate(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, err := h.service.Deactivate(c.Request.Context(), id, mustActor(c))
	respond(c, http.StatusOK, result, err)
}
func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		util.Fail(c, util.WrapError(http.StatusBadRequest, util.CodeValidation, "request body validation failed", err))
		return false
	}
	return true
}
func respond(c *gin.Context, status int, data any, err error) {
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Success(c, status, data)
}
func mustActor(c *gin.Context) util.Actor {
	actor, _ := middleware.ActorFromContext(c)
	return actor
}
func optionalUint(c *gin.Context, key string) (uint, bool) {
	raw := c.Query(key)
	if raw == "" {
		return 0, true
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeBadRequest, key+" must be a positive integer"))
		return 0, false
	}
	return uint(value), true
}
