package util
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"github.com/gin-gonic/gin"
)
type ErrorCode string
const (
	CodeBadRequest       ErrorCode = "BAD_REQUEST"
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	CodeForbidden        ErrorCode = "FORBIDDEN"
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeValidation       ErrorCode = "VALIDATION_FAILED"
	CodeRateLimited      ErrorCode = "RATE_LIMITED"
	CodeInternal         ErrorCode = "INTERNAL_ERROR"
	CodeIdempotency      ErrorCode = "IDEMPOTENCY_KEY_REQUIRED"
	CodeStateTransition  ErrorCode = "INVALID_STATE_TRANSITION"
	CodeReviewerConflict ErrorCode = "REVIEWER_AUTHOR_CONFLICT"
)
type AppError struct {
	Status  int
	Code    ErrorCode
	Message string
	Cause   error
}
func (e *AppError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}
func (e *AppError) Unwrap() error { return e.Cause }
func NewError(status int, code ErrorCode, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}
func WrapError(status int, code ErrorCode, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Cause: cause}
}
func Conflict(message string) *AppError {
	return NewError(http.StatusConflict, CodeConflict, message)
}
func NotFound(entity string) *AppError {
	return NewError(http.StatusNotFound, CodeNotFound, entity+" was not found")
}
type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id"`
}
func Success(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{Code: "OK", Message: "success", Data: data, RequestID: RequestID(c)})
}
func Fail(c *gin.Context, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = WrapError(http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", err)
	}
	c.Error(appErr) //nolint:errcheck
	c.AbortWithStatusJSON(appErr.Status, Envelope{
		Code: string(appErr.Code), Message: appErr.Message, RequestID: RequestID(c),
	})
}
func RequestID(c *gin.Context) string {
	if value, exists := c.Get("request_id"); exists {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
type Actor struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	RequestID   string `json:"request_id"`
}
func ParseUintParam(c *gin.Context, name string) (uint, error) {
	raw := c.Param(name)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return 0, NewError(http.StatusBadRequest, CodeBadRequest, name+" must be a positive integer")
	}
	return uint(value), nil
}
func Pagination(c *gin.Context) (page, pageSize int) {
	page = positiveInt(c.Query("page"), 1)
	pageSize = positiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
func CanonicalJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal canonical JSON: %w", err)
	}
	return string(data), nil
}
func HashString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
func CompactText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}
