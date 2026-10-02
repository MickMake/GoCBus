# GoCBus

GoCBus is a Go-based gateway between **Clipsal C-Bus** and **MQTT**, with specific support for the **Home Assistant MQTT framework**.

The aim is to provide a small, reliable and maintainable way to integrate a C-Bus installation with Home Assistant without requiring C-Gate.

## Background

GoCBus began as a Go conversion of Michael Farrell's [libcbus](https://github.com/micolous/cbus) Python project.

GoCBus is being developed as a behaviour-first migration of that work into idiomatic Go, with deliberate improvements where the existing behaviour has an identified reliability or state-synchronisation weakness.

The original libcbus project remains an important reference for the C-Bus protocol implementation.

## Implementation philosophy

GoCBus is intentionally being developed as a behaviour-first migration of libcbus into idiomatic Go.

Where libcbus already implements a C-Bus protocol behaviour successfully, GoCBus should reproduce that observable behaviour before attempting to improve it.

The goal is not a line-for-line Python translation. Python-specific structure, asyncio patterns and class hierarchies should not be preserved merely for familiarity.

Behavioural changes are made deliberately and should be limited to obvious defects, missing reliability behaviour, state-synchronisation problems, reconnect/recovery issues, or architecture that prevents the stated GoCBus goals.

Other enhancements should be deferred until the migrated behaviour is working and verified.

## Goals

GoCBus is intended to:

- Communicate directly with Clipsal C-Bus PCI/CNI interfaces
- Monitor C-Bus network activity
- Send C-Bus commands
- Maintain an accurate representation of C-Bus group state
- Publish C-Bus devices and state through MQTT
- Support Home Assistant MQTT Discovery
- Recover cleanly from C-Bus, MQTT and network interruptions
- Avoid a dependency on C-Gate

## Current Status

**Early implementation.**

Slices 1-4 provide the `GoCBus` executable, minimal JSON configuration,
deterministic raw traffic capture/replay, raw serial PCI or TCP CNI transport,
bounded incremental wire framing, and typed protocol packet encoding/decoding.
Lighting interpretation and PCI initialisation remain planned work.

See [docs/MigrationPlanning](docs/MigrationPlanning) for the design and migration planning.

## Developing with local Codex

Start with the [macOS setup and slice workflow](docs/CodexLocalSetup.md). Repository instructions live in [AGENTS.md](AGENTS.md); the [v0.42 plan](docs/v0.42/ImplementationPlan.md) links each slice's design and implementation prompt.

Go implementation began with Slice 1. The setup guide distinguishes the
deterministic build/test commands available now from opt-in hardware validation
introduced in Slice 2.

### Transport configuration

The configuration accepts logging plus at most one transport:

```json
{
  "log_level": "info",
  "raw_log": false,
  "tcp_address": "192.0.2.1:10001",
  "capture_path": "/private/tmp/gocbus.ndjson"
}
```

Use `serial_device` instead of `tcp_address` for a physical PCI; serial is fixed
at the libcbus-compatible 9600 8N1 settings. Run `GoCBus -config config.json`.
The `-serial`, `-tcp`, `-capture`, `-log-level`, and `-raw-log` flags override
file values. `-serial` and `-tcp` are mutually exclusive. Supported log levels
are `trace`, `debug`, `info`, `warn`, and `error`.

The process connects, records any raw bytes received, and runs in the foreground
until interrupted or disconnected. A disconnect is reported as a failure; this
slice deliberately provides reusable reconnect primitives but no automatic
retry policy. Raw logging is separately gated and does not lower the ordinary
log threshold. Capture files are truncated at startup and written with owner-only
permissions; they may contain installation-specific data.
With no transport configured, `GoCBus` validates its settings and exits without
opening a device or network connection.

The explicit receive-only real-hardware harness is documented in
[integration/hardware](integration/hardware). It never runs with the default
test suite.

### Wire framing

`internal/wire` separates arbitrary stream reads into CRLF-terminated PCI
responses and CR-terminated commands. It also identifies power-up, PCI-error,
and confirmation responses without interpreting higher-level packets. Pending
input is limited to 256 bytes; overflow and incomplete input are returned as
observable diagnostic events. The current executable still operates as a raw
capture path until later integration slices add consumers for framed events.

### Protocol packets

`internal/protocol` translates complete wire events into typed
point-to-multipoint, point-to-point, device-management, reset, power-up,
PCI-error, and confirmation packets. Application SAL and CAL bytes remain raw
for their owning later slices. Malformed or unsupported packets are returned as
explicit unknown packets with their exact wire bytes preserved.

## Licence

GoCBus is licensed under the **GNU General Public License version 3** (GPLv3). See [LICENSE](LICENSE).

GoCBus is informed by and migrates behaviour from Michael Farrell's `libcbus`, which is licensed under the GNU Lesser General Public License version 3 or later (LGPL-3.0-or-later).

## Acknowledgements

GoCBus originated from the work in [micolous/cbus](https://github.com/micolous/cbus) (`libcbus`) by Michael Farrell.

That project provided both the original inspiration and much of the protocol knowledge used to bootstrap GoCBus.

GoCBus is not affiliated with or endorsed by Clipsal or Schneider Electric.
