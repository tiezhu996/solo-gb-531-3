package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"hazop-safeguard-coverage/backend/internal/algorithm"
	"hazop-safeguard-coverage/backend/internal/config"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/handler"
	"hazop-safeguard-coverage/backend/internal/middleware"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/router"
	"hazop-safeguard-coverage/backend/internal/service"
	"hazop-safeguard-coverage/backend/internal/util"
)

type exportFixture struct {
	engine *gin.Engine
	cfg    config.Config
	db     *gorm.DB
	evals  repository.CoverageEvaluationRepository
	svc    service.CoverageEvaluationService

	node     model.ProcessNode
	scenario model.DeviationScenario
}

type testEnvelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"request_id"`
}

func setupExportFixture(t *testing.T) *exportFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.ProcessNode{}, &model.DeviationScenario{},
		&model.Safeguard{}, &model.CoverageEvaluation{}, &model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	cfg := config.Config{
		JWTSecret: "evidence-pack-test-secret-0123456789", JWTIssuer: "hazop-safeguard-coverage",
		JWTExpiry: time.Hour, LoginLimitPerMinute: 10000, RunLimitPerMinute: 10000,
	}
	ctx := context.Background()
	now := time.Now().UTC()

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	users := []model.User{
		{Username: "engineer", DisplayName: "Engineer", PasswordHash: string(passwordHash), Role: "process_engineer", Active: true, CreatedAt: now, UpdatedAt: now},
		{Username: "reviewer", DisplayName: "Reviewer", PasswordHash: string(passwordHash), Role: "safety_reviewer", Active: true, CreatedAt: now, UpdatedAt: now},
		{Username: "auditor", DisplayName: "Auditor", PasswordHash: string(passwordHash), Role: "auditor", Active: true, CreatedAt: now, UpdatedAt: now},
	}
	userRepo := repository.NewUserRepository(db)
	for index := range users {
		if err := db.Create(&users[index]).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	nodeRepo := repository.NewProcessNodeRepository(db)
	scenarioRepo := repository.NewDeviationScenarioRepository(db)
	safeguardRepo := repository.NewSafeguardRepository(db)
	evalRepo := repository.NewCoverageEvaluationRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	node := model.ProcessNode{
		NodeCode: "R-901", Name: "Evidence Reactor", UnitName: "Oxidation", Medium: "hydrocarbon",
		DesignPressure: 2.5, DesignTemperature: 200, OwnerTeam: "pss", Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := nodeRepo.Create(ctx, &node); err != nil {
		t.Fatalf("seed node: %v", err)
	}
	scenario := model.DeviationScenario{
		ProcessNodeID: node.ID, Guideword: "more", Parameter: "temperature",
		Cause: "cooling loss", Consequence: "overpressure",
		Likelihood: 4, Severity: 5, ScenarioState: "analyzed", Version: 1,
		CreatedBy: users[0].ID, CreatedByName: "engineer", CreatedAt: now, UpdatedAt: now,
	}
	if err := scenarioRepo.Create(ctx, &scenario); err != nil {
		t.Fatalf("seed scenario: %v", err)
	}
	verified := now.AddDate(0, 0, -3)
	safeguards := []model.Safeguard{
		{
			Name: "High temperature SIS trip", SafeguardType: "sis", TargetScenarioID: scenario.ID,
			IndependenceKey: "SIS-T901", Effectiveness: 0.9, TestIntervalDays: 365,
			LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "proof test A",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			Name: "Shared logic solver trip", SafeguardType: "sis", TargetScenarioID: scenario.ID,
			IndependenceKey: "SIS-T901", Effectiveness: 0.5, TestIntervalDays: 365,
			LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "same logic solver, deduplicated",
			CreatedAt: now, UpdatedAt: now,
		},
	}
	for index := range safeguards {
		if err := safeguardRepo.Create(ctx, &safeguards[index]); err != nil {
			t.Fatalf("seed safeguard: %v", err)
		}
	}

	svc := service.NewCoverageEvaluationService(
		evalRepo, scenarioRepo, nodeRepo, safeguardRepo, auditRepo, algorithm.NewEvaluator(),
	)
	h := handler.NewCoverageEvaluationHandler(svc)
	auth := middleware.NewAuthenticator(userRepo, cfg)
	limiter := middleware.NewRateLimiter(10000)

	engine := gin.New()
	v1 := engine.Group("/api/v1")
	v1.POST("/auth/login", limiter.Middleware("login"), auth.Login)
	api := v1.Group("")
	api.Use(auth.RequireAuth())
	router.RegisterCoverageEvaluationRoutes(api, h, limiter)

	return &exportFixture{
		engine: engine, cfg: cfg, db: db, evals: evalRepo, svc: svc,
		node: node, scenario: scenario,
	}
}

func (f *exportFixture) login(t *testing.T, username string) string {
	t.Helper()
	body := `{"username":"` + username + `","password":"password123"}`
	w := f.do(t, http.MethodPost, "/api/v1/auth/login", "", body)
	if w.Code != http.StatusOK {
		t.Fatalf("login %s failed: %d %s", username, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return resp.Data.Token
}

func (f *exportFixture) forgedToken(t *testing.T, role string, userID uint) string {
	t.Helper()
	claims := middleware.Claims{
		UserID: userID, Username: "operator-" + role, DisplayName: "Operator", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: f.cfg.JWTIssuer, Subject: "operator-" + role,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(f.cfg.JWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func (f *exportFixture) runCompleted(t *testing.T, idempotencyKey string) model.CoverageEvaluation {
	t.Helper()
	_, _, err := f.svc.Run(context.Background(), dto.RunCoverageEvaluationRequest{ScenarioID: f.scenario.ID}, idempotencyKey, util.Actor{
		UserID: 1, Username: "engineer", Role: "process_engineer", RequestID: "setup-run-" + idempotencyKey,
	})
	if err != nil {
		t.Fatalf("seed completed evaluation: %v", err)
	}
	stored, err := f.evals.FindByIdempotencyKey(context.Background(), idempotencyKey)
	if err != nil {
		t.Fatalf("reload seeded evaluation: %v", err)
	}
	return stored
}

// insertStateRecord clones a completed evaluation row into another lifecycle state.
func (f *exportFixture) insertStateRecord(t *testing.T, base model.CoverageEvaluation, key, state string, mutate func(*model.CoverageEvaluation)) model.CoverageEvaluation {
	t.Helper()
	clone := base
	clone.ID = 0
	clone.IdempotencyKey = key
	clone.EvaluationState = state
	clone.CoverageScore = 0
	if mutate != nil {
		mutate(&clone)
	}
	if err := f.evals.Create(context.Background(), &clone); err != nil {
		t.Fatalf("insert %s evaluation: %v", state, err)
	}
	return clone
}

func (f *exportFixture) do(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)
	return w
}

func (f *exportFixture) export(t *testing.T, id uint, token string) *httptest.ResponseRecorder {
	t.Helper()
	return f.do(t, http.MethodGet, fmt.Sprintf("/api/v1/coverage-evaluations/%d/evidence-pack", id), token, "")
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) testEnvelope {
	t.Helper()
	var envelope testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope %q: %v", w.Body.String(), err)
	}
	return envelope
}

func decodePack(t *testing.T, w *httptest.ResponseRecorder) dto.CoverageEvidencePackResponse {
	t.Helper()
	envelope := decodeEnvelope(t, w)
	var pack dto.CoverageEvidencePackResponse
	if err := json.Unmarshal(envelope.Data, &pack); err != nil {
		t.Fatalf("decode pack: %v", err)
	}
	return pack
}

// TestEvidencePackExportLifecycleStates covers queued, running, completed, failed and voided
// exports: every state must carry an accurate readable status explanation, while the frozen
// input snapshot stays byte-stable regardless of lifecycle state or repeated exports.
func TestEvidencePackExportLifecycleStates(t *testing.T) {
	f := setupExportFixture(t)
	engineer := f.login(t, "engineer")
	reviewer := f.login(t, "reviewer")

	completed := f.runCompleted(t, "http-export-completed-01")

	queued := f.insertStateRecord(t, completed, "http-export-queued-0001", "queued", func(e *model.CoverageEvaluation) {
		e.UncoveredPaths, e.DeduplicatedSafeguards, e.Explanation = "[]", "[]", "{}"
	})
	running := f.insertStateRecord(t, completed, "http-export-running-0001", "running", func(e *model.CoverageEvaluation) {
		e.UncoveredPaths, e.DeduplicatedSafeguards, e.Explanation = "[]", "[]", "{}"
	})
	failed := f.insertStateRecord(t, completed, "http-export-failed-0001", "failed", func(e *model.CoverageEvaluation) {
		e.UncoveredPaths, e.DeduplicatedSafeguards, e.Explanation = "[]", "[]", "{}"
		e.FailureReason = "simulated algorithm failure: snapshot requires persisted node"
	})
	voidedRun := f.runCompleted(t, "http-export-voided-0001")
	if _, err := f.svc.Void(context.Background(), voidedRun.ID, util.Actor{
		UserID: 2, Username: "reviewer", Role: "safety_reviewer", RequestID: "setup-void",
	}); err != nil {
		t.Fatalf("seed voided evaluation: %v", err)
	}
	voided, _ := f.evals.GetByID(context.Background(), voidedRun.ID)

	cases := []struct {
		name             string
		id               uint
		state            string
		label            string
		summaryKeyword   string
		failureReason    string
		wantSteps        int
		wantUncovered    int
		wantDeduplicated int
		frozenJSON       string
		frozenHash       string
	}{
		{name: "queued", id: queued.ID, state: "queued", label: "Queued", summaryKeyword: "等待计算", wantSteps: 0, wantUncovered: 0, wantDeduplicated: 0, frozenJSON: queued.InputSnapshot, frozenHash: queued.InputHash},
		{name: "running", id: running.ID, state: "running", label: "Running", summaryKeyword: "计算中", wantSteps: 0, wantUncovered: 0, wantDeduplicated: 0, frozenJSON: running.InputSnapshot, frozenHash: running.InputHash},
		{name: "completed", id: completed.ID, state: "completed", label: "Completed", summaryKeyword: "等待人工确认", wantSteps: 2, wantUncovered: 0, wantDeduplicated: 1, frozenJSON: completed.InputSnapshot, frozenHash: completed.InputHash},
		{name: "failed", id: failed.ID, state: "failed", label: "Failed", summaryKeyword: "计算失败", failureReason: "simulated algorithm failure", wantSteps: 0, wantUncovered: 0, wantDeduplicated: 0, frozenJSON: failed.InputSnapshot, frozenHash: failed.InputHash},
		{name: "voided", id: voided.ID, state: "voided", label: "Voided", summaryKeyword: "已作废", wantSteps: 2, wantUncovered: 0, wantDeduplicated: 1, frozenJSON: voided.InputSnapshot, frozenHash: voided.InputHash},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 导出两次：结果内容必须可重复，冻结快照不随时间或状态改变。
			first := f.export(t, tc.id, engineer)
			if first.Code != http.StatusOK {
				t.Fatalf("first export status = %d body=%s", first.Code, first.Body.String())
			}
			second := f.export(t, tc.id, reviewer) // 复核人同样可只读导出
			if second.Code != http.StatusOK {
				t.Fatalf("second export status = %d body=%s", second.Code, second.Body.String())
			}
			pack := decodePack(t, first)
			packAgain := decodePack(t, second)

			if pack.State.Code != tc.state || pack.State.Label != tc.label {
				t.Fatalf("state = %s/%s, want %s/%s", pack.State.Code, pack.State.Label, tc.state, tc.label)
			}
			if !strings.Contains(pack.State.ReadableSummary, tc.summaryKeyword) {
				t.Fatalf("readable summary %q must identify %q", pack.State.ReadableSummary, tc.summaryKeyword)
			}
			if tc.failureReason != "" {
				if !strings.Contains(pack.State.FailureReason, tc.failureReason) {
					t.Fatalf("failure reason %q must contain %q", pack.State.FailureReason, tc.failureReason)
				}
			} else if pack.State.FailureReason != "" {
				t.Fatalf("%s pack must not carry a failure reason, got %q", tc.name, pack.State.FailureReason)
			}
			if len(pack.ScoreSteps) != tc.wantSteps {
				t.Fatalf("score steps = %d, want %d", len(pack.ScoreSteps), tc.wantSteps)
			}
			if len(pack.UncoveredPaths) != tc.wantUncovered {
				t.Fatalf("uncovered paths = %d, want %d", len(pack.UncoveredPaths), tc.wantUncovered)
			}
			if len(pack.DeduplicatedSafeguards) != tc.wantDeduplicated {
				t.Fatalf("deduplicated safeguards = %d, want %d", len(pack.DeduplicatedSafeguards), tc.wantDeduplicated)
			}
			if pack.InputHash != tc.frozenHash {
				t.Fatalf("input hash = %q, want frozen %q", pack.InputHash, tc.frozenHash)
			}
			gotSnapshot, err := json.Marshal(pack.InputSnapshot)
			if err != nil {
				t.Fatalf("re-marshal snapshot: %v", err)
			}
			if string(gotSnapshot) != tc.frozenJSON {
				t.Fatalf("frozen snapshot changed on %s export", tc.name)
			}
			// 第二次导出的冻结内容与第一次完全一致（仅 exported_at 允许不同）。
			if string(mustMarshal(t, packAgain.InputSnapshot)) != string(mustMarshal(t, pack.InputSnapshot)) ||
				packAgain.InputHash != pack.InputHash || packAgain.State.Code != pack.State.Code {
				t.Fatal("repeated export produced inconsistent evidence")
			}
		})
	}
}

// TestEvidencePackExportCompletedMatchesDetail pins the rule that export reuses the exact
// immutable snapshot of the existing detail entry point instead of rebuilding inputs.
func TestEvidencePackExportCompletedMatchesDetail(t *testing.T) {
	f := setupExportFixture(t)
	token := f.login(t, "auditor")
	completed := f.runCompleted(t, "http-export-detail-0001")

	detailW := f.do(t, http.MethodGet, fmt.Sprintf("/api/v1/coverage-evaluations/%d", completed.ID), token, "")
	if detailW.Code != http.StatusOK {
		t.Fatalf("detail status = %d", detailW.Code)
	}
	var detailEnv struct {
		Data dto.CoverageEvaluationResponse `json:"data"`
	}
	if err := json.Unmarshal(detailW.Body.Bytes(), &detailEnv); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	packW := f.export(t, completed.ID, token)
	pack := decodePack(t, packW)
	if string(mustMarshal(t, pack.InputSnapshot)) != string(mustMarshal(t, detailEnv.Data.InputSnapshot)) {
		t.Fatal("exported snapshot diverges from the existing evaluation snapshot")
	}
	if pack.InputHash != detailEnv.Data.InputHash {
		t.Fatal("exported input hash diverges from the stored evaluation hash")
	}
}

// TestEvidencePackExportAuthenticationAndAuthorization covers 401 without a session and
// 403 for an authenticated role without read permission.
func TestEvidencePackExportAuthenticationAndAuthorization(t *testing.T) {
	f := setupExportFixture(t)
	completed := f.runCompleted(t, "http-export-auth-0001")

	unauthenticated := f.export(t, completed.ID, "")
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("no token: status = %d, want 401", unauthenticated.Code)
	}
	env := decodeEnvelope(t, unauthenticated)
	if env.Code != string(util.CodeUnauthorized) || env.Data != nil {
		t.Fatalf("unauthenticated response = %#v", env)
	}

	malformed := f.do(t, http.MethodGet, fmt.Sprintf("/api/v1/coverage-evaluations/%d/evidence-pack", completed.ID), "not-a-bearer-token", "")
	if malformed.Code != http.StatusUnauthorized {
		t.Fatalf("malformed authorization: status = %d, want 401", malformed.Code)
	}

	// A validly signed JWT for an unknown/operator role passes authentication but fails RBAC.
	operatorToken := f.forgedToken(t, "field_operator", 99)
	forbidden := f.export(t, completed.ID, operatorToken)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("operator role: status = %d, want 403", forbidden.Code)
	}
	forbiddenEnv := decodeEnvelope(t, forbidden)
	if forbiddenEnv.Code != string(util.CodeForbidden) || !strings.Contains(forbiddenEnv.Message, "permission") {
		t.Fatalf("forbidden response must name the violated RBAC rule: %#v", forbiddenEnv)
	}
	if forbiddenEnv.Data != nil {
		t.Fatal("forbidden export must not return evidence data")
	}
}

// TestEvidencePackExportCorruptRecords verifies that a damaged persisted record is rejected
// with 422 and a message that names the broken evidence artifact, so no evidence file is produced.
func TestEvidencePackExportCorruptRecords(t *testing.T) {
	f := setupExportFixture(t)
	token := f.login(t, "engineer")
	completed := f.runCompleted(t, "http-export-corrupt-001")

	cases := []struct {
		name       string
		key        string
		mutate     map[string]any
		wantSubstr string
	}{
		{
			name:       "frozen snapshot corrupt",
			key:        "http-corrupt-snapshot-01",
			mutate:     map[string]any{"input_snapshot": "{broken-json"},
			wantSubstr: "frozen input snapshot",
		},
		{
			name:       "scoring explanation corrupt",
			key:        "http-corrupt-explain-001",
			mutate:     map[string]any{"explanation": "{broken-json"},
			wantSubstr: "scoring steps explanation",
		},
		{
			name:       "uncovered paths corrupt",
			key:        "http-corrupt-paths-0001",
			mutate:     map[string]any{"uncovered_paths": "{broken-json"},
			wantSubstr: "uncovered paths record",
		},
		{
			name:       "dedup record corrupt",
			key:        "http-corrupt-dedup-0001",
			mutate:     map[string]any{"deduplicated_safeguards": "{broken-json"},
			wantSubstr: "independence dedup record",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := completed
			row.ID = 0
			row.IdempotencyKey = tc.key
			if err := f.evals.Create(context.Background(), &row); err != nil {
				t.Fatalf("seed row: %v", err)
			}
			if err := f.db.Model(&model.CoverageEvaluation{}).Where("id = ?", row.ID).Updates(tc.mutate).Error; err != nil {
				t.Fatalf("corrupt row: %v", err)
			}
			w := f.export(t, row.ID, token)
			if w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422; body=%s", w.Code, w.Body.String())
			}
			env := decodeEnvelope(t, w)
			if env.Code != string(util.CodeValidation) {
				t.Fatalf("error code = %q, want VALIDATION_FAILED", env.Code)
			}
			if !strings.Contains(env.Message, tc.wantSubstr) {
				t.Fatalf("error message %q must name violated artifact %q", env.Message, tc.wantSubstr)
			}
			if env.Data != nil {
				t.Fatal("failed export must not return evidence payload for download")
			}
		})
	}
}

// TestEvidencePackExportMissingRecord keeps the not-found contract explicit.
func TestEvidencePackExportMissingRecord(t *testing.T) {
	f := setupExportFixture(t)
	token := f.login(t, "auditor")
	w := f.export(t, 424242, token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	env := decodeEnvelope(t, w)
	if env.Code != string(util.CodeNotFound) || !strings.Contains(env.Message, "coverage evaluation") {
		t.Fatalf("missing-record response = %#v", env)
	}
	if env.Data != nil {
		t.Fatal("missing-record export must not return evidence payload")
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}
