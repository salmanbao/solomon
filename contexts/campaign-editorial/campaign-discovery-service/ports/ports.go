package ports

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Payload     []byte
	ExpiresAt   time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

type CampaignProjection struct {
	CampaignID      string
	State           string
	BudgetRemaining float64
	SubmissionCount int
}

type CampaignProjectionProvider interface {
	GetCampaignProjections(ctx context.Context, campaignIDs []string) (map[string]CampaignProjection, error)
}

type ReputationProjectionProvider interface {
	GetCreatorTiers(ctx context.Context, creatorIDs []string) (map[string]string, error)
}

type BrowseFilters struct {
	Category        string
	BudgetMin       float64
	BudgetMax       float64
	DeadlineAfter   *time.Time
	DeadlineBefore  *time.Time
	Platforms       []string
	State           string
	ExcludeFeatured bool
}

type BrowseQuery struct {
	UserID   string
	PageSize int
	Cursor   string
	SortBy   string
	Filters  BrowseFilters
}

type SearchQuery struct {
	UserID    string
	Query     string
	Category  string
	BudgetMin float64
	Limit     int
	Offset    int
}

type CampaignSummary struct {
	CampaignID       string
	Title            string
	Description      string
	CreatorName      string
	CreatorTier      string
	BudgetTotal      float64
	BudgetSpent      float64
	BudgetCurrency   string
	RatePer1KViews   float64
	EstimatedViews   int
	EstimatedEarning float64
	SubmissionCount  int
	ApprovalRate     float64
	Deadline         string
	Category         string
	Platforms        []string
	State            string
	IsFeatured       bool
	FeaturedUntil    string
	MatchScore       float64
	IsEligible       bool
	Eligibility      string
	UserSaved        bool
	TrendingStatus   string
	CreatedAt        string
	CombinedScore    float64
}

type BrowseSummary struct {
	ResultCount  int
	SearchTimeMS int
	CacheHit     bool
}

type Pagination struct {
	NextCursor     string
	PrevCursor     string
	HasNext        bool
	HasPrev        bool
	TotalEstimated int
	PageSize       int
}

type BrowseResult struct {
	Campaigns  []CampaignSummary
	Pagination Pagination
	Summary    BrowseSummary
}

type SearchResultItem struct {
	CampaignID      string
	Title           string
	Description     string
	CreatorName     string
	MatchScore      float64
	Budget          float64
	Deadline        string
	Category        string
	SubmissionCount int
	IsFeatured      bool
}

type SearchResult struct {
	Items         []SearchResultItem
	Total         int
	Limit         int
	Offset        int
	HasNext       bool
	ExecutionTime int
	IndexVersion  string
}

type CampaignDetails struct {
	Campaign CampaignSummary
}

type BookmarkCommand struct {
	UserID     string
	CampaignID string
	Tag        string
	Note       string
}

type BookmarkRecord struct {
	BookmarkID string
	UserID     string
	CampaignID string
	Tag        string
	Note       string
	CreatedAt  time.Time
}

type Repository interface {
	BrowseCampaigns(ctx context.Context, query BrowseQuery) (BrowseResult, error)
	SearchCampaigns(ctx context.Context, query SearchQuery) (SearchResult, error)
	GetCampaignDetails(ctx context.Context, userID string, campaignID string) (CampaignDetails, error)
	SaveBookmark(ctx context.Context, command BookmarkCommand, now time.Time) (BookmarkRecord, error)
}
