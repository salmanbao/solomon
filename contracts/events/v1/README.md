# Event Contracts v1

Versioned event payload contracts for implemented Solomon modules.

## M09 Content Library Marketplace
- `distribution.claimed.schema.json` (emitted)
- `distribution.published.schema.json` (consumed)
- `distribution.failed.schema.json` (consumed)

## M21 Authorization Service
- `authz.policy_changed.schema.json` (emitted)

## M26 Submission Service
- `submission.created.schema.json` (emitted)
- `submission.approved.schema.json` (emitted)
- `submission.rejected.schema.json` (emitted)
- `submission.flagged.schema.json` (emitted)
- `submission.auto_approved.schema.json` (emitted)
- `submission.verified.schema.json` (emitted)
- `submission.view_locked.schema.json` (emitted)
- `submission.cancelled.schema.json` (reserved; emitted when cancellation workflow is enabled)

## M08 Voting Engine
- `vote.created.schema.json` (emitted)
- `vote.updated.schema.json` (emitted)
- `vote.retracted.schema.json` (emitted)
- `voting_round.closed.schema.json` (emitted)

## Legacy
- `authorization.role_assigned.schema.json` is kept for backward compatibility with older consumers.
