package admindashboardservice

import (
	"time"

	httpadapter "solomon/contexts/internal-ops/admin-dashboard-service/adapters/http"
	"solomon/contexts/internal-ops/admin-dashboard-service/adapters/memory"
	"solomon/contexts/internal-ops/admin-dashboard-service/application"
	"solomon/contexts/internal-ops/admin-dashboard-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository             ports.Repository
	Idempotency            ports.IdempotencyStore
	AuthorizationClient    ports.AuthorizationClient
	ModerationClient       ports.ModerationClient
	AbusePreventionClient  ports.AbusePreventionClient
	FinanceClient          ports.FinanceClient
	BillingClient          ports.BillingClient
	RewardClient           ports.RewardClient
	AffiliateClient        ports.AffiliateClient
	PayoutClient           ports.PayoutClient
	ResolutionClient       ports.ResolutionClient
	ConsentClient          ports.ConsentClient
	PortabilityClient      ports.PortabilityClient
	RetentionClient        ports.RetentionClient
	LegalClient            ports.LegalClient
	SupportClient          ports.SupportClient
	EditorWorkflowClient   ports.EditorWorkflowClient
	ClippingWorkflowClient ports.ClippingWorkflowClient
	AutoClippingClient     ports.AutoClippingClient
	DeveloperPortalClient  ports.DeveloperPortalClient
	IntegrationHubClient   ports.IntegrationHubClient
	WebhookManagerClient   ports.WebhookManagerClient
	DataMigrationClient    ports.DataMigrationClient
	Clock                  ports.Clock
	IdempotencyTTL         time.Duration
}

func NewModule(deps Dependencies) Module {
	return Module{
		Handler: httpadapter.Handler{
			Service: application.Service{
				Repo:                   deps.Repository,
				Idempotency:            deps.Idempotency,
				AuthorizationClient:    deps.AuthorizationClient,
				ModerationClient:       deps.ModerationClient,
				AbusePreventionClient:  deps.AbusePreventionClient,
				FinanceClient:          deps.FinanceClient,
				BillingClient:          deps.BillingClient,
				RewardClient:           deps.RewardClient,
				AffiliateClient:        deps.AffiliateClient,
				PayoutClient:           deps.PayoutClient,
				ResolutionClient:       deps.ResolutionClient,
				ConsentClient:          deps.ConsentClient,
				PortabilityClient:      deps.PortabilityClient,
				RetentionClient:        deps.RetentionClient,
				LegalClient:            deps.LegalClient,
				SupportClient:          deps.SupportClient,
				EditorWorkflowClient:   deps.EditorWorkflowClient,
				ClippingWorkflowClient: deps.ClippingWorkflowClient,
				AutoClippingClient:     deps.AutoClippingClient,
				DeveloperPortalClient:  deps.DeveloperPortalClient,
				IntegrationHubClient:   deps.IntegrationHubClient,
				WebhookManagerClient:   deps.WebhookManagerClient,
				DataMigrationClient:    deps.DataMigrationClient,
				Clock:                  deps.Clock,
				IdempotencyTTL:         deps.IdempotencyTTL,
			},
		},
	}
}

func NewInMemoryModule() Module {
	store := memory.NewStore()
	module := NewModule(Dependencies{
		Repository:             store,
		Idempotency:            store,
		AuthorizationClient:    store,
		ModerationClient:       store,
		AbusePreventionClient:  store,
		FinanceClient:          store,
		BillingClient:          store,
		RewardClient:           store,
		AffiliateClient:        store,
		PayoutClient:           store,
		ResolutionClient:       store,
		ConsentClient:          store,
		PortabilityClient:      store,
		RetentionClient:        store,
		LegalClient:            store,
		SupportClient:          store,
		EditorWorkflowClient:   store,
		ClippingWorkflowClient: store,
		AutoClippingClient:     store,
		DeveloperPortalClient:  store,
		IntegrationHubClient:   store,
		WebhookManagerClient:   store,
		DataMigrationClient:    store,
		Clock:                  store,
		IdempotencyTTL:         7 * 24 * time.Hour,
	})
	module.Store = store
	return module
}
