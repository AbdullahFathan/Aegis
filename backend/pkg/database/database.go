package database

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func OpenPostgres(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_DSN is empty")
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}

func OpenSQLite(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Region{},
		&Location{},
		&Area{},
		&Incident{},
		&IncidentNumberCounter{},
		&IncidentWorkflowLog{},
		&RootCauseAnalysis{},
		&CorrectiveAction{},
		&IncidentFile{},
		&Notification{},
		&AuditLog{},
	)
}

func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
