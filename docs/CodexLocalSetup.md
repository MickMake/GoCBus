# Local Codex setup on macOS

This guide prepares a local GoCBus checkout for Codex. It does not implement Slice 1 or install anything automatically. The [release plan](v0.42/ImplementationPlan.md) and [implementation state](v0.42/ImplementationState.md) remain authoritative.

## 1. Prerequisites

Use Codex with the local checkout on your Mac. The terminal instructions below use Codex CLI; a local app/IDE session should use the same repository directory and task prompts.

| Tool | Needed for |
| --- | --- |
| Git | Checkout, branches and diffs |
| Codex CLI or a local Codex app/IDE session | Running the coding agent |
| GitHub CLI (`gh`), authenticated for MickMake/GoCBus | Checking PR merge status and creating/updating PRs in the terminal workflow |
| Go toolchain | Slice 1 onward; select and record the supported Go version in that slice's `go.mod` and development notes |
| Python and libcbus dependencies | Only when running reference/differential tests that need them; not required to run GoCBus |

Install missing tools through their official instructions or your existing Mac package manager. Do not replace a working installation or change global Codex settings just for this repository.

Verify locally in Terminal:

```sh
git --version
gh --version
gh auth status
go version
```

For CLI use, also run `codex --version`. If GitHub authentication is missing, run `gh auth login`; for HTTPS Git operations, `gh auth setup-git` can configure Git to use that authentication. An existing working SSH setup is also fine. Codex sign-in and GitHub authentication are separate.

