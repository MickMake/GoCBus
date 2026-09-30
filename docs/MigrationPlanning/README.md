# GoCBus Migration Planning

## Purpose

GoCBus is a Go implementation of a direct Clipsal C-Bus to MQTT gateway, with specific support for the Home Assistant MQTT framework.

The project began as a migration from Michael Farrell's `micolous/cbus` (`libcbus`) Python implementation. libcbus is the primary behavioural baseline for the first GoCBus implementation. GoCBus should preserve proven protocol behaviour unless there is clear evidence that the behaviour is wrong, incomplete, or prevents the project from meeting its reliability goals. Implementation structure may be changed freely where required to make the code idiomatic Go; behaviour should not change merely because a different design appears cleaner.

The v0.42 behavioural reference is pinned to libcbus commit `cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`. That reference should not move silently during the migration; any change of reference must be explicit and recorded.

## Primary goals

- Communicate directly with Clipsal C-Bus PCI/CNI hardware without C-Gate.
- Support the C-Bus Lighting application first.
- Provide a reliable C-Bus state model rather than only forwarding events.
- Publish devices and state over MQTT.
- Support Home Assistant MQTT Discovery.
- Recover cleanly after process, C-Bus, MQTT or network interruptions.
- Keep the protocol implementation usable independently of MQTT and Home Assistant.

## Migration rules

For each migrated behaviour:

1. Identify the relevant libcbus implementation.
2. Capture or reproduce its observable input/output behaviour.
3. Add Go tests for that behaviour.
4. Implement the smallest idiomatic Go equivalent.
5. Verify against libcbus fixtures and, where practical, real hardware.
6. Record any intentional behavioural difference.

Protocol documentation and real hardware observations take precedence when they demonstrate that libcbus behaviour is incorrect or incomplete.

## Key finding from libcbus

The existing Python MQTT gateway is principally event-driven.

During PCI initialisation, libcbus deliberately configures the interface with MMI reporting disabled. The protocol implementation contains support for constructing a Lighting status request, including a level request, but the call in the periodic time-sync path is commented out.

The `cmqttd` MQTT bridge then derives state primarily from live Lighting ON, OFF and RAMP events.

This creates a startup/reconnect weakness: if a C-Bus group changes while the gateway is offline, restarting the gateway does not necessarily establish that group's current state. Home Assistant may remain stale or unknown until another event occurs for that group.

GoCBus should treat this as a first-class design problem.

## Proposed architecture

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

The protocol layer should report what happened on C-Bus. It should not own Home Assistant state.

The state engine should consume protocol events and status information and maintain the best-known state of each group.

## State principles

- Unknown is a valid state.
- Absence of information must never be interpreted as OFF.
- Live SAL events should be processed immediately.
- Startup status interrogation must not block the receive loop.
- A startup snapshot must never overwrite a newer live event.
- Synchronisation attempts are generation-scoped. If a group receives a live SAL event after a synchronisation generation begins, status data from that generation must not overwrite that live event unless protocol evidence provides a stronger ordering guarantee.
- A bare group address must not be assumed globally unique; state identity must preserve the C-Bus context required to distinguish network/application/group identity as the implementation exposes it.
- On C-Bus disconnect, a last-known value may be retained for diagnostics, but it is no longer current/authoritative until confirmed by new live or status evidence.
- State should be periodically reconcilable rather than assuming no packets are ever lost.
- For ramps, distinguish the requested target from a definitely known instantaneous level.

## Startup and resynchronisation

The intended startup flow is:

1. Open the PCI/CNI transport.
2. Initialise the interface.
3. Initialise group state as UNKNOWN.
4. Start receiving C-Bus traffic immediately.
5. Request Lighting binary status/MMI.
6. Request Lighting level status/MMI where supported.
7. Merge status replies with live SAL events received during synchronisation.
8. Ensure older snapshot data cannot overwrite newer live events.
9. Publish groups as their state becomes known.
10. Continue processing live SAL events.
11. Use later status reports and/or controlled periodic status requests for reconciliation.

