package database
import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"hazop-safeguard-coverage/backend/internal/config"
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/model"
	"time"
)
func Open(cfg config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.DBDriver {
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.DBDriver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: false,
		TranslateError:                           true,
	})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.DBDriver, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("obtain database connection pool: %w", err)
	}
	if cfg.DBDriver == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	} else {
		sqlDB.SetMaxOpenConns(20)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	if err := seed(db); err != nil {
		return nil, err
	}
	return db, nil
}
func migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&model.User{},
		&model.ProcessNode{},
		&model.DeviationScenario{},
		&model.Safeguard{},
		&model.CoverageEvaluation{},
		&model.AuditLog{},
	)
	if err != nil {
		return fmt.Errorf("migrate database schema: %w", err)
	}
	return nil
}
type seedAccount struct {
	Username    string
	DisplayName string
	Password    string
	Role        constants.Role
}
func seed(db *gorm.DB) error {
	accounts := []seedAccount{
		{Username: "admin", DisplayName: "System Administrator", Password: "admin123", Role: constants.RoleAdmin},
		{Username: "engineer", DisplayName: "Process Engineer", Password: "engineer123", Role: constants.RoleProcessEngineer},
		{Username: "reviewer", DisplayName: "Safety Reviewer", Password: "reviewer123", Role: constants.RoleSafetyReviewer},
		{Username: "auditor", DisplayName: "Compliance Auditor", Password: "auditor123", Role: constants.RoleAuditor},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		users := make(map[string]model.User, len(accounts))
		for _, account := range accounts {
			var user model.User
			err := tx.Where("username = ?", account.Username).First(&user).Error
			if err == nil {
				users[account.Username] = user
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("find seed user %s: %w", account.Username, err)
			}
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
			if hashErr != nil {
				return fmt.Errorf("hash seed password for %s: %w", account.Username, hashErr)
			}
			user = model.User{
				Username: account.Username, DisplayName: account.DisplayName,
				PasswordHash: string(hash), Role: string(account.Role), Active: true,
			}
			if err := tx.Create(&user).Error; err != nil {
				return fmt.Errorf("create seed user %s: %w", account.Username, err)
			}
			users[account.Username] = user
		}
		return seedDomain(tx, users)
	})
}
func seedDomain(tx *gorm.DB, users map[string]model.User) error {
	var count int64
	if err := tx.Model(&model.ProcessNode{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count seed process nodes: %w", err)
	}
	if count > 0 {
		return nil
	}
	now := time.Now().UTC().Truncate(time.Second)
	nodes := []model.ProcessNode{
		{
			NodeCode: "R-101", Name: "Oxidation Reactor", UnitName: "Oxidation Unit",
			Medium: "Hydrocarbon / air mixture", DesignPressure: 2.5, DesignTemperature: 235,
			OwnerTeam: "Reaction Safety", Status: "active", CreatedAt: now, UpdatedAt: now,
		},
		{
			NodeCode: "V-204", Name: "High-pressure Separator", UnitName: "Separation Unit",
			Medium: "Wet process gas", DesignPressure: 8.2, DesignTemperature: 120,
			OwnerTeam: "Separation Operations", Status: "active", CreatedAt: now, UpdatedAt: now,
		},
	}
	if err := tx.Create(&nodes).Error; err != nil {
		return fmt.Errorf("create seed process nodes: %w", err)
	}
	engineer := users["engineer"]
	reviewer := users["reviewer"]
	scenarios := []model.DeviationScenario{
		{
			ProcessNodeID: nodes[0].ID, Guideword: "more", Parameter: "temperature",
			Cause:       "Cooling water flow lost; runaway side reaction",
			Consequence: "Reactor overpressure; flammable material release",
			Likelihood:  4, Severity: 5, ScenarioState: "analyzed", Version: 2,
			CreatedBy: engineer.ID, CreatedByName: engineer.Username,
			CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour),
		},
		{
			ProcessNodeID: nodes[1].ID, Guideword: "more", Parameter: "pressure",
			Cause:       "Blocked gas outlet; downstream valve inadvertently closed",
			Consequence: "Separator shell rupture; personnel exposure",
			Likelihood:  3, Severity: 5, ScenarioState: "draft", Version: 1,
			CreatedBy: engineer.ID, CreatedByName: engineer.Username,
			CreatedAt: now.Add(-36 * time.Hour), UpdatedAt: now.Add(-36 * time.Hour),
		},
		{
			ProcessNodeID: nodes[0].ID, Guideword: "reverse", Parameter: "flow",
			Cause:       "Check valve leakage",
			Consequence: "Backflow contaminates upstream feed",
			Likelihood:  2, Severity: 3, ScenarioState: "verified", Version: 3,
			CreatedBy: engineer.ID, CreatedByName: engineer.Username,
			ReviewedBy: &reviewer.ID, ReviewedByName: reviewer.Username,
			CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now.Add(-12 * time.Hour),
		},
	}
	if err := tx.Create(&scenarios).Error; err != nil {
		return fmt.Errorf("create seed deviation scenarios: %w", err)
	}
	validVerification := now.AddDate(0, 0, -20)
	expiredVerification := now.AddDate(0, 0, -500)
	safeguards := []model.Safeguard{
		{
			Name: "High temperature SIS trip", SafeguardType: "interlock",
			TargetScenarioID: scenarios[0].ID, IndependenceKey: "SIS-R101-TEMP",
			Effectiveness: 0.8, TestIntervalDays: 365, LastVerifiedAt: &validVerification,
			LifecycleState: "active", EvidenceNote: "Proof-test certificate SIS-2026-041",
			LastVerificationBy: &reviewer.ID, CreatedAt: now, UpdatedAt: now,
		},
		{
			Name: "Duplicate temperature shutdown indication", SafeguardType: "alarm",
			TargetScenarioID: scenarios[0].ID, IndependenceKey: "SIS-R101-TEMP",
			Effectiveness: 0.45, TestIntervalDays: 180, LastVerifiedAt: &validVerification,
			LifecycleState: "active", EvidenceNote: "Shares final element with SIS trip; intentionally demonstrates independence deduplication",
			LastVerificationBy: &reviewer.ID, CreatedAt: now, UpdatedAt: now,
		},
		{
			Name: "Separator pressure relief valve", SafeguardType: "relief",
			TargetScenarioID: scenarios[1].ID, IndependenceKey: "PSV-V204-01",
			Effectiveness: 0.9, TestIntervalDays: 365, LastVerifiedAt: &expiredVerification,
			LifecycleState: "expired", EvidenceNote: "Historical certificate expired; reproducible coverage gap",
			LastVerificationBy: &reviewer.ID, CreatedAt: now, UpdatedAt: now,
		},
		{
			Name: "Feed check valve", SafeguardType: "containment",
			TargetScenarioID: scenarios[2].ID, IndependenceKey: "NRV-R101-FEED",
			Effectiveness: 0.7, TestIntervalDays: 180, LastVerifiedAt: &validVerification,
			LifecycleState: "active", EvidenceNote: "Inspection record CV-882",
			LastVerificationBy: &reviewer.ID, CreatedAt: now, UpdatedAt: now,
		},
	}
	if err := tx.Create(&safeguards).Error; err != nil {
		return fmt.Errorf("create seed safeguards: %w", err)
	}
	return nil
}
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("obtain database connection for close: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}
