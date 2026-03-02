package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	discoveryhttp "solomon/contexts/campaign-editorial/campaign-discovery-service/transport/http"
)

type discoverFeedResponse struct {
	Status     string             `json:"status"`
	Data       []discoverFeedItem `json:"data"`
	NextCursor string             `json:"next_cursor,omitempty"`
	HasMore    bool               `json:"has_more"`
	Timestamp  string             `json:"timestamp"`
}

type discoverFeedItem struct {
	ItemID          string  `json:"item_id"`
	ItemType        string  `json:"item_type"`
	Title           string  `json:"title"`
	RankScore       float64 `json:"rank_score"`
	BudgetTotal     float64 `json:"budget_total"`
	RatePer1KViews  float64 `json:"rate_per_1k_views"`
	SubmissionCount int     `json:"submission_count"`
	Category        string  `json:"category"`
}

func (s *Server) handleDiscoverRoot(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if strings.TrimSpace(query.Get("tab")) != "" || strings.TrimSpace(query.Get("cursor")) != "" {
		s.handleDiscoverFeed(w, r)
		return
	}
	s.handleProductDiscover(w, r)
}

func (s *Server) handleDiscoverFeed(w http.ResponseWriter, r *http.Request) {
	if !requireDiscoverAuthorization(w, r) || !requireDiscoverRequestID(w, r) {
		return
	}
	userID, ok := requireDiscoverUser(w, r)
	if !ok {
		return
	}

	tab := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("tab")))
	if tab == "" {
		tab = "all"
	}
	if tab != "all" && tab != "campaigns" {
		writeDiscoverError(w, http.StatusBadRequest, "INVALID_REQUEST", "tab must be one of: all, campaigns", nil)
		return
	}

	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 100 {
			writeDiscoverError(w, http.StatusBadRequest, "INVALID_REQUEST", "limit must be an integer between 1 and 100", nil)
			return
		}
		limit = parsed
	}

	resp, err := s.campaignDiscovery.Handler.BrowseCampaignsHandler(r.Context(), userID, discoveryhttp.BrowseCampaignsRequest{
		PageSize: strconv.Itoa(limit),
		Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		SortBy:   "relevance",
		State:    "active",
	})
	if err != nil {
		writeDiscoverDomainError(w, err)
		return
	}

	items := make([]discoverFeedItem, 0, len(resp.Data.Campaigns))
	for _, campaign := range resp.Data.Campaigns {
		items = append(items, discoverFeedItem{
			ItemID:          campaign.CampaignID,
			ItemType:        "campaign",
			Title:           campaign.Title,
			RankScore:       campaign.MatchScore,
			BudgetTotal:     campaign.BudgetTotal,
			RatePer1KViews:  campaign.RatePer1KViews,
			SubmissionCount: campaign.SubmissionCount,
			Category:        campaign.Category,
		})
	}

	writeJSON(w, http.StatusOK, discoverFeedResponse{
		Status:     "success",
		Data:       items,
		NextCursor: resp.Data.Pagination.NextCursor,
		HasMore:    resp.Data.Pagination.HasNext,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})
}
