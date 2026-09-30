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

**Early development.**

The initial implementation is focused on the C-Bus Lighting application and establishing a reliable protocol, transport and state-management foundation.

See [docs/MigrationPlanning](docs/MigrationPlanning) for the design and migration planning.

## Acknowledgements

GoCBus originated from the work in [micolous/cbus](https://github.com/micolous/cbus) (`libcbus`) by Michael Farrell.

That project provided both the original inspiration and much of the protocol knowledge used to bootstrap GoCBus.

GoCBus is not affiliated with or endorsed by Clipsal or Schneider Electric.