The exact MMI/status behaviour and bus load should be verified against real hardware and available Clipsal documentation rather than assumed from the Python implementation. Hardware verification is used to validate migrated behaviour and resolve uncertainty; it should not be used as an excuse to redesign behaviour that libcbus already handles correctly.

## Initial implementation scope

Before protocol work begins, Slice 1 establishes the minimal `GoCBus` executable/configuration spine: config loading, validation, immediately useful CLI overrides and the common test/capture primitives. Configuration should grow only as later slices require it; this is not a request to build a configuration framework in advance.

The first useful vertical slice is:

```text
connect -> receive raw frames -> decode Lighting ON/OFF/RAMP -> emit typed Go events
```

After that:

1. Add status request/report handling.
2. Add state synchronisation and reconciliation.
3. Add MQTT transport.
4. Add Home Assistant MQTT Discovery.
5. Harden reconnect and failure behaviour.

### Initial protocol scope

- Serial PCI transport
- TCP CNI transport
- PCI reset/initialisation
- Incremental framing
- Checksums and confirmations
- Lighting ON
- Lighting OFF
- Lighting RAMP / level
- Lighting terminate-ramp where practical
- Lighting status requests
- Binary status/MMI parsing
- Level status/MMI parsing
- Preservation/logging of unsupported messages

## Suggested Go responsibility boundaries

```text
cmd/GoCBus/         executable entry point
internal/runtime/   minimal configuration/bootstrap, later full lifecycle integration
internal/capture/   raw traffic capture/replay helpers
internal/transport/ serial PCI and TCP CNI connections
internal/wire/      buffering, framing, checksums and resynchronisation
internal/protocol/  C-Bus packet structures and encoding/decoding
internal/lighting/  typed Lighting events and commands
internal/status/    Lighting status/MMI handling
internal/state/     known/unknown/stale group state and reconciliation
```

These are responsibility boundaries, not a promise to manufacture one package per heading. Prefer a small package structure and simplify topology when implementation evidence supports it.

## Testing strategy

Use libcbus as the behavioural baseline, not an architecture to reproduce.

### Differential migration testing

Where practical, present the same input to libcbus and GoCBus and compare their logical outputs. Differences are failures unless the difference is intentional, the reason is documented, and the GoCBus behaviour is supported by protocol documentation or hardware evidence.

Useful tests include:

- Feed identical captured C-Bus traffic to libcbus and GoCBus and compare logical events.
- Split frames at every possible byte boundary.
- Deliver multiple frames in one read.
- Inject corrupt checksums/noise and verify parser resynchronisation.
- Replay a recorded startup/status sequence.
- Deliver a live SAL event while initial status synchronisation is underway.
- Disconnect and reconnect C-Bus.
- Disconnect and reconnect MQTT.
- Test stale retained MQTT state.
- Test ramps, terminate-ramp and restart during a ramp.

Useful diagnostics should include raw RX/TX frames, decoded messages, state transitions, reconciliation reasons, malformed-frame counters and unknown packet capture.

Real-hardware validation begins with the Transport slice through an explicit opt-in integration harness. Hardware tests must never run as part of the default deterministic unit-test suite. The same harness should be extended by later slices for framing, Lighting transmit, status/MMI and synchronisation validation rather than creating a second application.

## Deliberate non-goals for the first version

- Reimplementing C-Gate.
- Replacing C-Bus Toolkit.
- Supporting every C-Bus application immediately.
- Building a commissioning UI.
- Creating abstractions merely because they may become useful one day.

Get Lighting working reliably first.

## Reference implementation

- Repository: https://github.com/micolous/cbus
- Project: libcbus
- Pinned behavioural reference commit for v0.42: `cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`

The pinned reference should move only by an explicit migration-planning decision. libcbus identifies itself as LGPL-3.0-or-later; GoCBus is licensed GPLv3.

Important reference areas include:

- `cbus/protocol/pciprotocol.py`
- `cbus/daemon/cmqttd.py`
- Lighting SAL implementations
- Status request and status report implementations

## Development principle

The goal is a boring, dependable infrastructure service.

Protocol correctness, recovery and observable behaviour matter more than clever abstractions.
