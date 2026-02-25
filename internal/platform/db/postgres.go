package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Postgres wraps DB connectivity.
// Keep transaction helpers here to support outbox + state consistency.
type Postgres struct {
	DB *gorm.DB
}

func Connect(dsn string) (*Postgres, error) {
	if dsn == "" {
		return nil, errors.New("postgres dsn is required")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gorm postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve postgres sql db handle: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Postgres{DB: db}, nil
}

func (p *Postgres) Close() error {
	if p == nil || p.DB == nil {
		return nil
	}
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
