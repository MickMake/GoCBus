# GoCBus v0.42 - Implementation Plan

## Release intent

v0.42 is the first pre-v1.0 implementation release of GoCBus.

The purpose of this release is to establish the core C-Bus-to-MQTT gateway architecture and prove the critical path from C-Bus traffic through a state engine to Home Assistant-compatible MQTT.

This document is the gold-standard implementation plan for v0.42. It should be followed unless new evidence makes part of it materially wrong. Any variation must be recorded in `ImplementationState.md`.

## Release goals

v0.42 should establish:

1. Direct communication with Clipsal C-Bus PCI/CNI hardware without C-Gate.
2. Reliable receive-side framing and protocol decoding.
3. Typed Lighting events for ON, OFF and RAMP.
4. C-Bus status interrogation and startup synchronisation.
5. A state engine that distinguishes UNKNOWN from known state.
6. MQTT publishing and commands.
7. Home Assistant MQTT Discovery support.
8. Recovery from C-Bus, MQTT and process restarts.
9. Diagnostics good enough to understand protocol and state failures.

## Architecture

The intended data flow is:

```text
C-Bus PCI / CNI
      |
      v
  Transport
      |
      v
   Framing
      |
      v
C-Bus Protocol
      |
      v
 Typed Events
      |
      v
 State Engine
      |
      v
    MQTT
      |
      v
Home Assistant
```

The protocol implementation must remain independent of MQTT and Home Assistant.

## Implementation slices

### Slice 1 - Transport and raw receive

Goal:

```text
connect -> receive bytes -> log/capture raw traffic
```

Deliverables:

- Serial PCI transport.
- TCP CNI transport.
- Connection lifecycle.
- Raw RX/TX diagnostics.
- Capture/replay format suitable for tests.

Exit criteria:

- GoCBus can connect to a real or replayed C-Bus endpoint.
- Incoming traffic can be captured without interpretation.
- Disconnects are detected cleanly.

### Slice 2 - Framing and Lighting decode

Goal:

```text
receive bytes -> frame packets -> decode Lighting ON/OFF/RAMP -> emit typed events
```

Deliverables:

- Incremental receive buffer.
- Packet boundary detection.
- Checksum handling where applicable.
- Parser resynchronisation after malformed input.
- Typed Lighting ON event.
- Typed Lighting OFF event.
- Typed Lighting RAMP event.
- Unknown packet preservation/logging.

Exit criteria:

- Recorded traffic can be replayed into deterministic typed Lighting events.
- Fragmented and combined reads are handled correctly.
- Invalid data does not permanently desynchronise the parser.

### Slice 3 - Commands and confirmations

Deliverables:

- Lighting ON command.
- Lighting OFF command.
- Lighting RAMP command.
- Terminate-ramp where practical.
- Confirmation handling.
- Send pacing/back-pressure where required by the PCI.

Exit criteria:

- Commands can be sent to real hardware and confirmed or failed explicitly.
- The implementation does not silently lose command failures.

### Slice 4 - Status/MMI and state engine

This slice addresses the principal weakness identified in the old libcbus MQTT gateway.

Deliverables:

- Lighting status request support.
- Binary status/MMI decode.
- Level status/MMI decode where supported.
- Per-group state with UNKNOWN/known distinction.
- Merge of live SAL events with startup status replies.
- Ordering/versioning so an older snapshot cannot overwrite a newer live event.
- Periodic or passive reconciliation strategy.

Startup flow:

1. Connect and initialise.
2. Start the receive loop immediately.
3. Mark state UNKNOWN.
4. Request Lighting binary status.
5. Request Lighting level status where supported.
6. Merge replies with live events.
7. Expose each group as soon as its state becomes known.
8. Continue reconciliation after startup.

Exit criteria:

- Restarting GoCBus establishes current Lighting state without waiting for someone to press every switch.
- A live state change during startup cannot be overwritten by stale snapshot data.

### Slice 5 - MQTT gateway

Deliverables:

- MQTT connection lifecycle.
- State publication.
- Command subscription.
- Availability state.
- Retained-state strategy.
- Mapping between C-Bus group addresses and MQTT entities.

Exit criteria:

- C-Bus Lighting changes appear through MQTT.
- MQTT commands produce C-Bus commands.
- Reconnects do not cause retained stale state to be treated as authoritative.

### Slice 6 - Home Assistant integration

Deliverables:

- Home Assistant MQTT Discovery.
- Lighting brightness support.
- Stable entity identifiers.
- Availability reporting.
- Sensible naming/configuration mechanism.

Exit criteria:

- Home Assistant discovers configured C-Bus Lighting groups.
- HA controls and displayed state remain consistent with physical C-Bus operation.

### Slice 7 - Hardening

Deliverables:

- Reconnect testing.
- Long-running soak testing.
- Startup race tests.
- Malformed-frame tests.
- Metrics/counters for parser and transport failures.
- Documentation of unsupported protocol areas.
- Packaging/service instructions as appropriate.

## Testing principles

Use libcbus as a behavioural reference, not as an architecture to translate.

Where practical:

- Replay identical captures through libcbus and GoCBus and compare logical events.
- Split each known frame at every possible byte boundary.
- Combine multiple frames into one read.
- Inject corrupt frames and noise.
- Test startup status replies interleaved with live SAL.
- Test C-Bus reconnects.
- Test MQTT reconnects.
- Test restart while lights are already on.
- Test ramps and terminate-ramp.

## Deliberate non-goals for v0.42

- Replacing C-Bus Toolkit.
- Reimplementing C-Gate.
- Supporting every C-Bus application.
- Building a commissioning user interface.
- Premature generic frameworks or plugin systems.

Lighting working reliably is more valuable than a beautiful abstraction for applications that do not yet exist.

## Change control

This plan is authoritative for v0.42.

If implementation reveals that a design assumption is wrong:

1. Record the discovery in `ImplementationState.md`.
2. Describe the required variation and reason.
3. Update this plan only when the new direction has been agreed.
4. Do not silently drift from the plan.

Whole-earth rupture exemptions will be assessed on their merits.
