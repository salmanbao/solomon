package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/campaign-discovery-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-discovery-service/ports"
)

type Service struct {
	Repo               ports.Repository
	Idempotency        ports.IdempotencyStore
	CampaignProjection ports.CampaignProjectionProvider
	ReputationProvider ports.ReputationProjectionProvider
	Clock              ports.Clock
	IdempotencyTTL     time.Duration
	Logger             *slog.Logger
}

func (s Service) BrowseCampaigns(ctx context.Context, query ports.BrowseQuery) (ports.BrowseResult, error) {
	query.UserID = strings.TrimSpace(query.UserID)
	if query.UserID == "" {
		return ports.BrowseResult{}, domainerrors.ErrInvalidRequest
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	query.SortBy = strings.ToLower(strings.TrimSpace(query.SortBy))
	switch query.SortBy {
	case "", "popularity", "budget", "deadline", "relevance":
	default:
		return ports.BrowseResult{}, domainerrors.ErrInvalidRequest
	}
	if query.Filters.BudgetMin > 0 && query.Filters.BudgetMax > 0 && query.Filters.BudgetMin > query.Filters.BudgetMax {
		return ports.BrowseResult{}, domainerrors.ErrInvalidRequest
	}

	result, err := s.Repo.BrowseCampaigns(ctx, query)
	if err != nil {
		return ports.BrowseResult{}, err
	}
	campaigns, err := s.enrichCampaigns(ctx, result.Campaigns)
	if err != nil {
		return ports.BrowseResult{}, err
	}
	result.Campaigns = campaigns
	return result, nil
}

func (s Service) SearchCampaigns(ctx context.Context, query ports.SearchQuery) (ports.SearchResult, error) {
	query.UserID = strings.TrimSpace(query.UserID)
	query.Query = strings.TrimSpace(query.Query)
	if query.UserID == "" || query.Query == "" {
		return ports.SearchResult{}, domainerrors.ErrInvalidRequest
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		return ports.SearchResult{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.SearchCampaigns(ctx, query)
}

func (s Service) GetCampaignDetails(ctx context.Context, userID string, campaignID string) (ports.CampaignDetails, error) {
	userID = strings.TrimSpace(userID)
	campaignID = strings.TrimSpace(campaignID)
	if userID == "" || campaignID == "" {
		return ports.CampaignDetails{}, domainerrors.ErrInvalidRequest
	}
	details, err := s.Repo.GetCampaignDetails(ctx, userID, campaignID)
	if err != nil {
		return ports.CampaignDetails{}, err
	}
	items, err := s.enrichCampaigns(ctx, []ports.CampaignSummary{details.Campaign})
	if err != nil {
		return ports.CampaignDetails{}, err
	}
	if len(items) == 1 {
		details.Campaign = items[0]
	}
	return details, nil
}

func (s Service) SaveBookmark(ctx context.Context, idempotencyKey string, command ports.BookmarkCommand) (ports.BookmarkRecord, error) {
	command.UserID = strings.TrimSpace(command.UserID)
	command.CampaignID = strings.TrimSpace(command.CampaignID)
	command.Tag = strings.TrimSpace(command.Tag)
	command.Note = strings.TrimSpace(command.Note)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if command.UserID == "" || command.CampaignID == "" {
		return ports.BookmarkRecord{}, domainerrors.ErrInvalidRequest
	}
	if idempotencyKey == "" {
		return ports.BookmarkRecord{}, domainerrors.ErrIdempotencyKeyRequired
	}

	requestHash := hashStrings(command.UserID, command.CampaignID, command.Tag, command.Note)
	var output ports.BookmarkRecord
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &output) },
		func() ([]byte, error) {
			item, err := s.Repo.SaveBookmark(ctx, command, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return output, err
}

func (s Service) enrichCampaigns(ctx context.Context, campaigns []ports.CampaignSummary) ([]ports.CampaignSummary, error) {
	if len(campaigns) == 0 {
		return campaigns, nil
	}

	campaignIDs := make([]string, 0, len(campaigns))
	creatorIDs := make([]string, 0, len(campaigns))
	campaignSeen := make(map[string]struct{}, len(campaigns))
	creatorSeen := make(map[string]struct{}, len(campaigns))
	for _, item := range campaigns {
		if _, ok := campaignSeen[item.CampaignID]; !ok {
			campaignSeen[item.CampaignID] = struct{}{}
			campaignIDs = append(campaignIDs, item.CampaignID)
		}
		if _, ok := creatorSeen[item.CreatorName]; !ok && item.CreatorName != "" {
			creatorSeen[item.CreatorName] = struct{}{}
			creatorIDs = append(creatorIDs, item.CreatorName)
		}
	}

	projections := map[string]ports.CampaignProjection{}
	if s.CampaignProjection != nil {
		item, err := s.CampaignProjection.GetCampaignProjections(ctx, campaignIDs)
		if err != nil {
			return nil, domainerrors.ErrDependencyUnavailable
		}
		projections = item
	}

	tiers := map[string]string{}
	if s.ReputationProvider != nil && len(creatorIDs) > 0 {
		item, err := s.ReputationProvider.GetCreatorTiers(ctx, creatorIDs)
		if err != nil {
			return nil, domainerrors.ErrDependencyUnavailable
		}
		tiers = item
	}

	out := make([]ports.CampaignSummary, len(campaigns))
	for idx, item := range campaigns {
		if projection, ok := projections[item.CampaignID]; ok {
			item.State = projection.State
			if projection.BudgetRemaining >= 0 {
				item.BudgetSpent = item.BudgetTotal - projection.BudgetRemaining
				if item.BudgetSpent < 0 {
					item.BudgetSpent = 0
				}
			}
			if projection.SubmissionCount >= 0 {
				item.SubmissionCount = projection.SubmissionCount
			}
		}
		if tier, ok := tiers[item.CreatorName]; ok && strings.TrimSpace(tier) != "" {
			item.CreatorTier = strings.TrimSpace(tier)
		}
		if item.BudgetSpent < item.BudgetTotal {
			item.IsEligible = true
			item.Eligibility = ""
		} else {
			item.IsEligible = false
			item.Eligibility = "campaign budget exhausted"
		}
		out[idx] = item
	}
	return out, nil
}

func (s Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	execute func() ([]byte, error),
) error {
	now := s.now()
	record, found, err := s.Idempotency.Get(ctx, key, now)
	if err != nil {
		return err
	}
	if found {
		if record.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(record.Payload)
	}

	payload, err := execute()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Payload:     payload,
		ExpiresAt:   now.Add(s.idempotencyTTL()),
	}); err != nil {
		return err
	}

	resolveLogger(s.Logger).Debug("campaign discovery idempotent mutation committed",
		"event", "campaign_discovery_idempotent_mutation_committed",
		"module", "campaign-editorial/campaign-discovery-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
