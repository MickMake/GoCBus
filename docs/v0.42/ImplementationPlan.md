# GoCBus v0.42 - Implementation Plan

## Release intent

v0.42 is the first pre-v1.0 implementation release of GoCBus.

v0.42 is a **behaviour-first Go migration of the libcbus functionality required for a reliable C-Bus Lighting to MQTT/Home Assistant gateway**. The default rule is behavioural compatibility with libcbus. GoCBus may use different internal structure where that is more idiomatic in Go, but observable protocol behaviour should remain compatible unless a deviation is explicitly required to fix an identified defect, reliability problem, or state-synchronisation weakness. libcbus is the primary implementation reference; Clipsal/Schneider protocol documentation and real hardware observations may override it where evidence shows that libcbus is wrong or incomplete.

This document is the gold-standard implementation plan. Deviations must be recorded in `ImplementationState.md`.

libcbus is a source/behavioural reference, not a runtime dependency. GoCBus must not require Python or libcbus to run.

The v0.42 behavioural reference is pinned to libcbus commit `cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`. Do not silently follow upstream changes during implementation.

The fourteen slices are the planned v0.42 roadmap, not a requirement that every slice ship under the same release tag. v0.42 may be cut after any completed slice once the implemented subset forms a useful, stable release. Unshipped slices roll forward to a later release unless implementation evidence requires their design to change.

## Document authority

The root README and migration overview explain purpose and rationale. This plan governs release intent and shared rules; slice designs supply local contracts and implementation prompts execute them. `ImplementationState.md` records agreed decisions, evidence and actual progress. If these disagree, resolve the conflict through change control before implementing the disputed behaviour; a prompt must not silently override a design.

## Locked binary name

`GoCBus` — case intentional.

## Planned package responsibilities

```text
cmd/
  GoCBus/

internal/
  capture/
  transport/
  wire/
  protocol/
  lighting/
  status/
  state/
  mqtt/
  homeassistant/
  passthrough/
  runtime/
  control/
```

Directories need only be created when their owning slice is implemented. These boundaries express intended responsibility, not immutable package topology. A slice may adjust package boundaries where implementation evidence shows that a simpler Go structure is preferable, provided responsibility separation is preserved and the variation is recorded in `ImplementationState.md`.

## Behavioural change policy

### Compatibility-preserving migration

This is the default. Reproduce proven libcbus observable behaviour using idiomatic Go.

### Required corrective change

Allowed when supported by evidence, including startup/state synchronisation defects, reconnect/recovery defects, incomplete confirmation handling, malformed-input handling, known protocol errors, or behaviour contradicted by hardware/protocol documentation. These changes must be documented in `ImplementationState.md`.

### Deferred enhancement

Do not include behaviour merely because it would be cleaner or more general. New C-Bus applications, speculative abstractions, plugin frameworks, broad protocol redesign, expanded discovery mechanisms and unrelated operational features should wait until the migrated behaviour is working and verified.

## Runtime model

GoCBus runs in the foreground by default and behaves cleanly under an external service manager. Daemon/service packaging is secondary. Minimal executable/configuration bootstrap begins in Slice 1 and is extended as later slices require settings. The lifecycle states DOWN, CONNECTING, INITIALISING, SYNCING, UP, PASSTHROUGH, DEGRADED and ERROR describe the **C-Bus lifecycle**, not every subsystem in the process. MQTT maintains its own smaller connection state so orthogonal failures are not forced into one enum.

## Release slices

Each slice should migrate the minimum libcbus behaviour required for that slice. Do not redesign adjacent behaviour unless the current slice cannot be completed reliably without doing so.

1. [v0.42.1 - Foundation and minimal runtime/configuration](slices/v0.42.1-design.md) - [implementation prompt](slices/v0.42.1-implementation-prompt.md)
2. [v0.42.2 - Transport](slices/v0.42.2-design.md) - [implementation prompt](slices/v0.42.2-implementation-prompt.md)
3. [v0.42.3 - Wire and framing](slices/v0.42.3-design.md) - [implementation prompt](slices/v0.42.3-implementation-prompt.md)
4. [v0.42.4 - Protocol packets](slices/v0.42.4-design.md) - [implementation prompt](slices/v0.42.4-implementation-prompt.md)
5. [v0.42.5 - Lighting receive](slices/v0.42.5-design.md) - [implementation prompt](slices/v0.42.5-implementation-prompt.md)
6. [v0.42.6 - Lighting transmit and PCI initialisation](slices/v0.42.6-design.md) - [implementation prompt](slices/v0.42.6-implementation-prompt.md)
7. [v0.42.7 - Status and MMI](slices/v0.42.7-design.md) - [implementation prompt](slices/v0.42.7-implementation-prompt.md)
8. [v0.42.8 - State engine](slices/v0.42.8-design.md) - [implementation prompt](slices/v0.42.8-implementation-prompt.md)
9. [v0.42.9 - MQTT gateway](slices/v0.42.9-design.md) - [implementation prompt](slices/v0.42.9-implementation-prompt.md)
10. [v0.42.10 - Home Assistant](slices/v0.42.10-design.md) - [implementation prompt](slices/v0.42.10-implementation-prompt.md)
11. [v0.42.11 - Maintenance passthrough](slices/v0.42.11-design.md) - [implementation prompt](slices/v0.42.11-implementation-prompt.md)
12. [v0.42.12 - Runtime integration and configuration completion](slices/v0.42.12-design.md) - [implementation prompt](slices/v0.42.12-implementation-prompt.md)
13. [v0.42.13 - Operational control and status](slices/v0.42.13-design.md) - [implementation prompt](slices/v0.42.13-implementation-prompt.md)
14. [v0.42.14 - Hardening and release readiness](slices/v0.42.14-design.md) - [implementation prompt](slices/v0.42.14-implementation-prompt.md)

