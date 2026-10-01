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

The foundation slice provides the initial `GoCBus` executable, minimal JSON
configuration, common wire helpers, and deterministic raw traffic
capture/replay support. Transport and protocol handling remain planned work.

See [docs/MigrationPlanning](docs/MigrationPlanning) for the design and migration planning.

## Developing with local Codex

Start with the [macOS setup and slice workflow](docs/CodexLocalSetup.md). Repository instructions live in [AGENTS.md](AGENTS.md); the [v0.42 plan](docs/v0.42/ImplementationPlan.md) links each slice's design and implementation prompt.

Go implementation began with Slice 1. The setup guide distinguishes the
deterministic build/test commands available now from opt-in hardware validation
introduced in Slice 2.

### Foundation configuration

Slice 1 accepts an optional JSON file containing only logging settings:

```json
{
  "log_level": "info",
  "raw_log": false
}
```

Run `GoCBus -config config.json`. The `-log-level` and `-raw-log` flags override
file values. Supported levels are `trace`, `debug`, `info`, `warn`, and `error`.
Raw traffic is disabled by default and is emitted at trace level when enabled.
Its separate gate does not change the configured level for ordinary logs.
The Slice 1 executable validates configuration and exits; transport arrives in
Slice 2.

## Licence

GoCBus is licensed under the **GNU General Public License version 3** (GPLv3). See [LICENSE](LICENSE).

GoCBus is informed by and migrates behaviour from Michael Farrell's `libcbus`, which is licensed under the GNU Lesser General Public License version 3 or later (LGPL-3.0-or-later).

## Acknowledgements

GoCBus originated from the work in [micolous/cbus](https://github.com/micolous/cbus) (`libcbus`) by Michael Farrell.

That project provided both the original inspiration and much of the protocol knowledge used to bootstrap GoCBus.

GoCBus is not affiliated with or endorsed by Clipsal or Schneider Electric.
