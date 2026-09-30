# GoCBus v0.42 - Implementation State

## Current status

**Release:** v0.42  
**Phase:** Detailed slice design / pre-implementation  
**Implementation status:** Not yet started

The migration design has been drafted and the v0.42 implementation plan is being established before production code is written.

## Completed

- GoCBus repository created.
- Project purpose defined as a Clipsal C-Bus to MQTT gateway with specific support for Home Assistant's MQTT framework.
- libcbus selected as the principal protocol and behavioural reference.
- Initial libcbus review completed.
- Existing startup/state synchronisation weakness identified in the Python MQTT gateway.
- Migration planning documentation created.
- v0.42 implementation structure created.
- GPLv3 licensing established for GoCBus; libcbus LGPL3+ attribution retained.

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

No implementation work has yet been deferred from v0.42 because coding has not started.

As implementation proceeds, deferred items should be recorded here with:

- what was deferred,
- why,
- expected release or condition for reconsideration.

## Implementation variations from the agreed design

None at present because implementation has not started. Planning revisions are recorded separately above.

Any future variation must record:

1. Original design expectation.
2. Actual implementation.
3. Reason for the variation.
4. Consequences or follow-up work.

## Current next step

Begin Slice 1 from `ImplementationPlan.md`:

```text
minimal GoCBus/config bootstrap -> foundation primitives -> checksum/validation helpers -> capture/replay scaffolding -> deterministic fixtures/tests
```

Transport connection work and the opt-in real-hardware integration harness begin in Slice 2. Before coding, inspect the pinned libcbus files required for Slice 1, verify the current repository state, propose the implementation branch, and stop for approval.


## Slice design status

Fourteen implementation slices are now defined under `docs/v0.42/slices/`, each with a design contract and implementation prompt. Binary name is locked as `GoCBus`. Package responsibility boundaries, foreground-first runtime behaviour, maintenance passthrough, runtime status state machine and operational MQTT/HA controls are included in the v0.42 plan.
