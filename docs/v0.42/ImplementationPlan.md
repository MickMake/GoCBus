# GoCBus v0.42 - Implementation Plan

## Release intent

v0.42 is the first pre-v1.0 implementation release of GoCBus.

v0.42 is a **behaviour-first Go migration of the libcbus functionality required for a reliable C-Bus Lighting to MQTT/Home Assistant gateway**. The default rule is behavioural compatibility with libcbus. GoCBus may use different internal structure where that is more idiomatic in Go, but observable protocol behaviour should remain compatible unless a deviation is explicitly required to fix an identified defect, reliability problem, or state-synchronisation weakness. libcbus is the primary implementation reference; Clipsal/Schneider protocol documentation and real hardware observations may override it where evidence shows that libcbus is wrong or incomplete.

This document is the gold-standard implementation plan. Deviations must be recorded in `ImplementationState.md`.

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

GoCBus runs in the foreground by default and behaves cleanly under an external service manager. Daemon/service packaging is secondary. Runtime/C-Bus lifecycle state is authoritative and includes DOWN, CONNECTING, INITIALISING, SYNCING, UP, PASSTHROUGH, DEGRADED and ERROR.

## Release slices

Each slice should migrate the minimum libcbus behaviour required for that slice. Do not redesign adjacent behaviour unless the current slice cannot be completed reliably without doing so.

1. [v0.42.1 - Foundation](slices/v0.42.1-design.md) - [implementation prompt](slices/v0.42.1-implementation-prompt.md)
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
12. [v0.42.12 - Runtime and configuration](slices/v0.42.12-design.md) - [implementation prompt](slices/v0.42.12-implementation-prompt.md)
13. [v0.42.13 - Operational control and status](slices/v0.42.13-design.md) - [implementation prompt](slices/v0.42.13-implementation-prompt.md)
14. [v0.42.14 - Hardening and release readiness](slices/v0.42.14-design.md) - [implementation prompt](slices/v0.42.14-implementation-prompt.md)

Each design file is the local design contract. Its implementation prompt starts the coding session for that slice.

## State synchronisation

State synchronisation is an intentional GoCBus behavioural improvement over the current libcbus MQTT gateway and is therefore not constrained to reproduce its startup behaviour.

Receive live traffic before status interrogation. Groups begin UNKNOWN. Binary/level status replies establish state while live SAL continues. Older snapshot information must never overwrite a newer live event.

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

MQTT/Home Assistant expose useful controls rather than every internal knob: passthrough, debug/raw logging, force-resync and safe reconnect controls. Status includes the C-Bus/runtime lifecycle state and useful health timestamps/counters.

## Deliberate non-goals

- Replacing Toolkit or reimplementing C-Gate.
- Every C-Bus application.
- Full simultaneous GoCBus/C-Gate virtual-CNI multiplexing.
- CNI discovery unless real usage proves it necessary.
- Premature plugin/framework architecture.

## Change control

When evidence breaks the plan: record it in `ImplementationState.md`, describe the variation and reason, agree the change, then update the relevant design. Do not silently drift.

Whole-earth rupture exemptions remain available.
