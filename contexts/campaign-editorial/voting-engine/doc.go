// Package votingengine implements M08 Voting Engine inside the
// campaign-editorial context.
//
// The module owns vote lifecycle orchestration (create/update/retract),
// weighted leaderboard reads, and vote-related event production/consumption
// through outbox-backed workers. It keeps business rules in application/domain
// layers and isolates infrastructure concerns behind ports and adapters.
package votingengine
