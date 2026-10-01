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
3. State the intended scope and branch name, then stop for Mick's approval before editing. Slice branches use `v0.42.<slice>-<short-description>`; use the exact branch name in the slice prompt when supplied (for example `v0.42.1-foundation`).
4. After approval, create the branch from latest `origin/main`. Implement only the approved scope; do not start the next slice as a small extra.

Approval persists for that scope. When explicitly continuing approved work or fixing its review feedback, inspect and use that existing branch/PR; its own open PR is not unrelated prior work. Do not create a fresh branch or ask for the same approval again. Stop if unexpected work or conflicting instructions prevent a safe continuation.

Approval to implement a slice includes committing, pushing, opening its PR and addressing valid review findings on that branch. Leave merging to Mick unless explicitly instructed to merge. Never reset, clean or force-push away other work.

## Completion and review loop

1. Update `ImplementationState.md` as slice work starts, is implemented/validated, or becomes blocked. Record the slice, branch, actual changes, tests/results, deviations and remaining work. Before submission, describe implemented work as awaiting PR review, not merged or released. A docs-only change does not implement a slice.
2. Validate, review the entire diff, commit and push, then create/update one PR against `main`. Keep the PR description and validation results accurate after fixes.
3. Wait for code review to finish for the latest pushed commit, and inspect review submissions, inline threads and required checks. A green CI check, an old review, or no comments yet is not proof of a completed current review.
4. Request/re-request the configured review when needed. For this repository's Codex review, use `@codex review` on the PR if the latest commit has no review queued/running/completed; do not spam duplicate requests. PR review requests and replies explaining fixes are part of this workflow.
5. Assess every finding. Fix valid in-scope issues on the same branch, add relevant tests, update slice state/results, commit and push. Explain a rejected finding with evidence; ask Mick about unresolved disagreements or scope changes rather than blindly implementing every suggestion. Resolve a thread only when its concern is addressed.
6. Repeat review and validation for the new head within the three-round limit below. Finish successfully only when the latest head has completed review, no actionable findings remain unresolved, and required checks pass. Report the PR, reviewed commit, validation and remaining limitations; then wait for Mick to merge.

Allow at most **three review-and-fix rounds per PR**, including the initial review. A round is one completed review of a submitted head plus any resulting fixes and validation; polling and duplicate review responses do not start new rounds. Record `Review round: N/3` and the reviewed commit in the PR description when each round completes, and carry the count into the next substantive slice-state update. The count persists across commits, sessions and resumed work; do not reset it by opening a replacement PR for the same work.

If round three is clean and required checks pass, hand off for merge. Otherwise stop: after any in-scope round-three fixes, update the slice state as blocked on the review limit (including count, remaining findings/checks and any unreviewed latest head), commit/push the handoff, and report it to Mick. Do not request or wait for a fourth review, act on an automatically triggered fourth review, or claim completion. Further rounds require Mick's explicit approval of a new finite limit.

While the session is active, poll review status at sensible intervals and give progress updates. If review/checks remain unavailable or pending after about ten minutes, or access/session limits prevent waiting, report the exact pending item and a resume instruction; do not claim review is complete or promise background monitoring after the session ends.

Keep implementation milestones in `ImplementationState.md`; GitHub is authoritative for live review/merge status. Include the PR link in the state record on the next substantive update, or in the handoff if there is no further edit. Avoid an endless sequence of status-only commits invalidating their own reviews. On the next session, reconcile the record with the actual merge result before selecting another slice; never mark work merged speculatively.

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
