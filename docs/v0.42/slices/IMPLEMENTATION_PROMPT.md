# GoCBus implementation prompt

I want to develop GoCBus, a Go gateway for direct Clipsal C-Bus communication over MQTT with specific support for Home Assistant's MQTT framework.

The project began as a migration from Michael Farrell's `micolous/cbus` / libcbus Python project:

https://github.com/micolous/cbus

Reference snapshot reviewed during initial planning:

`cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1`

## Direction

Build a small, dependable Go implementation that talks directly to C-Bus PCI/CNI hardware without C-Gate.

Start with the Lighting application. Do not attempt to recreate C-Gate, Toolkit or every C-Bus application.

Use libcbus as the primary behavioural baseline. For behaviour already implemented and proven in libcbus, reproduce the same observable result before attempting improvements.

Do not translate Python class structure, asyncio organisation or implementation patterns line-for-line. Translate behaviour into the simplest idiomatic Go structure that preserves it.

Do not change protocol behaviour merely because another design appears cleaner. Where GoCBus intentionally differs from libcbus, document the difference, reason and supporting evidence in `ImplementationState.md`.

## Important state-synchronisation issue

The old Python implementation has a weakness for a stateful consumer such as Home Assistant:

- `PCIProtocol.pci_reset()` deliberately configures the interface with MMI reporting disabled.
- The protocol can construct a Lighting `StatusRequestSAL` with a level request, but the call in `timesync()` is commented out.
- `cmqttd` derives MQTT state primarily from live ON/OFF/RAMP callbacks rather than performing a startup state interrogation.

The Go design must correct this.

## Architecture

Keep these concerns separate:

```text
transport -> framing -> protocol encode/decode -> typed events -> state engine -> MQTT
```

The protocol decoder should be as stateless as practical. A separate state engine owns the best-known current state.

Unknown is a real state. Never assume OFF simply because no state has been observed.

## First milestone

Implement only this vertical slice first:

```text
connect -> receive raw frames -> decode Lighting ON/OFF/RAMP -> emit typed Go events
```

Do not add MQTT or Home Assistant until that works reliably.

Afterwards add status/MMI synchronisation and state reconciliation, then MQTT and Home Assistant discovery.

## Startup/state requirements

Eventually startup should:

1. Connect and initialise the PCI/CNI.
2. Start the receive loop immediately.
3. Mark group state UNKNOWN.
4. Request Lighting binary and level status where supported.
5. Merge status replies with live SAL events.
6. Never allow an older startup snapshot to overwrite a newer live event.
7. Publish state progressively as it becomes known.
8. Continue reconciliation after startup.

## Testing

Build capture/replay support early.

Test fragmented reads, combined frames, corrupt data, checksums, resynchronisation, startup races, reconnects and ramps.

Where possible, feed identical captured traffic into libcbus and GoCBus and compare the resulting logical events.

## Working style

Keep it idiomatic and simple.

Before implementing code:

1. Inspect the latest `main`.
2. Inspect prior and currently open PRs so existing decisions are not accidentally undone.
3. Identify the minimum libcbus files/functions needed for the next vertical slice.
4. Identify the exact libcbus behaviour being migrated and the tests/fixtures that can demonstrate compatibility.
5. Identify protocol facts that still require verification.
6. Propose the branch name and implementation plan.
7. Stop for approval before making code changes.
