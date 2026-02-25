package campaignservice

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/campaign-editorial/campaign-service/adapters/http"
	"solomon/contexts/campaign-editorial/campaign-service/adapters/memory"
	"solomon/contexts/campaign-editorial/campaign-service/application/commands"
	"solomon/contexts/campaign-editorial/campaign-service/application/queries"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Campaigns      ports.CampaignRepository
	Media          ports.MediaRepository
	History        ports.HistoryRepository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func NewModule(deps Dependencies) Module {
	createCampaign := commands.CreateCampaignUseCase{
		Campaigns:      deps.Campaigns,
		Idempotency:    deps.Idempotency,
		Clock:          deps.Clock,
		IDGenerator:    deps.IDGenerator,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}
	updateCampaign := commands.UpdateCampaignUseCase{
		Campaigns: deps.Campaigns,
		Clock:     deps.Clock,
		Logger:    deps.Logger,
	}
	changeStatus := commands.ChangeStatusUseCase{
		Campaigns: deps.Campaigns,
		History:   deps.History,
		Clock:     deps.Clock,
		IDGen:     deps.IDGenerator,
		Logger:    deps.Logger,
	}
	increaseBudget := commands.IncreaseBudgetUseCase{
		Campaigns: deps.Campaigns,
		History:   deps.History,
		Clock:     deps.Clock,
		IDGen:     deps.IDGenerator,
		Logger:    deps.Logger,
	}
	generateUploadURL := commands.GenerateUploadURLUseCase{
		Campaigns: deps.Campaigns,
		Clock:     deps.Clock,
		IDGen:     deps.IDGenerator,
		Logger:    deps.Logger,
	}
	confirmMedia := commands.ConfirmMediaUseCase{
		Campaigns: deps.Campaigns,
		Media:     deps.Media,
		Clock:     deps.Clock,
		Logger:    deps.Logger,
	}

	listCampaigns := queries.ListCampaignsUseCase{
		Campaigns: deps.Campaigns,
		Logger:    deps.Logger,
	}
	getCampaign := queries.GetCampaignUseCase{
		Campaigns: deps.Campaigns,
		Logger:    deps.Logger,
	}
	listMedia := queries.ListMediaUseCase{
		Media:  deps.Media,
		Logger: deps.Logger,
	}
	getAnalytics := queries.GetAnalyticsUseCase{
		Campaigns: deps.Campaigns,
		Clock:     deps.Clock,
		Logger:    deps.Logger,
	}
	exportAnalytics := queries.ExportAnalyticsUseCase{
		Clock:  deps.Clock,
		Logger: deps.Logger,
	}

	return Module{
		Handler: httpadapter.Handler{
			CreateCampaign:    createCampaign,
			UpdateCampaign:    updateCampaign,
			ChangeStatus:      changeStatus,
			IncreaseBudget:    increaseBudget,
			GenerateUploadURL: generateUploadURL,
			ConfirmMedia:      confirmMedia,
			ListCampaigns:     listCampaigns,
			GetCampaign:       getCampaign,
			ListMedia:         listMedia,
			GetAnalytics:      getAnalytics,
			ExportAnalytics:   exportAnalytics,
			Logger:            deps.Logger,
		},
	}
}

func NewInMemoryModule(seed []entities.Campaign, logger *slog.Logger) Module {
	store := memory.NewStore(seed)
	module := NewModule(Dependencies{
		Campaigns:      store,
		Media:          store,
		History:        store,
		Idempotency:    store,
		Clock:          store,
		IDGenerator:    store,
		IdempotencyTTL: 7 * 24 * time.Hour,
		Logger:         logger,
	})
	module.Store = store
	return module
}
