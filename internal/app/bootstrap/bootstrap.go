package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	campaignservice "solomon/contexts/campaign-editorial/campaign-service"
	campaignpostgres "solomon/contexts/campaign-editorial/campaign-service/adapters/postgres"
	campaignworkers "solomon/contexts/campaign-editorial/campaign-service/application/workers"
	contentlibrarymarketplace "solomon/contexts/campaign-editorial/content-library-marketplace"
	postgresadapter "solomon/contexts/campaign-editorial/content-library-marketplace/adapters/postgres"
	workerapp "solomon/contexts/campaign-editorial/content-library-marketplace/application/workers"
	distributionservice "solomon/contexts/campaign-editorial/distribution-service"
	distributionpostgres "solomon/contexts/campaign-editorial/distribution-service/adapters/postgres"
	distributioncommands "solomon/contexts/campaign-editorial/distribution-service/application/commands"
	distributionworkers "solomon/contexts/campaign-editorial/distribution-service/application/workers"
	submissionservice "solomon/contexts/campaign-editorial/submission-service"
	submissionpostgres "solomon/contexts/campaign-editorial/submission-service/adapters/postgres"
	submissionworkers "solomon/contexts/campaign-editorial/submission-service/application/workers"
	votingengine "solomon/contexts/campaign-editorial/voting-engine"
	votingpostgres "solomon/contexts/campaign-editorial/voting-engine/adapters/postgres"
	votingworkers "solomon/contexts/campaign-editorial/voting-engine/application/workers"
	authorization "solomon/contexts/identity-access/authorization-service"
	authevents "solomon/contexts/identity-access/authorization-service/adapters/events"
	authmemory "solomon/contexts/identity-access/authorization-service/adapters/memory"
	authpostgres "solomon/contexts/identity-access/authorization-service/adapters/postgres"
	authworkers "solomon/contexts/identity-access/authorization-service/application/workers"
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
	postgres             *db.Postgres
	outboxRelay          workerapp.OutboxRelay
	distribution         workerapp.DistributionStatusConsumer
	distributionClaims   distributionworkers.ClaimedConsumer
	distributionOutbox   distributionworkers.OutboxRelay
	distributionSchedule distributionworkers.SchedulerJob
	expirer              workerapp.ClaimExpirer
	campaignOutbox       campaignworkers.OutboxRelay
	campaignSubmission   campaignworkers.SubmissionCreatedConsumer
	campaignDeadlineJob  campaignworkers.DeadlineCompleter
	submissionOutbox     submissionworkers.OutboxRelay
	submissionLaunch     submissionworkers.CampaignLaunchedConsumer
	submissionAuto       submissionworkers.AutoApproveJob
	submissionViewLock   submissionworkers.ViewLockJob
	authzOutbox          authworkers.OutboxRelay
	votingOutbox         votingworkers.OutboxRelay
	votingSubmission     votingworkers.SubmissionLifecycleConsumer
	votingCampaign       votingworkers.CampaignStateConsumer
	pollInterval         time.Duration
	logger               *slog.Logger
}

