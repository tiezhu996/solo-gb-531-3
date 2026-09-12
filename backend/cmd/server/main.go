package main
import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/algorithm"
	"hazop-safeguard-coverage/backend/internal/config"
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/database"
	"hazop-safeguard-coverage/backend/internal/handler"
	"hazop-safeguard-coverage/backend/internal/middleware"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/router"
	"hazop-safeguard-coverage/backend/internal/service"
	"hazop-safeguard-coverage/backend/internal/util"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := database.Close(db); closeErr != nil {
			logger.Error("database close failed", "error", closeErr)
		}
	}()
	nodeRepo := repository.NewProcessNodeRepository(db)
	scenarioRepo := repository.NewDeviationScenarioRepository(db)
	safeguardRepo := repository.NewSafeguardRepository(db)
	evaluationRepo := repository.NewCoverageEvaluationRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	userRepo := repository.NewUserRepository(db)
	nodeHandler := handler.NewProcessNodeHandler(service.NewProcessNodeService(nodeRepo, auditRepo))
	scenarioHandler := handler.NewDeviationScenarioHandler(service.NewDeviationScenarioService(scenarioRepo, nodeRepo, auditRepo))
	safeguardHandler := handler.NewSafeguardHandler(service.NewSafeguardService(safeguardRepo, scenarioRepo, auditRepo))
	evaluationHandler := handler.NewCoverageEvaluationHandler(service.NewCoverageEvaluationService(
		evaluationRepo, scenarioRepo, nodeRepo, safeguardRepo, auditRepo, algorithm.NewEvaluator(),
	))
	auth := middleware.NewAuthenticator(userRepo, cfg)
	loginLimiter := middleware.NewRateLimiter(cfg.LoginLimitPerMinute)
	runLimiter := middleware.NewRateLimiter(cfg.RunLimitPerMinute)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.Recovery(logger), middleware.ErrorHandler(logger))
	engine.GET("/healthz", func(c *gin.Context) {
		sqlDB, dbErr := db.DB()
		if dbErr != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			util.Fail(c, util.NewError(http.StatusInternalServerError, util.CodeInternal, "database is unavailable"))
			return
		}
		util.Success(c, http.StatusOK, gin.H{
			"status": "healthy", "service": "hazop-safeguard-coverage", "time": time.Now().UTC(),
		})
	})
	v1 := engine.Group("/api/v1")
	v1.POST("/auth/login", loginLimiter.Middleware("login"), auth.Login)
	api := v1.Group("")
	api.Use(auth.RequireAuth(), middleware.Audit(auditRepo))
	router.RegisterProcessNodeRoutes(api, nodeHandler)
	router.RegisterDeviationScenarioRoutes(api, scenarioHandler)
	router.RegisterSafeguardRoutes(api, safeguardHandler)
	router.RegisterCoverageEvaluationRoutes(api, evaluationHandler, runLimiter)
	api.GET("/audit-logs", middleware.RequireRoles(constants.RoleAdmin, constants.RoleSafetyReviewer, constants.RoleAuditor), middleware.AuditListHandler(auditRepo))
	api.GET("/meta/enums", middleware.RequirePermission(constants.PermissionRead), func(c *gin.Context) {
		util.Success(c, http.StatusOK, gin.H{
			"deviation_guideword": constants.DeviationGuidewordValues(),
			"coverage_state":      constants.CoverageStateValues(),
			"scenario_state":      constants.ScenarioStateValues(),
			"roles":               constants.RoleValues(),
		})
	})
	engine.NoRoute(func(c *gin.Context) {
		util.Fail(c, util.NewError(http.StatusNotFound, util.CodeNotFound, "route was not found"))
	})
	server := &http.Server{Addr: ":" + cfg.Port, Handler: engine, ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	logger.Info("server started", "port", cfg.Port, "database_driver", cfg.DBDriver)
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}
	return nil
}
