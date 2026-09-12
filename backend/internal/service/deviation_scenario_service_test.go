package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeviationServiceStateAndReviewerSeparation(t *testing.T) {
	db := testDB(t)
	nodeRepo := repository.NewProcessNodeRepository(db)
	scenarioRepo := repository.NewDeviationScenarioRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	node := model.ProcessNode{
		NodeCode: "T-101", Name: "Test Node", UnitName: "Test Unit", Medium: "water",
		DesignPressure: 1, DesignTemperature: 100, OwnerTeam: "test", Status: "active",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := nodeRepo.Create(context.Background(), &node); err != nil {
		t.Fatalf("create node: %v", err)
	}
	service := NewDeviationScenarioService(scenarioRepo, nodeRepo, auditRepo)
	engineer := util.Actor{UserID: 10, Username: "engineer", Role: "process_engineer", RequestID: "test-request-1"}
	authorReviewer := util.Actor{UserID: 10, Username: "engineer", Role: "safety_reviewer", RequestID: "test-request-1b"}
	reviewer := util.Actor{UserID: 20, Username: "reviewer", Role: "safety_reviewer", RequestID: "test-request-2"}
	created, err := service.Create(context.Background(), dto.CreateDeviationScenarioRequest{
		ProcessNodeID: node.ID, Guideword: "more", Parameter: "temperature",
		Cause: "cooling loss", Consequence: "overpressure", Likelihood: 4, Severity: 5,
	}, authorReviewer)
	if err != nil {
		t.Fatalf("create scenario: %v", err)
	}
	analyzed, err := service.Transition(context.Background(), created.ID, dto.TransitionScenarioRequest{
		ToState: "analyzed", Version: created.Version,
	}, engineer)
	if err != nil || analyzed.ScenarioState != "analyzed" {
		t.Fatalf("analyze transition: state=%s err=%v", analyzed.ScenarioState, err)
	}
	_, err = service.Transition(context.Background(), created.ID, dto.TransitionScenarioRequest{
		ToState: "accepted", Version: analyzed.Version,
	}, reviewer)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("illegal transition should return 409, got %v", err)
	}
	_, err = service.Transition(context.Background(), created.ID, dto.TransitionScenarioRequest{
		ToState: "verified", Version: analyzed.Version,
	}, authorReviewer)
	if !errors.As(err, &appErr) || appErr.Code != util.CodeReviewerConflict {
		t.Fatalf("author review should be rejected, got %v", err)
	}
	verified, err := service.Transition(context.Background(), created.ID, dto.TransitionScenarioRequest{
		ToState: "verified", Version: analyzed.Version,
	}, reviewer)
	if err != nil || verified.ScenarioState != "verified" || verified.ReviewedBy == nil || *verified.ReviewedBy != reviewer.UserID {
		t.Fatalf("review transition failed: %#v err=%v", verified, err)
	}
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.ProcessNode{}, &model.DeviationScenario{},
		&model.Safeguard{}, &model.CoverageEvaluation{}, &model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
