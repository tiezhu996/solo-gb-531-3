package router

import (
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/handler"
	"hazop-safeguard-coverage/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterCoverageEvaluationRoutes(
	api *gin.RouterGroup,
	h *handler.CoverageEvaluationHandler,
	runLimiter *middleware.RateLimiter,
) {
	group := api.Group("/coverage-evaluations")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), h.List)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), h.Get)
	group.GET("/:id/compare/:other_id", middleware.RequirePermission(constants.PermissionRead), h.Compare)
	group.POST("", middleware.RequirePermission(constants.PermissionEvaluation), runLimiter.Middleware("coverage-run"), h.Run)
	group.POST("/:id/confirm", middleware.RequirePermission(constants.PermissionConfirm), h.Confirm)
	group.POST("/:id/void", middleware.RequirePermission(constants.PermissionConfirm), h.Void)
	group.POST("/:id/replay", middleware.RequirePermission(constants.PermissionEvaluation), runLimiter.Middleware("coverage-replay"), h.Replay)
}
