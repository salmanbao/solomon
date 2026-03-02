# Clipping Tool Service (M24)

Monolith implementation for `M24-Clipping-Tool-Service`.

## Dependency Alignment
- Canonical provider dependency: `M06-Media-Processing-Pipeline` (`owner_api`)
- Canonical consumer dependency: `M25-Auto-Clipping-AI`

`M06` integration is isolated behind `MediaProcessingClient`, preserving
runtime boundaries between Solomon and mesh.

## Data Ownership
Owned tables (canonical):
- `editor_usage_analytics`
- `export_campaign_submissions`
- `export_jobs`
- `project_timelines`
- `project_versions`
- `user_projects`

Mutation endpoints require idempotency and use canonical error envelopes.