Use the [official Codex CLI guide](https://developers.openai.com/codex/cli) for installation and sign-in, [GitHub CLI](https://cli.github.com/manual/) for GitHub setup, and [Go installation](https://go.dev/doc/install) for the compiler. This guide does not pin a Codex model or require an API key configuration.

## 2. Open the checkout

For a new checkout, run these from your chosen development directory:

```sh
git clone https://github.com/MickMake/GoCBus.git
cd GoCBus
```

If the checkout already exists, open that directory instead; preserve any unfinished work. Confirm the repository and GitHub access:

```sh
git remote -v
git status --short --branch
git fetch origin
gh repo view MickMake/GoCBus
gh pr list --repo MickMake/GoCBus --state open
```

Start a CLI session from the repository root with `codex`, or select this checkout in your local app/IDE session. The root [AGENTS.md](../AGENTS.md) supplies repository instructions. Existing global or nested instructions may also apply; ask Codex to identify the loaded guidance before the first task. Do not regenerate AGENTS.md with `/init` over the project's reviewed instructions.

Keep your normal permission controls enabled. The written branch/approval workflow is an agent instruction, not an operating-system access control or a guarantee that every command will prompt.

## 3. Make the pinned Python reference available

Keep the reference outside GoCBus, for example as a sibling checkout. From the GoCBus root, these commands create a **new** reference directory; if it already exists, inspect it rather than overwriting or checking out over local changes:

```sh
git clone https://github.com/micolous/cbus.git ../libcbus-reference
git -C ../libcbus-reference checkout --detach cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1
git -C ../libcbus-reference rev-parse HEAD
```

The last command must print the full pinned SHA above. Give Codex the reference's actual path. If the local session cannot access a sibling directory, allow that specific directory through your client's normal controls or read the pinned source through GitHub; do not disable the sandbox broadly.

Read only the files relevant to the selected slice. If differential tests require executing Python, use a separate virtual environment and inspect the pinned project's dependency instructions first. Reference execution must not accidentally start its live MQTT/PCI daemon. Do not vendor the whole Python project into GoCBus.

## 4. Start or resume one slice

First-task prompt to paste into Codex:

```text
Prepare Slice 1 for implementation using AGENTS.md and
docs/v0.42/slices/v0.42.1-implementation-prompt.md.
Read the current implementation plan/state and Slice 1 design.
Check previous branches and PRs are merged and inspect latest main.
Identify the pinned libcbus source and fixtures required for this slice.
Report any missing local prerequisites, propose the specified branch and
implementation intent, then stop for my approval before making changes.
```

Provide the libcbus checkout path alongside the prompt. After the proposed scope/branch looks right, approve it in that session. Codex should create the slice branch from latest `origin/main` (for example `v0.42.1-foundation`), implement that slice, update its state, validate, commit/push and open a PR. It then waits for review and fixes valid findings as described below. It must not merge or proceed to Slice 2 without the corresponding instruction.

For a fresh session continuing unfinished work:

```text
Resume the already approved work on <branch> / PR <number>.
Read AGENTS.md, the slice design/prompt and ImplementationState.md.
Inspect the actual working tree, commits and review feedback first.
Continue only the approved scope on that branch; do not discard local
changes or create a replacement branch. Report any missing context.
```

Replace the placeholders. An existing open PR for the work being resumed is expected. Other unmerged prior work still needs Mick's decision. A reviewer working read-only should report findings, not silently apply them.

## 5. Validation by project stage

### Before Slice 1 / documentation only

There is no Go module, executable or Go test suite yet. Run:

```sh
git diff --check
git diff --stat
git diff
```

These inspect unstaged changes. Also inspect `git diff --cached` and use `git diff --check --cached` if edits are staged. Review newly created files too: ordinary `git diff` omits untracked files. After committing, review the whole PR with `git diff origin/main...HEAD` and `git diff --check origin/main...HEAD`.

Check changed relative Markdown links and that affected slice designs/prompts agree. Record runtime tests as not applicable, not passed. Do not run `go mod init` as a setup shortcut; the module/toolchain choices belong to approved Slice 1 work.

### Once Slice 1 provides Go packages

Run from the repository root with the toolchain supported by `go.mod`:

```sh
go test ./...
go vet ./...
go build ./...
```

Format changed Go files with `gofmt -w <file...>` before these checks. Once `cmd/GoCBus` exists, verify its case-sensitive binary name without writing a build artifact into the source tree:

```sh
gocbus_build_dir="$(mktemp -d)"
go build -o "$gocbus_build_dir/GoCBus" ./cmd/GoCBus
```

Remove that temporary build directory when finished. When concurrency or recovery is introduced/changed, also run relevant race tests, for example `go test -race ./...`; report any unsupported toolchain/platform prerequisites instead of claiming success. Add focused acceptance scenarios from the owning slice, not unrelated feature tests.

### Hardware from Slice 2 onward

The harness does not exist yet. Slice 2 must document its exact opt-in command and required local settings when it is implemented; this guide intentionally provides no pretend hardware command.

Default tests use fixtures/fakes and must not contact serial PCI, TCP CNI, live MQTT or physical loads. For an authorised hardware run, record the target/interface, test scope, setup assumptions, command, observations and limitations. Identify any reset or load-changing operation before running it. Do not guess a device path or scan for a target. Keep machine-specific configuration and raw captures local; commit only reviewed, sanitised fixtures with provenance.

## 6. Commit, review, fix and hand off

Keep `ImplementationState.md` current as work starts and changes: slice/branch, actual files, behaviour/fixtures, tests run, discoveries, deviations and remaining work. Distinguish in progress, implemented/validated with PR review outstanding, blocked, and actually merged. A docs-only setup change does not mark a release slice implemented.

Review the entire diff, commit/push and open/update one PR against `main`. Then follow the completion loop in AGENTS.md:

1. Wait for the configured code review of the latest pushed commit; also inspect inline threads, review submissions and required checks.
2. If no current review is queued/running/completed, request it. This repository uses Codex review; a PR comment containing `@codex review` requests another pass. Check for an existing run before requesting again.
3. Fix valid findings on the same branch, run relevant validation, update slice state and PR results, then commit/push. Do not implement unjustified or out-of-scope suggestions merely to silence a reviewer; explain or escalate them.
4. Wait for review of that new head and repeat within a maximum of **three review-and-fix rounds per PR**, including the initial review. Each round includes the completed review and resulting fixes/validation; polling does not consume rounds. Success requires no actionable findings remaining and required checks passing. A previous commit's clean review does not cover new changes.
5. Hand off the PR link, reviewed commit, checks and limitations for Mick to merge. An empty checks list alone does not establish that review completed.

Track `Review round: N/3` and the reviewed commit in the PR description after each completed round; carry it into the next substantive slice-state update. Resuming a session, pushing another commit or replacing the PR does not reset the count for the same work. If the third round is not clean, finish any in-scope fixes, record the slice as blocked on the review limit with remaining issues/checks and any unreviewed head, commit/push and hand off to Mick. Do not request/wait for or act on a fourth review without explicit approval of a new finite limit. A clean third round with passing checks can proceed to the normal merge handoff.

GitHub owns live review/merge status. Add the PR link to the slice's state record on the next substantive update, or provide it in the handoff if no further edit is needed. Record implementation milestones without repeatedly pushing status-only changes to declare the same commit reviewed. Reconcile actual merged status in the next session before starting another slice.

Codex waits while its local session is active. If review/checks are unavailable or still pending after about ten minutes, it should report the pending item and how to resume rather than announce completion. Resume with: `Continue the review/fix loop for PR <number> on its existing branch; inspect the latest head and reviews first.`

The next slice starts from newly merged `main`, not from the previous work branch. This workflow does not auto-merge.

## Official Codex references

- [AGENTS.md discovery and precedence](https://developers.openai.com/codex/guides/agents-md)
- [Codex CLI installation and local usage](https://developers.openai.com/codex/cli)

Checked when this guide was written on 30 September 2026. Use the official pages for changing client installation/UI details; repository workflow is governed by AGENTS.md and the release documents.
