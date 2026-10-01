package protocol

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/MickMake/GoCBus/internal/wire"
)

func TestDecodePointToMultipointLibCBusFixtures(t *testing.T) {
	tests := []struct {
		name         string
		kind         wire.EventKind
		data         string
		direction    wire.Direction
		source       byte
		application  wire.ApplicationID
		payload      []byte
		confirmation byte
	}{
		{
			name:         "Serial Interface Guide lighting command",
			kind:         wire.Command,
			data:         "\\0538000108BAg\r",
			direction:    wire.ToPCI,
			application:  wire.ApplicationLighting,
			payload:      []byte{0x01, 0x08},
			confirmation: 'g',
		},
		{
			name:         "Serial Interface Guide status request",
			kind:         wire.Command,
			data:         "\\05FF007A38004Ah\r",
			direction:    wire.ToPCI,
			application:  0xff,
			payload:      []byte{0x7a, 0x38, 0x00},
			confirmation: 'h',
		},
		{
			name:        "libcbus null lighting regression",
			kind:        wire.Response,
			data:        "05063800BD\r\n",
			direction:   wire.FromPCI,
			source:      0x06,
			application: wire.ApplicationLighting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, ok := Decode(wire.Event{Kind: tt.kind, Data: []byte(tt.data)}).(*PointToMultipoint)
			if !ok {
				t.Fatalf("packet = %#v, want PointToMultipoint", packet)
			}
			if packet.Direction != tt.direction || packet.SourceAddress != tt.source ||
				packet.Application != tt.application || packet.Confirmation != tt.confirmation ||
				!packet.Checksum || !bytes.Equal(packet.Payload, tt.payload) {
				t.Fatalf("packet = %#v, want direction %d source %02X application %02X payload % X confirmation %q",
					packet, tt.direction, tt.source, tt.application, tt.payload, tt.confirmation)
			}
			assertEncoded(t, packet, []byte(tt.data))
		})
	}
}

func TestDecodePointToPointLibCBusReplyFixture(t *testing.T) {
	data := []byte("8604990082300328\r\n")
	packet, ok := Decode(wire.Event{Kind: wire.Response, Data: data}).(*PointToPoint)
	if !ok {
		t.Fatalf("packet = %#v, want PointToPoint", packet)
	}
	if packet.Direction != wire.FromPCI || packet.Priority != Priority2 ||
		packet.SourceAddress != 0x04 || packet.UnitAddress != 0x99 || packet.Bridged ||
		!packet.Checksum || !bytes.Equal(packet.CAL, []byte{0x82, 0x30, 0x03}) {
		t.Fatalf("packet = %#v", packet)
	}
	assertEncoded(t, packet, data)
}

func TestPointToPointBridgedRoundTrip(t *testing.T) {
	want := &PointToPoint{
		Header: Header{
			Direction:    wire.ToPCI,
			Priority:     Priority3,
			Checksum:     true,
			Confirmation: 'h',
		},
		Bridged:       true,
		BridgeAddress: 0x10,
		Hops:          []byte{0x20, 0x30},
		UnitAddress:   0x40,
		CAL:           []byte{0x21, 0x01},
	}
	encoded, err := Encode(want)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Decode(wire.Event{Kind: wire.Command, Data: encoded}).(*PointToPoint)
	if !ok {
		t.Fatalf("packet = %#v, want PointToPoint", got)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("packet = %#v, want %#v", got, want)
	}
	assertEncoded(t, got, encoded)
}

func TestDecodeDeviceManagementLibCBusFixtures(t *testing.T) {
	tests := []struct {
		name         string
		kind         wire.EventKind
		data         string
		direction    wire.Direction
		source       byte
		checksum     bool
		confirmation byte
	}{
		{
			name:         "basic mode command",
			kind:         wire.Command,
			data:         "A3210038g\r",
			direction:    wire.ToPCI,
			confirmation: 'g',
		},
		{
			name:      "PCI response",
			kind:      wire.Response,
			data:      "A30421003800\r\n",
			direction: wire.FromPCI,
			source:    0x04,
			checksum:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, ok := Decode(wire.Event{Kind: tt.kind, Data: []byte(tt.data)}).(*DeviceManagement)
			if !ok {
				t.Fatalf("packet = %#v, want DeviceManagement", packet)
			}
			if packet.Direction != tt.direction || packet.Priority != Priority2 ||
				packet.SourceAddress != tt.source || packet.Checksum != tt.checksum ||
				packet.Confirmation != tt.confirmation || packet.Parameter != 0x21 || packet.Value != 0x38 {
				t.Fatalf("packet = %#v", packet)
			}
			assertEncoded(t, packet, []byte(tt.data))
		})
	}
}

func TestSpecialPacketsMatchLibCBus(t *testing.T) {
	tests := []struct {
		name       string
		event      wire.Event
		packetType Type
		encoded    string
	}{
		{"power-up", wire.Event{Kind: wire.PowerUp, Data: []byte("+")}, TypePowerUp, "++"},
		{"PCI error", wire.Event{Kind: wire.PCIError, Data: []byte("!")}, TypePCIError, "!"},
		{"successful confirmation", wire.Event{Kind: wire.Confirmation, Data: []byte("g.")}, TypeConfirmation, "g."},
		{"failed confirmation canonicalises suffix", wire.Event{Kind: wire.Confirmation, Data: []byte("z?")}, TypeConfirmation, "z#"},
		{"reset", wire.Event{Kind: wire.Command, Data: []byte("~\r")}, TypeReset, "~\r"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := Decode(tt.event)
			if packet.Type() != tt.packetType {
				t.Fatalf("packet type = %d, want %d", packet.Type(), tt.packetType)
			}
			assertEncoded(t, packet, []byte(tt.encoded))
		})
	}
}

