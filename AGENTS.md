# GoCBus agent instructions

## Project and source of truth

GoCBus is a small Go gateway from Clipsal C-Bus to MQTT/Home Assistant. Keep it simple: migrate proven libcbus behaviour into idiomatic Go and make only agreed, evidence-backed improvements. Binary name: `GoCBus`.

- For release/slice work, read [ImplementationPlan](docs/v0.42/ImplementationPlan.md), [ImplementationState](docs/v0.42/ImplementationState.md), and the selected slice's design and implementation prompt under `docs/v0.42/slices/`.
- The plan governs shared rules; designs define local contracts; prompts execute them; the state document records decisions and progress. Resolve conflicts before implementing disputed behaviour.
- Use [MigrationPlanning](docs/MigrationPlanning/README.md) for rationale and [local setup](docs/CodexLocalSetup.md) for environment/workflow help. Do not duplicate the release plan in another planning framework.
- Read only the additional source/docs needed for the task. Do not depend on remembered chat history as the implementation specification.

## Branch and approval workflow

Before starting new implementation or documentation changes:

1. Inspect the working tree without discarding or overwriting user work. Fetch `origin` and identify latest `origin/main`.
2. Check previous local/remote work branches and PRs for merge status. If prior work is unmerged, stop and report the branch/PR. A deleted branch or closed PR alone is not proof of merge; check the PR's merged status when ancestry is ambiguous, including squash merges. If GitHub access is unavailable, report that verification is blocked rather than guessing.
3. State the intended scope and branch name, then stop for Mick's approval before editing. Use the slice prompt's branch name when one is supplied.
4. After approval, create the branch from latest `origin/main`. Implement only the approved scope; do not start the next slice as a small extra.

Approval persists for that scope. When explicitly continuing approved work or fixing its review feedback, inspect and use that existing branch/PR; its own open PR is not unrelated prior work. Do not create a fresh branch or ask for the same approval again. Stop if unexpected work or conflicting instructions prevent a safe continuation.

At completion, update implementation progress/decisions when applicable, validate the change, review the diff, commit and open/update one PR against `main`. Report tests actually run, limitations and deviations. Leave merging to Mick unless explicitly instructed to merge. Never reset, clean or force-push away other work.

## Implementation boundaries

- Pinned libcbus reference: `micolous/cbus` commit `cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`. Verify the checkout/ref; never silently follow upstream HEAD.
- Identify relevant Python behaviour and fixtures before porting it. Preserve observable protocol behaviour, not Python architecture. Record deliberate deviations and their evidence in `ImplementationState.md`.
- Slice 8 intentionally improves state management. Follow its contracts rather than restoring libcbus's event-only/optimistic state behaviour.
- libcbus/Python are development references, not GoCBus runtime dependencies. Preserve upstream attribution when porting source/fixtures.
- Keep protocol, state and MQTT/HA responsibilities separate; do not create packages or abstractions solely to fill the planned directory list.
- Unknown/stale state, startup event ordering, command confirmation, synchronisation and exclusive passthrough follow the release/slice contracts. Do not substitute guessed hardware guarantees.

## Validation and hardware

- This repository starts in pre-implementation state. Check whether `go.mod`, packages and the executable exist before selecting commands. Do not invent a successful build or create scaffolding merely to make a documentation check pass.
- Documentation-only changes: inspect the diff, run `git diff --check`, and check changed relative links and any affected design/prompt pairs. No runtime tests are required.
- Once Go code exists: format changed Go files, run `go test ./...`, `go vet ./...`, and build the implemented packages/executable. Run focused race/failure tests where concurrency or recovery is affected; see the setup guide for commands and prerequisites.
- Default tests must be deterministic and independent of serial ports, CNI endpoints, a live broker and real C-Bus loads. Hardware validation starts in Slice 2 through a separate explicit opt-in harness.
- Do not connect to or transmit on live hardware just because an endpoint is reachable. Use the agreed target and explicitly authorised test scope; describe commands that may change loads or reset the interface. Record hardware tests separately from unit tests.
- Keep credentials, machine-specific endpoints and unreviewed raw captures out of commits. Add only deliberate, sanitised fixtures with provenance.

## Code Review Rules

Check the approved slice and acceptance criteria, behavioural compatibility and documented exceptions, startup/reconnect ordering, bounded work and failure outcomes. Flag contradictions between plans, designs and prompts. Distinguish demonstrated defects from hardware questions still assigned to a later slice; do not demand unrelated features or speculative redesign.
