# GoCBus v0.42 - Implementation State

## Current status

**Release:** v0.42  
**Phase:** Planning / pre-implementation  
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

## Current design decisions

- No C-Gate dependency.
- Start with the C-Bus Lighting application.
- Keep protocol handling separate from MQTT/Home Assistant.
- Maintain explicit UNKNOWN state rather than assuming OFF.
- Begin receiving live traffic before initial status interrogation.
- Startup status data must not overwrite newer live events.
- Use libcbus as a behavioural reference rather than perform a line-for-line port.
- Build in small vertical slices.

## Known issues / questions to resolve

These are not necessarily defects. They are implementation questions that require validation during v0.42.

### PCI/CNI initialisation

- Confirm the exact PCI/CNI interface options required for monitoring plus MMI/status handling.
- Determine whether automatic MMI should remain enabled continuously or whether controlled status requests are preferable.
- Validate behaviour on the actual C-Bus hardware in use.

### Status synchronisation

- Confirm binary and level status response behaviour against real hardware.
- Measure startup synchronisation timing rather than assuming a fixed duration.
- Define ordering/version rules for status snapshots interleaved with live SAL events.

### Ramps

- Decide how to represent a ramp in state:
  - known target level,
  - transition duration,
  - optional estimated current level,
  - definitely known current level after reconciliation.

### Confirmation handling

- libcbus notes that confirmation codes require more complete handling.
- Determine the minimum reliable command/confirmation model for GoCBus.

### Unsupported messages

- Unknown protocol messages must be preserved/logged rather than silently discarded.
- Catalogue real-world messages observed during capture/soak testing.

## Deferred beyond current slice

No implementation work has yet been deferred from v0.42 because coding has not started.

As implementation proceeds, deferred items should be recorded here with:

- what was deferred,
- why,
- expected release or condition for reconsideration.

## Variations from original design

None at present.

Any future variation must record:

1. Original design expectation.
2. Actual implementation.
3. Reason for the variation.
4. Consequences or follow-up work.

## Current next step

Begin Slice 1 from `ImplementationPlan.md`:

```text
connect -> receive bytes -> log/capture raw traffic
```

Before coding, inspect the libcbus files required for that slice, verify the current repository state, propose the implementation branch, and stop for approval.