func BuildAPI() (*APIApp, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger := slog.Default().With("service", cfg.ServiceName, "process", "api")
	if strings.TrimSpace(cfg.PostgresDSN) == "" {
		return nil, errors.New("POSTGRES_DSN is required")
	}

	pg, err := db.Connect(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
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

	campaignRepo := campaignpostgres.NewRepository(pg.DB, logger)
	submissionRepo := submissionpostgres.NewRepository(pg.DB, logger)
	campaignModule := campaignservice.NewModule(campaignservice.Dependencies{
		Campaigns:      campaignRepo,
		Media:          campaignRepo,
		History:        campaignRepo,
		Idempotency:    campaignRepo,
		Outbox:         campaignRepo,
		Clock:          campaignpostgres.SystemClock{},
		IDGenerator:    campaignpostgres.UUIDGenerator{},
		IdempotencyTTL: 7 * 24 * time.Hour,
		Logger:         logger,
	})
	submissionModule := submissionservice.NewModule(submissionservice.Dependencies{
		Repository:     submissionRepo,
		Campaigns:      submissionRepo,
		Idempotency:    submissionRepo,
		Outbox:         submissionRepo,
		Clock:          submissionpostgres.SystemClock{},
		IDGen:          submissionpostgres.UUIDGenerator{},
		IdempotencyTTL: 7 * 24 * time.Hour,
		Logger:         logger,
	})
	distributionRepo := distributionpostgres.NewRepository(pg.DB, logger)
	distributionModule := distributionservice.NewModule(distributionservice.Dependencies{
		Repository: distributionRepo,
		Clock:      distributionpostgres.SystemClock{},
		IDGen:      distributionpostgres.UUIDGenerator{},
		Outbox:     distributionRepo,
		Logger:     logger,
	})
	votingRepo := votingpostgres.NewRepository(pg.DB, logger)
	votingModule := votingengine.NewModule(votingengine.Dependencies{
		Votes:          votingRepo,
		Idempotency:    votingRepo,
		Outbox:         votingRepo,
		Clock:          votingpostgres.SystemClock{},
		IDGen:          votingpostgres.UUIDGenerator{},
		IdempotencyTTL: 7 * 24 * time.Hour,
		Logger:         logger,
	})

	server := httpserver.New(
		module,
		authModule,
		campaignModule,
		submissionModule,
		distributionModule,
		votingModule,
		logger,
		normalizeAddr(cfg.HTTPPort),
	)
	return &APIApp{
		server:   server,
		postgres: pg,
		logger:   logger,
	}, nil
}

func BuildWorker() (*WorkerApp, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger := slog.Default().With("service", cfg.ServiceName, "process", "worker")
	if strings.TrimSpace(cfg.PostgresDSN) == "" {
		return nil, errors.New("POSTGRES_DSN is required")
	}

	pg, err := db.Connect(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	kafka, err := messaging.NewKafka(cfg.KafkaBrokers, logger)
	if err != nil {
		_ = pg.Close()
		return nil, fmt.Errorf("init messaging adapter: %w", err)
	}

	marketplaceRepo := postgresadapter.NewRepository(pg.DB, logger)
	campaignRepo := campaignpostgres.NewRepository(pg.DB, logger)
	submissionRepo := submissionpostgres.NewRepository(pg.DB, logger)
	votingRepo := votingpostgres.NewRepository(pg.DB, logger)
	authRepo := authpostgres.NewRepository(pg.DB, logger)
	distributionRepo := distributionpostgres.NewRepository(pg.DB, logger)
	distributionCommands := distributioncommands.UseCase{
		Repository: distributionRepo,
		Clock:      distributionpostgres.SystemClock{},
		IDGen:      distributionpostgres.UUIDGenerator{},
		Outbox:     distributionRepo,
		Logger:     logger,
	}
	authPublisher := authevents.NewKafkaPublisher(kafka, logger, "authz.policy_changed")
	return &WorkerApp{
		postgres: pg,
		outboxRelay: workerapp.OutboxRelay{
			Outbox:    marketplaceRepo,
			Publisher: kafka,
			Clock:     postgresadapter.SystemClock{},
			Topic:     "distribution.claimed",
			BatchSize: 100,
			Logger:    logger,
		},
		distribution: workerapp.DistributionStatusConsumer{
			Subscriber:    kafka,
			Claims:        marketplaceRepo,
			Dedup:         marketplaceRepo,
			Clock:         postgresadapter.SystemClock{},
			ConsumerGroup: "content-marketplace-distribution-cg",
			DedupTTL:      7 * 24 * time.Hour,
			Logger:        logger,
		},
		distributionClaims: distributionworkers.ClaimedConsumer{
			Subscriber:    kafka,
			Repository:    distributionRepo,
			Clock:         distributionpostgres.SystemClock{},
			IDGen:         distributionpostgres.UUIDGenerator{},
			ConsumerGroup: "distribution-service-claimed-cg",
			Logger:        logger,
		},
		distributionOutbox: distributionworkers.OutboxRelay{
			Outbox:    distributionRepo,
			Publisher: kafka,
			Clock:     distributionpostgres.SystemClock{},
			BatchSize: 100,
			Logger:    logger,
		},
		distributionSchedule: distributionworkers.SchedulerJob{
			Commands:  distributionCommands,
			BatchSize: 100,
			Logger:    logger,
		},
		expirer: workerapp.ClaimExpirer{
			Claims: marketplaceRepo,
			Clock:  postgresadapter.SystemClock{},
			Logger: logger,
		},
		campaignOutbox: campaignworkers.OutboxRelay{
			Outbox:    campaignRepo,
			Publisher: kafka,
			Clock:     campaignpostgres.SystemClock{},
			BatchSize: 100,
			Logger:    logger,
		},
		campaignSubmission: campaignworkers.SubmissionCreatedConsumer{
			Subscriber:    kafka,
			Campaigns:     campaignRepo,
			Dedup:         campaignRepo,
			Clock:         campaignpostgres.SystemClock{},
			ConsumerGroup: "campaign-service-submission-created-cg",
			DedupTTL:      7 * 24 * time.Hour,
			Logger:        logger,
		},
		submissionOutbox: submissionworkers.OutboxRelay{
			Outbox:    submissionRepo,
			Publisher: kafka,
			Clock:     submissionpostgres.SystemClock{},
			BatchSize: 100,
			Logger:    logger,
		},
		submissionLaunch: submissionworkers.CampaignLaunchedConsumer{
			Subscriber:    kafka,
			Dedup:         submissionRepo,
			Clock:         submissionpostgres.SystemClock{},
			ConsumerGroup: "submission-service-campaign-launched-cg",
			DedupTTL:      7 * 24 * time.Hour,
			Logger:        logger,
		},
		submissionAuto: submissionworkers.AutoApproveJob{
			Repository:  submissionRepo,
			AutoApprove: submissionRepo,
			Clock:       submissionpostgres.SystemClock{},
			IDGen:       submissionpostgres.UUIDGenerator{},
			Outbox:      submissionRepo,
			BatchSize:   100,
			Logger:      logger,
		},
		submissionViewLock: submissionworkers.ViewLockJob{
			Repository:      submissionRepo,
			ViewLock:        submissionRepo,
			Clock:           submissionpostgres.SystemClock{},
			IDGen:           submissionpostgres.UUIDGenerator{},
			Outbox:          submissionRepo,
			BatchSize:       100,
			PlatformFeeRate: 0.15,
			Logger:          logger,
		},
		campaignDeadlineJob: campaignworkers.DeadlineCompleter{
			Campaigns: campaignRepo,
			Clock:     campaignpostgres.SystemClock{},
			BatchSize: 100,
			Logger:    logger,
		},
		authzOutbox: authworkers.OutboxRelay{
			Outbox:    authRepo,
			Publisher: authPublisher,
			Clock:     authpostgres.SystemClock{},
			BatchSize: 100,
			Logger:    logger,
		},
		votingOutbox: votingworkers.OutboxRelay{
			Outbox:    votingRepo,
			Publisher: kafka,
			Clock:     votingpostgres.SystemClock{},
			BatchSize: 100,
			Logger:    logger,
		},
		votingSubmission: votingworkers.SubmissionLifecycleConsumer{
			Subscriber:    kafka,
			Dedup:         votingRepo,
			Votes:         votingRepo,
			Outbox:        votingRepo,
			Clock:         votingpostgres.SystemClock{},
			IDGen:         votingpostgres.UUIDGenerator{},
			ConsumerGroup: "voting-engine-submission-cg",
			DedupTTL:      7 * 24 * time.Hour,
			Logger:        logger,
		},
		votingCampaign: votingworkers.CampaignStateConsumer{
			Subscriber:    kafka,
			Dedup:         votingRepo,
			Votes:         votingRepo,
			Outbox:        votingRepo,
			Clock:         votingpostgres.SystemClock{},
			IDGen:         votingpostgres.UUIDGenerator{},
			ConsumerGroup: "voting-engine-campaign-cg",
			DedupTTL:      7 * 24 * time.Hour,
			Logger:        logger,
		},
		pollInterval: 500 * time.Millisecond,
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
	if a.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
	}
	if a.postgres != nil {
		if err := a.postgres.Close(); err != nil {
			return fmt.Errorf("close postgres: %w", err)
		}
	}
	return nil
}

func (w *WorkerApp) Run(ctx context.Context) error {
	logger := w.logger
	if logger == nil {
		logger = slog.Default()
	}

	if err := w.distribution.Start(ctx); err != nil {
		return fmt.Errorf("start marketplace distribution consumer: %w", err)
	}
	if err := w.distributionClaims.Start(ctx); err != nil {
		return fmt.Errorf("start distribution claimed consumer: %w", err)
	}
	if err := w.campaignSubmission.Start(ctx); err != nil {
		return fmt.Errorf("start campaign submission consumer: %w", err)
	}
	if err := w.submissionLaunch.Start(ctx); err != nil {
		return fmt.Errorf("start submission campaign launch consumer: %w", err)
	}
	if err := w.votingSubmission.Start(ctx); err != nil {
		return fmt.Errorf("start voting submission lifecycle consumer: %w", err)
	}
	if err := w.votingCampaign.Start(ctx); err != nil {
		return fmt.Errorf("start voting campaign state consumer: %w", err)
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	logger.Info("worker app started",
		"event", "bootstrap_worker_started",
		"module", "internal/app/bootstrap",
		"layer", "platform",
		"poll_interval", w.pollInterval.String(),
	)

	for {
		if err := w.expirer.RunOnce(ctx); err != nil {
			return fmt.Errorf("run marketplace claim expirer: %w", err)
		}
		if err := w.outboxRelay.RunOnce(ctx); err != nil {
			return fmt.Errorf("run marketplace outbox relay: %w", err)
		}
		if err := w.distributionSchedule.RunOnce(ctx); err != nil {
			return fmt.Errorf("run distribution schedule job: %w", err)
		}
		if err := w.distributionOutbox.RunOnce(ctx); err != nil {
			return fmt.Errorf("run distribution outbox relay: %w", err)
		}
		if err := w.campaignDeadlineJob.RunOnce(ctx); err != nil {
			return fmt.Errorf("run campaign deadline completer: %w", err)
		}
		if err := w.campaignOutbox.RunOnce(ctx); err != nil {
			return fmt.Errorf("run campaign outbox relay: %w", err)
		}
		if err := w.submissionAuto.RunOnce(ctx); err != nil {
			return fmt.Errorf("run submission auto-approve job: %w", err)
		}
		if err := w.submissionViewLock.RunOnce(ctx); err != nil {
			return fmt.Errorf("run submission view-lock job: %w", err)
		}
		if err := w.submissionOutbox.RunOnce(ctx); err != nil {
			return fmt.Errorf("run submission outbox relay: %w", err)
		}
		if err := w.authzOutbox.RunOnce(ctx); err != nil {
			return fmt.Errorf("run authz outbox relay: %w", err)
		}
		if err := w.votingOutbox.RunOnce(ctx); err != nil {
			return fmt.Errorf("run voting outbox relay: %w", err)
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
