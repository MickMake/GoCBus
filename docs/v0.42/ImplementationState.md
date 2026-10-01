# GoCBus v0.42 - Implementation State

## Current status

**Release:** v0.42  

**Phase:** Slice 2 PR review

**Implementation status:** Review round 1 fixes implemented and validated; awaiting review of the latest head

**Branch:** `v0.42.2-transport`

**PR:** [#7](https://github.com/MickMake/GoCBus/pull/7)

**Review round:** 1/3

Slice 1 was merged in PR #6. Slice 2 implements raw serial PCI and TCP CNI
transport, capture integration, and an explicit opt-in receive-only hardware
harness. PR review round 1 found three valid issues; their fixes are validated
locally and await review of the latest head. No framing, packet parsing, PCI
initialisation, or automatic reconnect policy is included.

## Completed

- GoCBus repository created.
- Project purpose defined as a Clipsal C-Bus to MQTT gateway with specific support for Home Assistant's MQTT framework.
- libcbus selected as the principal protocol and behavioural reference.
- Initial libcbus review completed.
- Existing startup/state synchronisation weakness identified in the Python MQTT gateway.
- Migration planning documentation created.
- v0.42 implementation structure created.
- GPLv3 licensing established for GoCBus; libcbus LGPL3+ attribution retained.
- Initial Go module and case-sensitive `cmd/GoCBus` executable added.
- Minimal JSON configuration loading, validation and CLI overrides added.
- Shared wire identifiers, framing constants and libcbus-compatible checksum
  helpers added.
- Timestamped raw RX/TX capture writing and replay reading added using a
  line-delimited JSON format.
- Logging levels and separately gated raw RX/TX trace conventions added.
- Deterministic unit tests and a sanitised capture fixture added.
- Raw serial PCI transport at 9600 8N1 and TCP CNI transport added.
- Context cancellation, clear disconnect propagation, redial primitives, and
  partial-I/O-aware raw traffic observation added.
- The executable can capture configured raw transport traffic until cancelled
  or disconnected.
- An explicit build-tagged and environment-gated receive-only hardware harness
  added for selected serial/TCP targets.

## Slice 1 implementation record

Slice: **v0.42.1 - Foundation and minimal runtime/configuration**

State: **Merged in PR #6**

Branch: `v0.42.1-foundation`

PR: [#6](https://github.com/MickMake/GoCBus/pull/6)

Review round: **1/3**

Reviewed commit: `c9984c71346cf5478d3c58098c8ae41ca5bf2feb`

Implemented files and packages:

- `go.mod` establishes module `github.com/MickMake/GoCBus` with Go 1.25.
- `cmd/GoCBus` provides the minimal executable and preserves the locked binary
  name.
- `internal/runtime` provides strict JSON configuration, `-config`,
  `-log-level` and `-raw-log` overrides, validation and logging setup.
- `internal/wire` provides the immediately required `ApplicationID` and
  `GroupAddress` types, the default Lighting application identifier, framing
  constants and checksum helpers.
- `internal/capture` provides timestamped RX/TX NDJSON records with hexadecimal
  byte data and strict replay validation.
- `testdata/capture/libcbus-lighting.ndjson` is a sanitised synthetic fixture
  based on the pinned libcbus tests and PCI examples; its timestamps are
  invented and its provenance is recorded beside it.
- `README.md` documents the implemented foundation and its configuration.

Validation completed with Go 1.25.10 on macOS arm64 using writable isolated
build, module and temporary caches:

- `gofmt` on all changed Go files: clean.
- `go test ./...`: pass.
- `go vet ./...`: pass.
- `go build ./...`: pass.
- `go build -o /private/tmp/gocbus-slice1-build/GoCBus ./cmd/GoCBus`: pass.
- Built executable smoke test with default configuration: pass and exits after
  reporting that transport is not implemented in v0.42.1.

Implementation decisions:

- Configuration contains only `log_level` and `raw_log`; future settings are
  deferred until their owning slices provide real consumers.
- Unknown JSON configuration fields and trailing JSON values are rejected so
  misspellings fail visibly.
- Capture files are newline-delimited JSON with RFC3339-nanosecond timestamps,
  `rx`/`tx` direction and uppercase hexadecimal bytes. This keeps fixtures
  diffable and replay independent of transport and packet decoding.
- Raw byte logs use a dedicated trace logger behind the separate `raw_log`
  gate. Ordinary diagnostics retain the configured `log_level` threshold.
- Only the default Lighting application and generic application/group identifier
  types were introduced; further protocol enums remain with their owning slices.

No hardware validation was run or required. Slice 1 adds no connection code and
does not contact serial devices, CNI endpoints, MQTT brokers or C-Bus loads.

Three preliminary independent working-tree checks were completed before the
first commit. The first found that enabling `raw_log` lowered the shared handler
threshold to trace; the subsequent checks found the working-tree content clean.
Raw traffic now uses a dedicated trace logger while ordinary diagnostics retain
their configured threshold, covered by
`TestRawLogDoesNotLowerOrdinaryLogLevel`. These preliminary checks did not
consume the PR review-round limit.

PR review round 1/3 independently reviewed submitted commit
`c9984c71346cf5478d3c58098c8ae41ca5bf2feb` and found two valid P2 issues:

- Top-level JSON `null` was accepted because decoding it into a preinitialised
  config struct was a no-op. Configuration now requires a non-null JSON object;
  focused tests also reject array and scalar roots while preserving unknown
  field and trailing-value checks.
- This state file incorrectly counted the three preliminary working-tree checks
  as PR review rounds 3/3. They are now recorded separately and the submitted
  review count is correctly 1/3.

Both fixes were submitted in commit `f0e0628`; the final state update was
`047845d`, and PR #6 was merged into `main` as `603d93e`. The remaining
validation limitation carried into Slice 2 was that the capture fixture was
synthetic and live PCI/CNI behaviour had not been exercised.

## Slice 2 implementation record

Slice: **v0.42.2 - Transport**

State: **Review round 1 fixes implemented and validated; awaiting review of the latest head**

Branch: `v0.42.2-transport`

PR: [#7](https://github.com/MickMake/GoCBus/pull/7)

Review round: **1/3**

Reviewed commit: `0c747d67d785c8f94da370f8af93fea03f8bd8c6`

Pinned reference verified at
`cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`. Relevant reference behaviour:

- `cbus/protocol/pciprotocol.py` and `cbus/daemon/cmqttd.py` open serial PCI
  connections at 9600 baud and TCP CNI connections from a host plus numeric
  port, and surface connection loss to the caller.
- libcbus uses pyserial's default 8N1 mode; GoCBus sets all four serial mode
  values explicitly rather than relying on dependency defaults.
- libcbus couples transport to packet handling and PCI reset. GoCBus preserves
  its connection settings and lifecycle behaviour but keeps framing, parsing,
  and reset/initialisation in their owning later slices.

Implemented files and packages:

- `internal/transport` provides reusable serial and TCP dialers returning raw
  `io.ReadWriteCloser` streams. Calling a dialer again after close is the
  reconnect hook; this slice supplies no retry policy.
- `internal/transport.Observe` records the bytes actually completed by each
  read/write, including partial I/O, and propagates observer failures without
  hiding completed byte counts.
- `internal/capture.Writer` now serialises concurrent RX/TX writes.
- `internal/runtime` adds strict `serial_device`, `tcp_address`, and
  `capture_path` configuration plus `-serial`, `-tcp`, and `-capture`
  overrides. It runs a foreground raw receive/capture loop, closes transport on
  cancellation, and reports unexpected disconnects as failures.
- `cmd/GoCBus` maps interrupt and termination signals to runtime cancellation.
- `integration/hardware` provides the same build-tagged harness for explicitly
  selected serial or TCP targets. A second environment gate and explicit local
  capture path are required.
- `README.md` and `docs/CodexLocalSetup.md` document configuration, lifecycle,
  capture sensitivity, and hardware opt-in.
- `go.bug.st/serial` v1.8.0 is the sole direct transport dependency.

Validation completed with Go 1.25.10 on macOS arm64 using writable isolated
build, module, and temporary caches:

- `gofmt` on all changed Go files: clean.
- `go test ./...`: pass, including loopback TCP connection, read/write,
  disconnect, cancellation, redial, capture, validation, and partial-I/O cases.
- `go test -race ./...`: pass.
- `go test -tags=hardware ./integration/hardware`: pass with the hardware test
  skipped because `GOCBUS_HARDWARE=1` was not set; this verifies opt-in gating
  and compilation only.
- `go vet ./...`: pass.
- `go build ./...`: pass.
- Exact `GoCBus` binary build and no-transport smoke run: pass.
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` `GoCBus` cross-build: pass.
- `git diff --check`: pass.
- Changed relative Markdown links: pass.

Self-review found and fixed these issues before submission:

- Cancellation during connection setup was initially reported as a transport
  failure. It now exits cleanly, with focused before/during-dial tests.
- `main` initially deferred signal cleanup immediately before `os.Exit`, whose
  semantics skip defers. Cleanup now runs explicitly before exit.
- TCP validation initially accepted service names; the pinned reference parses
  the port as an integer, so configuration and dialers now require a numeric
  port in the range 1-65535.
- Serial cancellation after the port opens now closes that port and is covered
  by a focused test.
- Live arbitrary transmit was excluded from the hardware harness because no
  target or command was authorised. Deterministic tests cover raw write and
  partial-write behaviour without risking C-Bus loads.

PR review round 1/3 reviewed commit
`0c747d67d785c8f94da370f8af93fea03f8bd8c6` and found three valid P2 issues:

- Existing capture files retained broader permissions because the creation mode
  did not apply when truncating them. Runtime and hardware capture now share an
  opener that explicitly enforces owner-only permissions, with a regression
  test covering a pre-existing `0644` file.
- File configuration was fully validated before CLI transport overrides, so a
  valid override could not replace an invalid or incomplete file value. The
  runtime now decodes first, applies all overrides, and validates the final
  configuration once; direct `LoadConfig` callers retain strict validation.
- Serial cancellation could not return while the platform's synchronous open
  call was blocked. Serial open now runs behind a context-aware handoff, returns
  promptly on cancellation, and closes a port if the underlying open finishes
  later. A deterministic blocking-open test covers the lifecycle.

Self-review of the round-one fixes found that the initial CLI-override test used
a relative capture path and left a test artifact in the package directory; it
now uses a temporary path. The race suite also found the asynchronous serial
cleanup test reading a non-atomic fake-port flag; the helper now uses an atomic
close state. The complete validation matrix above then passed.

No real serial PCI or TCP CNI was contacted. Serial driver behaviour, live
disconnect timing, received byte traffic, and hardware capture remain
unverified until Mick authorises a specific target and run. This is expected
for the submitted implementation; the harness is ready but does not prove
hardware behaviour by merely compiling.

## Current design decisions

- No C-Gate dependency.
- Start with the C-Bus Lighting application.
- Keep protocol handling separate from MQTT/Home Assistant.
- Maintain explicit UNKNOWN state rather than assuming OFF.
- Begin receiving live traffic before initial status interrogation.
- Startup status data must not overwrite newer live events.
- Use libcbus as a behavioural reference rather than perform a line-for-line port.
- Build in small vertical slices.
- Pin the v0.42 libcbus behavioural reference to commit `cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`.
- Establish minimal executable/configuration bootstrap in Slice 1 and extend configuration only as later slices require it.
- Begin opt-in real-hardware integration testing in Slice 2; default tests remain deterministic and hardware-independent.
- Treat the lifecycle enum as C-Bus lifecycle state; MQTT connection state is separate.
- Preserve responsibility boundaries while allowing package topology to simplify when implementation evidence supports it.
- The v0.42 roadmap may be cut into a release after any useful stable completed slice; remaining slices may roll forward.

## Planning revision history

| Revision | Decision and reason |
| --- | --- |
| PR #1 (`docs/migration-planning`) | Expanded the initial seven stages into fourteen slice contracts, including exclusive maintenance passthrough and operational controls, to keep intended capabilities visible. |
| PR #2 (`migration-planning2`) | Tightened behaviour-first migration, explicitly exempted Slice 8 state management, relaxed package topology and removed the redundant umbrella prompt to reduce accidental redesign. |
| PR #3 (`docs/migration-planning3`) | Moved minimal bootstrap/configuration earlier, introduced early hardware validation, pinned the reference, clarified freshness/subsystem status and allowed useful stable release cuts. |
| `docs/migration-planning4` | Clarifies integration milestones, release gates and slice-owned acceptance decisions without changing the fourteen-slice roadmap or behaviour-first intent. |

These are agreed planning refinements, not evidence of completed implementation. Hardware-dependent timings and policies remain for validation in their owning slices.

## Behavioural deviations from libcbus

Intentional changes from libcbus observable behaviour must be recorded here.

Each entry should include:

1. Relevant libcbus behaviour and source reference.
2. GoCBus behaviour.
3. Why the difference is required.
4. Evidence supporting the change.
5. Tests covering the difference.

Expected v0.42 deviations currently include reliable startup/current-state synchronisation, fuller confirmation tracking, and reconnect/state recovery behaviour.

The pinned `cbus/daemon/cmqttd.py` republishes requested state immediately after sending a command. GoCBus intentionally requires observed evidence instead; do not restore optimistic authoritative publication under the compatibility rule. Tests are due in Slices 8-9. Before Slice 9 fixes public topics/payloads, record the MQTT/HA compatibility decisions required by its design; discovery scope and entity identity must not be inferred from the Python defaults.

## Known issues / questions to resolve

These are not necessarily defects. They are implementation questions that require validation during v0.42.

### PCI/CNI initialisation

- Confirm the exact PCI/CNI interface options required for monitoring plus MMI/status handling.
- Determine whether automatic MMI should remain enabled continuously or whether controlled status requests are preferable.
- Validate behaviour on the actual C-Bus hardware in use.

### Status synchronisation

- Confirm binary and level status response behaviour against real hardware.
- Measure startup synchronisation timing rather than assuming a fixed duration.
- Validate the planned generation/dirty rule: a live SAL event received after a sync generation begins must prevent status data from that generation overwriting the group unless protocol evidence provides stronger ordering.
- Lock the canonical state identity before Slice 8; a bare group address must not be assumed globally unique and network/application/group context must be preserved as required by the protocol model.
- Validate reconnect publication semantics: last-known values may be retained for diagnostics, but after C-Bus loss they are stale/non-authoritative until new live/status evidence confirms them.

### Ramps

- Decide how to represent a ramp in state:
  - known target level,
  - transition duration,
  - optional estimated current level,
  - definitely known current level after reconciliation.

### Confirmation handling

- libcbus notes that confirmation codes require more complete handling.
- Determine the minimum reliable command/confirmation model for GoCBus, including allocation, matching, timeout and reconnect behaviour.
- A successful PCI/protocol confirmation is not automatically proof of final load/group state; authoritative state changes require observed SAL/status evidence or another explicitly validated protocol guarantee.

### Unsupported messages

- Unknown protocol messages must be preserved/logged rather than silently discarded.
- Catalogue real-world messages observed during capture/soak testing.

## Deferred beyond current slice

- Buffering, framing and packet decoding remain in Slices 3-5.
- PCI initialisation and live transmit validation remain in Slice 6.
- Automatic reconnect policy remains with later runtime integration rather than
  the Slice 2 lifecycle primitives.
- MQTT, Home Assistant and other subsystem settings remain absent until their
  owning slices implement the corresponding behaviour.
- Complete lifecycle orchestration and configuration consolidation remain in
  Slice 12.

## Implementation variations from the agreed design

The design did not prescribe the configuration or capture file encodings. Slice
1 uses strict JSON configuration and NDJSON capture records as the smallest
standard-library implementation. This does not vary an observable libcbus
contract because libcbus has no equivalent capture/config format for this
foundation.

`ValidateChecksum` returns `false` for empty input. Pinned libcbus
`validate_cbus_checksum` indexes the final byte and raises `IndexError` for an
empty value. The Go behaviour is a deliberate defensive correction: validation
helpers report invalid input without panicking. It is covered by
`TestValidateChecksumRejectsInvalid`.

Any future variation must record:

1. Original design expectation.
2. Actual implementation.
3. Reason for the variation.
4. Consequences or follow-up work.

## Current next step

Commit and push Slice 2, open one PR against `main`, then await independent
review of the submitted head. Do not start Slice 3.


## Slice design status

Fourteen implementation slices are now defined under `docs/v0.42/slices/`, each with a design contract and implementation prompt. Binary name is locked as `GoCBus`. Package responsibility boundaries, foreground-first runtime behaviour, maintenance passthrough, runtime status state machine and operational MQTT/HA controls are included in the v0.42 plan.
