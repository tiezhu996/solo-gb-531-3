package middleware
import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"hazop-safeguard-coverage/backend/internal/config"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)
type Claims struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	jwt.RegisteredClaims
}
type Authenticator struct {
	users repository.UserRepository
	cfg   config.Config
}
func NewAuthenticator(users repository.UserRepository, cfg config.Config) *Authenticator {
	return &Authenticator{users: users, cfg: cfg}
}
func (a *Authenticator) Login(c *gin.Context) {
	var request struct {
		Username string `json:"username" binding:"required,min=2,max=80"`
		Password string `json:"password" binding:"required,min=6,max=128"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeValidation, "username and password are required"))
		return
	}
	user, err := a.users.FindByUsername(c.Request.Context(), request.Username)
	if err != nil || !user.Active || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
		util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "invalid username or password"))
		return
	}
	now := time.Now().UTC()
	expires := now.Add(a.cfg.JWTExpiry)
	claims := Claims{
		UserID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: a.cfg.JWTIssuer, Subject: user.Username,
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(a.cfg.JWTSecret))
	if err != nil {
		util.Fail(c, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to create access token", err))
		return
	}
	util.Success(c, http.StatusOK, gin.H{
		"token": token, "token_type": "Bearer", "expires_at": expires,
		"user": gin.H{"id": user.ID, "username": user.Username, "display_name": user.DisplayName, "role": user.Role},
	})
}
func (a *Authenticator) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "Bearer token is required"))
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(a.cfg.JWTSecret), nil
		}, jwt.WithIssuer(a.cfg.JWTIssuer), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "access token is invalid or expired"))
			return
		}
		c.Set("actor", util.Actor{
			UserID: claims.UserID, Username: claims.Username, DisplayName: claims.DisplayName,
			Role: claims.Role, RequestID: util.RequestID(c),
		})
		c.Next()
	}
}
func ActorFromContext(c *gin.Context) (util.Actor, bool) {
	value, exists := c.Get("actor")
	actor, ok := value.(util.Actor)
	return actor, exists && ok
}
type rateWindow struct {
	start time.Time
	count int
}
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	limit   int
}
func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{windows: make(map[string]rateWindow), limit: limit}
}
func (r *RateLimiter) Middleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := scope + ":" + c.ClientIP()
		if actor, ok := ActorFromContext(c); ok {
			key = scope + ":" + actor.Username
		}
		now := time.Now()
		r.mu.Lock()
		window := r.windows[key]
		if window.start.IsZero() || now.Sub(window.start) >= time.Minute {
			window = rateWindow{start: now}
		}
		window.count++
		r.windows[key] = window
		if len(r.windows) > 10000 {
			for candidate, state := range r.windows {
				if now.Sub(state.start) > 2*time.Minute {
					delete(r.windows, candidate)
				}
			}
		}
		allowed := window.count <= r.limit
		retryAfter := int(time.Minute.Seconds() - now.Sub(window.start).Seconds())
		r.mu.Unlock()
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			util.Fail(c, util.NewError(http.StatusTooManyRequests, util.CodeRateLimited, "request rate limit exceeded"))
			return
		}
		c.Next()
	}
}
