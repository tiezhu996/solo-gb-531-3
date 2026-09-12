package middleware
import (
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/util"
	"log/slog"
	"net/http"
	"runtime/debug"
)
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered",
					"request_id", util.RequestID(c),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"panic", recovered,
					"stack", string(debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, util.Envelope{
					Code: string(util.CodeInternal), Message: "an unexpected error occurred",
					RequestID: util.RequestID(c),
				})
			}
		}()
		c.Next()
	}
}
