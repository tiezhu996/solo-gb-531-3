package router

import (
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/handler"
	"hazop-safeguard-coverage/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterDeviationScenarioRoutes(api *gin.RouterGroup, h *handler.DeviationScenarioHandler) {
	group := api.Group("/deviation-scenarios")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), h.List)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), h.Get)
	group.POST("", middleware.RequirePermission(constants.PermissionScenario), h.Create)
	group.PUT("/:id", middleware.RequirePermission(constants.PermissionScenario), h.Update)
	group.POST("/:id/transition", middleware.RequireRoles(
		constants.RoleAdmin, constants.RoleProcessEngineer, constants.RoleSafetyReviewer,
	), h.Transition)
}
