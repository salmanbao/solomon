package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	contentlibrarymarketplace "solomon/contexts/campaign-editorial/content-library-marketplace"
	postgresadapter "solomon/contexts/campaign-editorial/content-library-marketplace/adapters/postgres"
	workerapp "solomon/contexts/campaign-editorial/content-library-marketplace/application/workers"
	authorization "solomon/contexts/identity-access/authorization-service"
	authmemory "solomon/contexts/identity-access/authorization-service/adapters/memory"
	authpostgres "solomon/contexts/identity-access/authorization-service/adapters/postgres"
	"solomon/internal/platform/config"
	"solomon/internal/platform/db"
	"solomon/internal/platform/httpserver"
	"solomon/internal/platform/messaging"
)

// Package bootstrap is the composition root.
// Keep construction/wiring here so module code stays framework-agnostic.

type APIApp struct {
	server   *httpserver.Server
	postgres *db.Postgres
	logger   *slog.Logger
}

type WorkerApp struct {
	postgres     *db.Postgres
	outboxRelay  workerapp.OutboxRelay
	distribution workerapp.DistributionStatusConsumer
	expirer      workerapp.ClaimExpirer
	pollInterval time.Duration
	logger       *slog.Logger
}

func BuildAPI() (*APIApp, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger := slog.Default().With("service", cfg.ServiceName, "process", "api")
	if strings.TrimSpace(cfg.PostgresDSN) == "" {
		return nil, errors.New("POSTGRES_DSN is required")
	}

	pg, err := db.Connect(cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	repo := postgresadapter.NewRepository(pg.DB, logger)
	module := contentlibrarymarketplace.NewModule(contentlibrarymarketplace.Dependencies{
		Clips:          repo,
		Claims:         repo,
		Downloads:      repo,
		Idempotency:    repo,
		Clock:          postgresadapter.SystemClock{},
		IDGenerator:    postgresadapter.UUIDGenerator{},
		ClaimTTL:       24 * time.Hour,
		IdempotencyTTL: 7 * 24 * time.Hour,
		PreviewTTL:     15 * time.Minute,
		DownloadTTL:    24 * time.Hour,
		DownloadLimit:  5,
		Logger:         logger,
	})

	authRepo := authpostgres.NewRepository(pg.DB, logger)
	authCache := authmemory.NewStore()
	authModule := authorization.NewModule(authorization.Dependencies{
		Repository:         authRepo,
		Idempotency:        authRepo,
		PermissionCache:    authCache,
		Clock:              authpostgres.SystemClock{},
		IDGenerator:        authpostgres.UUIDGenerator{},
		IdempotencyTTL:     7 * 24 * time.Hour,
		PermissionCacheTTL: 5 * time.Minute,
		Logger:             logger,
	})

	server := httpserver.New(module, authModule, logger, normalizeAddr(cfg.HTTPPort))
	return &APIApp{
		server:   server,
		postgres: pg,
		logger:   logger,
	}, nil
}

func BuildWorker() (*WorkerApp, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger := slog.Default().With("service", cfg.ServiceName, "process", "worker")
	if strings.TrimSpace(cfg.PostgresDSN) == "" {
		return nil, errors.New("POSTGRES_DSN is required")
	}

	pg, err := db.Connect(cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	kafka, err := messaging.NewKafka(cfg.KafkaBrokers, logger)
	if err != nil {
		return nil, err
	}

	repo := postgresadapter.NewRepository(pg.DB, logger)
	return &WorkerApp{
		postgres: pg,
		outboxRelay: workerapp.OutboxRelay{
			Outbox:    repo,
			Publisher: kafka,
			Clock:     postgresadapter.SystemClock{},
			Topic:     "distribution.claimed",
			BatchSize: 100,
			Logger:    logger,
		},
		distribution: workerapp.DistributionStatusConsumer{
			Subscriber:    kafka,
			Claims:        repo,
			Dedup:         repo,
			Clock:         postgresadapter.SystemClock{},
			ConsumerGroup: "content-marketplace-distribution-cg",
			DedupTTL:      7 * 24 * time.Hour,
			Logger:        logger,
		},
		expirer: workerapp.ClaimExpirer{
			Claims: repo,
			Clock:  postgresadapter.SystemClock{},
			Logger: logger,
		},
		pollInterval: 2 * time.Second,
		logger:       logger,
	}, nil
}

func (a *APIApp) Run(_ context.Context) error {
	if a.logger != nil {
		a.logger.Info("api app started",
			"event", "bootstrap_api_started",
			"module", "internal/app/bootstrap",
			"layer", "platform",
		)
	}
	return a.server.Start()
}

func (a *APIApp) Close() error {
	if a.postgres != nil {
		return a.postgres.Close()
	}
	return nil
}

func (w *WorkerApp) Run(ctx context.Context) error {
	if err := w.distribution.Start(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.logger.Info("worker app started",
		"event", "bootstrap_worker_started",
		"module", "internal/app/bootstrap",
		"layer", "platform",
		"poll_interval", w.pollInterval.String(),
	)

	for {
		if err := w.expirer.RunOnce(ctx); err != nil {
			return err
		}
		if err := w.outboxRelay.RunOnce(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (w *WorkerApp) Close() error {
	if w.postgres != nil {
		return w.postgres.Close()
	}
	return nil
}

func normalizeAddr(port string) string {
	value := strings.TrimSpace(port)
	if value == "" {
		return ":8080"
	}
	if strings.HasPrefix(value, ":") {
		return value
	}
	return ":" + value
}
