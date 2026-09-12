package middleware
import (
	"fmt"
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
	"strconv"
	"strings"
	"time"
)
func Audit(audits repository.AuditRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions || c.Writer.Status() >= 400 {
			return
		}
		actor, ok := ActorFromContext(c)
		if !ok {
			return
		}
		entity, entityID := requestEntity(c.FullPath(), c.Param("id"))
		after, _ := util.CanonicalJSON(map[string]any{
			"method": c.Request.Method, "path": c.FullPath(), "status": c.Writer.Status(),
		})
		err := audits.Record(c.Request.Context(), model.AuditLog{
			RequestID: actor.RequestID, ActorID: actor.UserID, ActorName: actor.Username,
			ActorRole: actor.Role, EntityType: entity, EntityID: entityID, Action: "http_write",
			BeforeSnapshot: "{}", AfterSnapshot: after,
			DurationMS: time.Since(started).Milliseconds(), CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			_ = c.Error(fmt.Errorf("record HTTP write audit: %w", err))
		}
	}
}
func AuditListHandler(audits repository.AuditRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, size := util.Pagination(c)
		query := repository.AuditQuery{
			EntityType: c.Query("entity_type"), RequestID: c.Query("request_id"),
			Action: c.Query("action"), Page: page, PageSize: size,
		}
		if raw := c.Query("actor_id"); raw != "" {
			value, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeBadRequest, "actor_id must be a positive integer"))
				return
			}
			query.ActorID = uint(value)
		}
		if !parseAuditTime(c, "from", &query.From) || !parseAuditTime(c, "to", &query.To) {
			return
		}
		items, total, err := audits.List(c.Request.Context(), query)
		if err != nil {
			util.Fail(c, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list audit logs", err))
			return
		}
		util.Success(c, http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": size})
	}
}
func parseAuditTime(c *gin.Context, key string, target **time.Time) bool {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return true
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeBadRequest, key+" must use RFC3339"))
		return false
	}
	*target = &value
	return true
}
func requestEntity(path, rawID string) (string, uint) {
	path = strings.TrimPrefix(path, "/api/v1/")
	entity := strings.Split(path, "/")[0]
	entity = strings.ReplaceAll(entity, "-", "_")
	value, _ := strconv.ParseUint(rawID, 10, 64)
	return entity, uint(value)
}
