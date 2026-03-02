# Campaign Discovery Service (M23)

Monolith implementation for `M23-Campaign-Discovery-Service` with canonical
dependency and ownership alignment.

## Dependency Alignment
- DBR providers: `M04-Campaign-Service`, `M48-Reputation-Service`
- Consumer-facing API surface for `M53-Discover-Service` and
  `M58-Content-Recommendation-Engine`
- M53 runtime routes (`/api/v1/discover/feed`, `/api/v1/discover?tab=...`) are
  adapter views over this module to avoid duplicate ownership logic.

Provider usage is isolated behind module ports (`CampaignProjectionProvider`,
`ReputationProjectionProvider`) so this module stays boundary-safe.

## Data Ownership
Owned tables (canonical):
- `campaign_eligibility_cache`
- `campaign_ranking_scores`
- `discover_audit_log`
- `featured_placements`
- `user_bookmarks`

Mutations (`bookmark`) enforce idempotency and return a canonical error
envelope on failures.
