# Slice 2 hardware transport harness

This harness is deliberately excluded from default tests. It opens one named
serial PCI or TCP CNI, reads raw bytes for a bounded duration, and writes a
local NDJSON capture. It performs no transmit, framing, packet decoding, PCI
initialisation, device discovery, or automatic reconnect.

Run against a TCP CNI only after selecting the intended target:

```sh
GOCBUS_HARDWARE=1 \
GOCBUS_TCP=192.0.2.1:10001 \
GOCBUS_CAPTURE=/private/tmp/gocbus-hardware.ndjson \
go test -tags=hardware ./integration/hardware
```

For a serial PCI, replace `GOCBUS_TCP` with an explicit device such as
`GOCBUS_SERIAL=/dev/cu.usbserial-example`. `GOCBUS_DURATION` optionally changes
the default `15s` capture window. Write behaviour is covered by deterministic
local transport tests; live transmit testing requires a separately authorised
command and target.

Captures may contain installation-specific data and must be reviewed and
sanitised before committing.