Each design file is the local design contract. Its implementation prompt starts the coding session for that slice.

## Integration milestones and release cuts

Each slice integrates its capability into the existing `GoCBus` executable where meaningful, including the minimal configuration and lifecycle plumbing it needs. Owned-area lists identify primary responsibilities, not a prohibition on that plumbing. Pure decoding may be demonstrated through replay. Do not create another production application or pull unrelated later features forward.

- Slice 2 proves raw connection/capture; without PCI initialisation it must record any required interface setup assumptions.
- Slice 5 proves Lighting decoding through replay; it does not promise self-initialising live monitoring.
- Slice 6 establishes the first self-initialising live Lighting path using the existing receive decoder and hardware harness.
- Slices 7-11 extend that working path. Slice 12 consolidates runtime/configuration integration rather than assembling the first usable gateway.
- Introduce cancellation, clean shutdown and failure handling with each resource-owning slice. Slice 13 exposes existing lifecycle truth through controls/status rather than inventing it late.

For any selected release cut, require:

1. One runnable binary with its implemented subset, setup and limitations documented.
2. A demonstrated integrated path and clean startup/shutdown for that subset.
3. Passing deterministic tests and relevant failure/recovery checks for shipped capabilities.
4. Hardware evidence for hardware-dependent claims, or an explicit unverified limitation.
5. The applicable assembled-system checks from Slice 14, with results and remaining work recorded in `ImplementationState.md`.

These gates do not require all fourteen capabilities or speculative timing/API decisions before their owning slice.

## State synchronisation

State synchronisation is an intentional GoCBus behavioural improvement over the current libcbus MQTT gateway and is therefore not constrained to reproduce its startup behaviour.

Receive live traffic before status interrogation. Groups begin UNKNOWN. Binary/level status replies establish state while live SAL continues. Synchronisation is generation-scoped: if a group receives a live SAL event after a sync generation begins, status data from that generation cannot overwrite the live event unless protocol evidence provides a stronger ordering guarantee. A bare group number must not be treated as globally unique; the state key must preserve the C-Bus identity required to distinguish network/application/group context. On C-Bus disconnect, last-known values may remain available for diagnostics but become stale/non-authoritative until new live or status evidence confirms them.

## Command confirmation semantics

A successful PCI/protocol confirmation means that the command was accepted/confirmed at the interface/protocol level. It must not, by itself, be treated as proof that a physical load has reached the requested state. Authoritative Lighting state changes require live SAL/status evidence or another protocol guarantee explicitly validated against documentation/hardware.

## Hardware validation

An explicit opt-in real-hardware integration harness begins in Slice 2. It is extended by later slices for framing, transmit, status/MMI and synchronisation testing. Hardware tests must never run as part of the default deterministic test suite and must not grow into a second production executable.

## Maintenance passthrough

v0.42 deliberately avoids simultaneous virtual-CNI multiplexing.

```text
Normal:
Physical C-Bus <-> GoCBus <-> MQTT / Home Assistant

Passthrough:
Physical C-Bus <-> GoCBus TCP bridge <-> C-Gate / Toolkit
```

Normal C-Bus processing is suspended during passthrough. On exit GoCBus reclaims and reinitialises the interface and performs a full state resynchronisation.

## Operational control

MQTT/Home Assistant expose useful controls rather than every internal knob: passthrough, debug/raw logging, force-resync and safe reconnect controls. Status exposes the C-Bus lifecycle state separately from MQTT connection state, together with useful health timestamps/counters.

## Deliberate non-goals

- Replacing Toolkit or reimplementing C-Gate.
- Every C-Bus application.
- Full simultaneous GoCBus/C-Gate virtual-CNI multiplexing.
- CNI discovery unless real usage proves it necessary.
- Premature plugin/framework architecture.

## Change control

When evidence breaks the plan: record it in `ImplementationState.md`, describe the variation and reason, agree the change, then update the relevant design. Do not silently drift.

Whole-earth rupture exemptions remain available.

## Licence and attribution

GoCBus is licensed under GNU GPLv3. The libcbus reference project is LGPL-3.0-or-later and remains attributed to Michael Farrell. Behavioural/source reference for v0.42 remains pinned to the commit recorded above.
