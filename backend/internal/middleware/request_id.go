package middleware
import (
	"crypto/rand"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"strings"
)
const requestIDHeader = "X-Request-ID"
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if !validRequestID(requestID) {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header(requestIDHeader, requestID)
		c.Next()
	}
}
func validRequestID(value string) bool {
	if len(value) < 8 || len(value) > 80 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.' {
			continue
		}
		return false
	}
	return true
}
func generateRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(buffer)
}
