# Documentation Workflow (Solomon)

## Step 1: Read Before Writing
- Read target module under `solomon/contexts/<context>/<service>/`.
- Read module contracts and adapters.
- Read related canonical spec files.

## Step 2: Extract Key Behavior
- Identify domain rules and state transitions.
- Identify repository writes and read-only dependencies.
- Identify emitted/consumed events and outbox usage.

## Step 3: Update Docs
- Update module README first.
- Update `solomon/docs/` when architecture-level behavior changes.
- Add rationale block for non-trivial design decisions.

## Step 4: Validate Consistency
- Ensure dependency/ownership terminology matches canonical maps.
- Ensure described flows match code.
- Ensure testing and failure behavior sections are current.