func TestUnknownPacketsRemainObservable(t *testing.T) {
	tests := []wire.Event{
		{Kind: wire.Response, Data: []byte("05063800BE\r\n")},   // bad checksum
		{Kind: wire.Response, Data: []byte("05063800bd\r\n")},   // lowercase hex
		{Kind: wire.Response, Data: []byte("0001AA55\r\n")},     // unsupported address type
		{Kind: wire.Response, Data: []byte("05063801BC\r\n")},   // routing data
		{Kind: wire.Response, Data: []byte("0D063800B5\r\n")},   // reserved flag bit
		{Kind: wire.Response, Data: []byte("A504210038FE\r\n")}, // DP with wrong address type
		{Kind: wire.Incomplete, Data: []byte("0506")},
	}

	for _, event := range tests {
		packet, ok := Decode(event).(*Unknown)
		if !ok {
			t.Fatalf("Decode(%q) = %#v, want Unknown", event.Data, packet)
		}
		if packet.Reason == "" || packet.Event.Kind != event.Kind || !bytes.Equal(packet.Event.Data, event.Data) {
			t.Fatalf("unknown packet = %#v, want preserved event %#v", packet, event)
		}
		assertEncoded(t, packet, event.Data)
	}
}

func TestDecodeThroughWireFramer(t *testing.T) {
	framer, err := wire.NewFramer(wire.ToPCI)
	if err != nil {
		t.Fatal(err)
	}
	events := framer.Feed([]byte("\\0538000108BAg\r"))
	if len(events) != 1 {
		t.Fatalf("events = %#v, want one", events)
	}
	packet, ok := Decode(events[0]).(*PointToMultipoint)
	if !ok || packet.Application != wire.ApplicationLighting || !bytes.Equal(packet.Payload, []byte{0x01, 0x08}) {
		t.Fatalf("packet = %#v", packet)
	}
}

func TestDecodedPayloadDoesNotAliasInput(t *testing.T) {
	data := []byte("05063800BD\r\n")
	packet := Decode(wire.Event{Kind: wire.Response, Data: data}).(*PointToMultipoint)
	data[6] = 'F'
	assertEncoded(t, packet, []byte("05063800BD\r\n"))

	unknownData := []byte("NOT-HEX\r\n")
	unknown := Decode(wire.Event{Kind: wire.Response, Data: unknownData}).(*Unknown)
	unknownData[0] = 'X'
	assertEncoded(t, unknown, []byte("NOT-HEX\r\n"))
}

func TestEncodeRejectsInvalidPackets(t *testing.T) {
	tests := []struct {
		name   string
		packet Packet
	}{
		{
			name: "direction",
			packet: &PointToMultipoint{
				Header: Header{Direction: 99},
			},
		},
		{
			name: "priority",
			packet: &PointToMultipoint{
				Header: Header{Direction: wire.ToPCI, Priority: 4},
			},
		},
		{
			name: "confirmation code",
			packet: &PointToMultipoint{
				Header: Header{Direction: wire.ToPCI, Confirmation: 'a'},
			},
		},
		{
			name: "source confirmation",
			packet: &PointToMultipoint{
				Header: Header{Direction: wire.FromPCI, Confirmation: 'g'},
			},
		},
		{
			name: "bridge address",
			packet: &PointToPoint{
				Header:  Header{Direction: wire.ToPCI},
				Bridged: true,
			},
		},
		{
			name: "hop count",
			packet: &PointToPoint{
				Header:        Header{Direction: wire.ToPCI},
				Bridged:       true,
				BridgeAddress: 1,
				Hops:          make([]byte, 6),
			},
		},
		{
			name:   "confirmation packet code",
			packet: &Confirmation{Code: 'a'},
		},
		{
			name:   "unknown without data",
			packet: &Unknown{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Encode(tt.packet); err == nil {
				t.Fatalf("Encode(%#v) error = nil", tt.packet)
			}
		})
	}
	if _, err := Encode(nil); err == nil {
		t.Fatal("Encode(nil) error = nil")
	}

	typedNilPackets := []Packet{
		(*PointToMultipoint)(nil),
		(*PointToPoint)(nil),
		(*DeviceManagement)(nil),
		(*Reset)(nil),
		(*PowerUp)(nil),
		(*PCIError)(nil),
		(*Confirmation)(nil),
		(*Unknown)(nil),
	}
	for index, packet := range typedNilPackets {
		if _, err := Encode(packet); err == nil {
			t.Errorf("typed nil packet %d error = nil", index)
		}
	}
}

func assertEncoded(t *testing.T, packet Packet, want []byte) {
	t.Helper()
	got, err := Encode(packet)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("encoded = %q (% X), want %q (% X)", got, got, want, want)
	}
}
