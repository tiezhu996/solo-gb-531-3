package middleware
import (
	"errors"
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/util"
	"log/slog"
)
func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		last := c.Errors.Last()
		var appErr *util.AppError
		if errors.As(last.Err, &appErr) {
			level := slog.LevelWarn
			if appErr.Status >= 500 {
				level = slog.LevelError
			}
			logger.Log(c.Request.Context(), level, "request failed",
				"request_id", util.RequestID(c),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"status", appErr.Status,
				"code", appErr.Code,
				"error", appErr.Error(),
			)
			return
		}
		logger.Error("unclassified request error",
			"request_id", util.RequestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"error", last.Err,
		)
	}
}
