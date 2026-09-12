package router

import (
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/handler"
	"hazop-safeguard-coverage/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterProcessNodeRoutes(api *gin.RouterGroup, h *handler.ProcessNodeHandler) {
	group := api.Group("/process-nodes")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), h.List)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), h.Get)
	group.POST("", middleware.RequirePermission(constants.PermissionNodeWrite), h.Create)
	group.PUT("/:id", middleware.RequirePermission(constants.PermissionNodeWrite), h.Update)
	group.POST("/:id/deactivate", middleware.RequirePermission(constants.PermissionNodeWrite), h.Deactivate)
}
