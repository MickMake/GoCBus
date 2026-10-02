# GoCBus v0.42 - Implementation State

## Current status

**Release:** v0.42  

**Phase:** Slice 4 implementation

**Implementation status:** Blocked on review limit after validated round-3 fix; latest head unreviewed

**Branch:** `v0.42.4-protocol-packets`

**PR:** [#9](https://github.com/MickMake/GoCBus/pull/9)

**Review round:** 3/3

**Reviewed commit:** `63b76b53c7b52dfea6bb32e9f611f026cffca3bb`

Slices 1-3 were merged in PRs #6-#8. Slice 4 implements deterministic,
side-effect-free packet encoding and decoding over the existing framing layer.
Lighting semantics, state, transport access, PCI initialisation, and automatic
reconnect policy remain outside this slice.

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
- Incremental CR/CRLF wire framing, special PCI response recognition, bounded
  buffering, overflow resynchronisation, and incomplete-input diagnostics added.
- Typed protocol packet encoding/decoding added with exact preservation of
  unsupported and malformed wire input.

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

State: **Merged in PR #7**

Branch: `v0.42.2-transport`

PR: [#7](https://github.com/MickMake/GoCBus/pull/7)

Review round: **2/3**

Reviewed commit: `d8c6a802c272ad13fcdf83a3adf0facfc3e37caa`

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

PR review round 2/3 reviewed commit
`d8c6a802c272ad13fcdf83a3adf0facfc3e37caa` and found one valid P2
documentation issue: the current status identified PR #7, but the next-step
section still told a resumed session to open a PR. The next step now directs
future work to the existing PR and latest-head review. The documentation-only
fix was checked with `git diff --check` and changed-relative-link inspection.

No real serial PCI or TCP CNI was contacted. Serial driver behaviour, live
disconnect timing, received byte traffic, and hardware capture remain
unverified until Mick authorises a specific target and run. This is expected
for the submitted implementation; the harness is ready but does not prove
hardware behaviour by merely compiling.

The final Slice 2 state commit was `8f6d978`; PR #7 was merged into `main` as
`b95650d` before Slice 3 began.

## Slice 3 implementation record

Slice: **v0.42.3 - Wire and framing**

State: **Merged in PR #8**

Branch: `v0.42.3-wire-framing`

PR: [#8](https://github.com/MickMake/GoCBus/pull/8)

Review round: **1/3**

Reviewed commit: `2af55be3bea5f3c7c038c6d32b84d905acbf604f`

Pinned reference verified at
`cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`. Relevant reference behaviour:

- `cbus/protocol/buffered_protocol.py` incrementally retains incomplete input,
  repeatedly consumes complete items, and enforces a 256-byte buffer limit.
- `cbus/protocol/packet.py` uses CRLF for PCI responses, CR for commands,
  recognises each `+` power-up notification and `!` PCI error immediately, and
  consumes `g`-`z` confirmations as two-byte responses.
- `cbus/protocol/confirm_packet.py`, `error_packet.py`, `po_packet.py`, and
  `tests/test_special_packet.py` define the short response bytes and examples.

Implemented files and packages:

- `internal/wire.Framer` accepts arbitrary read fragments for either direction
  and emits exact, independently owned wire bytes including terminators.
- PCI responses use CRLF termination; commands use CR termination. Power-up,
  PCI-error, and confirmation responses are identified without pulling Slice 4
  packet semantics into the wire package.
- Pending input is limited to 256 bytes. Oversized malformed messages are
  discarded through the direction-appropriate terminator or, inbound, until a
  recognised short-response boundary. One overflow event is reported and
  following messages decode normally.
- `Flush` reports unterminated input at stream end and clears it so data cannot
  cross connection generations unnoticed.
- Deterministic tests cover every byte split for each known item, byte-at-a-time
  feeds, multiple items and commands in one read, exact-size boundaries,
  oversized input and recovery, unknown input preservation, stable returned
  storage, the libcbus confirmation fallback, and the existing capture fixture.
- The opt-in hardware harness now feeds passive RX bytes through the framer,
  requires a complete event, fails on overflow, and logs a partial event caused
  only by the bounded capture timeout. It still performs no transmit or PCI
  initialisation.
- `README.md` and the hardware-harness README describe the implemented framing
  boundary and remaining runtime limitation.

Validation completed with Go 1.25.10 on macOS arm64 using writable isolated
build, module, and temporary caches:

- `gofmt` on all changed Go files: clean.
- `go test ./...`: pass.
- `go vet ./...`: pass.
- `go build ./...`: pass.
- `go test -race ./internal/wire`: pass.
- `go test -tags=hardware ./integration/hardware`: pass with the hardware test
  skipped because `GOCBUS_HARDWARE=1` was not set; this verifies gating and
  compilation only.
- Exact `GoCBus` binary build: pass.
- `git diff --check`: pass after the final state update.

Self-review found and fixed these issues before submission:

- The first framer API had no way to expose partial input when a stream ended.
  `Flush` now returns an incomplete diagnostic and resets the buffer.
- The initial change left the opt-in hardware harness at raw capture only,
  despite the release plan requiring later slices to extend it for framing. The
  harness now passively validates complete RX wire events without transmitting.
- Initial overflow handling cleared 256 bytes and treated the remaining tail as
  a new frame. It now discards through the current message terminator, reports
  the complete dropped-byte count, and resumes only at the next message.
- Additional review added byte-at-a-time, coalesced-command, exact-limit, and
  unterminated-overflow tests to cover fragmentation and boundary cases.
- Review round 1 found that inbound overflow discard mode recognised only CRLF,
  allowing later power-up, PCI-error, or confirmation responses to be swallowed.
  Discard mode now treats each recognised short response as a resynchronisation
  boundary, reports the overflow without counting the valid response as dropped,
  and resumes normal framing. Split-boundary tests cover all three response forms.

No real serial PCI or TCP CNI was contacted. Live byte framing, noise patterns,
and device-specific response timing remain unverified until Mick authorises a
specific target and hardware run. Packet decoding and executable consumption of
framed events remain in later slices; the Slice 3 executable path continues to
provide raw capture only.

The final Slice 3 state commit was `638487a`; PR #8 was merged into `main` as
`7a22b5b` before Slice 4 began.

## Slice 4 implementation record

Slice: **v0.42.4 - Protocol packets**

State: **Blocked on review limit after validated round-3 fix; latest head unreviewed**

Branch: `v0.42.4-protocol-packets`

PR: [#9](https://github.com/MickMake/GoCBus/pull/9)

Review round: **3/3**

Reviewed commit: `63b76b53c7b52dfea6bb32e9f611f026cffca3bb`

Pinned reference verified at
`cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`. Relevant reference behaviour was
migrated from `cbus/protocol/packet.py`, `base_packet.py`, `pm_packet.py`,
`pp_packet.py`, `dm_packet.py`, `reset_packet.py`, `po_packet.py`,
`error_packet.py`, `confirm_packet.py`, and their packet fixtures.

Implemented files and packages:

- `internal/protocol` decodes complete `wire.Event` values and encodes complete
  serial-interface wire messages without transport access or side effects.
- Point-to-multipoint packets expose priority, source, confirmation,
  application, checksum presence, and raw application payload bytes. SAL
  semantics remain in Slice 5.
- Point-to-point packets expose direct or bridged addressing and preserve raw
  CAL bytes for the status/CAL slices. All libcbus bridge-routing lengths are
  supported for deterministic round trips.
- Device-management packets expose their parameter and value while preserving
  direction, priority, source, checksum, and confirmation metadata.
- Reset, power-up, PCI-error, and confirmation packets match libcbus's special
  packet forms, including canonical `#` encoding for failed confirmations and
  duplicated `++` power-up encoding.
- Unsupported destinations, invalid checksums, malformed routing, lowercase or
  non-hexadecimal input, reserved flags, and incomplete framing become explicit
  `Unknown` packets retaining the exact original wire event and diagnostic
  reason.
- `README.md` documents the protocol boundary and the continuing raw-capture
  executable limitation.

Deterministic tests use the pinned libcbus and Serial Interface Guide fixtures
for point-to-multipoint commands, status requests, null Lighting traffic,
point-to-point CAL replies, PCI setup parameters, and special packets. They also
cover bridge routing, framer integration, unknown preservation, input ownership,
and invalid encoder inputs.

Validation completed with Go 1.25.10 on macOS arm64 using writable isolated
build, module, and temporary caches:

- `gofmt` on all changed Go files: clean.
- `go test ./...`: pass.
- `go vet ./...`: pass.
- `go build ./...`: pass.
- `go test -race ./internal/protocol`: pass.
- `go test -tags=hardware ./integration/hardware`: pass with the hardware test
  skipped because `GOCBUS_HARDWARE=1` was not set; this verifies gating and
  compilation only.
- Exact `GoCBus` binary build: pass.
- Pinned libcbus packet reference tests (`test_pm_packet`,
  `test_special_packet`, and `test_reply`): 13 pass.
- `git diff --check`: pass after the final state update.

Self-review found that the initial decoder accepted reserved flag bits and a
device-management flag paired with the wrong destination type. Encoding those
typed packets would have normalised their flags and broken byte-for-byte round
trips. They are now preserved as `Unknown` packets, with focused regression
fixtures. Self-review also added typed-nil encoder validation so callers receive
an error instead of a panic.

Review round 1 found that the encoder allowed checksum-free regular packets
which its decoder necessarily rejected as malformed. Encoding now requires a
checksum for point-to-multipoint, point-to-point, and PCI-originated
device-management packets while retaining the documented checksum-free basic
device-management command. Focused tests cover every affected packet direction.
The fix self-review also corrected older invalid-input fixtures so they continue
to exercise their named validations rather than stopping at the new checksum
guard.

Review round 2 found that command decoding did not retain the basic/extended
mode distinction, allowing accepted packets to gain or lose the extended-mode
prefix when re-encoded. GoCBus now preserves basic-mode non-device commands and
extended-mode device-management commands as `Unknown` rather than returning a
typed packet that cannot round-trip exactly. Focused fixtures cover both mode
mismatches and exact wire preservation.

Review round 3 found that bridged point-to-point input with bridge address zero
decoded to a typed packet which the encoder rejected. Pinned libcbus also
rejects a zero bridge address. GoCBus now preserves that input as `Unknown`, and
a focused regression verifies the exact diagnostic and wire-byte round trip.
The fix passed the full local validation suite. No actionable finding remains
known, but the fix head is unreviewed. GitHub CI was pending at this state
update and its live result remains authoritative. The three-round limit is
exhausted; another review requires Mick's explicit approval of a new finite
limit.

No real serial PCI or TCP CNI was contacted. Application semantics, CAL/status
interpretation, command/confirmation correlation, and executable packet
consumption remain in later slices. The protocol package is deterministic and
side-effect free.

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

- Lighting receive semantics remain in Slice 5.
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

Pinned libcbus rejects a single receive callback larger than 256 bytes and
clears the accumulated buffer when appending a callback would exceed that
limit. GoCBus instead applies the same 256-byte limit to one incomplete wire
message, allowing an arbitrarily sized transport read containing multiple
bounded messages. On message overflow it discards through the applicable
terminator and emits an overflow diagnostic before resuming. This corrective
variation is required by Slice 3's arbitrary-read, multiple-frame, bounded
recovery criteria and is covered by
`TestFramerHandlesReadLargerThanBufferWhenItemsAreBounded` and
`TestFramerReportsOverflowAndRecovers`.

Pinned libcbus retains partial input without exposing a stream-end result.
GoCBus `Flush` returns an incomplete diagnostic and clears that data, preventing
an interrupted frame from leaking into a later connection. This is an
observability and reconnect-safety correction covered by
`TestFramerFlushReportsIncompleteInputAndRecovers`.

Pinned libcbus returns `InvalidPacket` values whose payload may already have
lost framing, prefixes, or hexadecimal representation during parsing. GoCBus
instead returns `protocol.Unknown` with the complete original `wire.Event` and
a diagnostic reason. This deliberate observability correction satisfies the
Slice 4 requirement that unsupported input remain observable and permits exact
re-encoding. It is covered by `TestUnknownPacketsRemainObservable`.

Pinned libcbus decodes bridged point-to-point routing but raises
`NotImplementedError` when encoding it. GoCBus encodes the same documented
bridge-length values while preserving bridge address, hops, unit address, and
raw CAL bytes. This narrow completion of the packet round trip is required by
Slice 4's encode/decode objective and is covered by
`TestPointToPointBridgedRoundTrip`; it does not add routing policy or CAL
semantics.

Pinned libcbus treats prefix-less non-device commands as direct CAL and accepts
extended-mode device-management commands, but its returned objects do not
retain enough mode information for exact re-encoding. Direct CAL interpretation
belongs to later slices, so GoCBus preserves both noncanonical mode/type forms
as `Unknown` with their complete original `wire.Event`. This prevents silent
prefix changes and is covered by `TestUnknownPacketsRemainObservable`.

Any future variation must record:

1. Original design expectation.
2. Actual implementation.
3. Reason for the variation.
4. Consequences or follow-up work.

## Current next step

Push the validated review-round-3 handoff to PR #9 and verify required CI. Do
not request or wait for a fourth review. Mick must decide the PR disposition or
explicitly authorise a new finite review limit. Do not start Slice 5.


## Slice design status

Fourteen implementation slices are now defined under `docs/v0.42/slices/`, each with a design contract and implementation prompt. Binary name is locked as `GoCBus`. Package responsibility boundaries, foreground-first runtime behaviour, maintenance passthrough, runtime status state machine and operational MQTT/HA controls are included in the v0.42 plan.
