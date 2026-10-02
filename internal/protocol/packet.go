// Package protocol encodes and decodes C-Bus serial-interface packets.
package protocol

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/MickMake/GoCBus/internal/wire"
)

// Type identifies a decoded protocol packet.
type Type uint8

const (
	TypePointToMultipoint Type = iota + 1
	TypePointToPoint
	TypeDeviceManagement
	TypeReset
	TypePowerUp
	TypePCIError
	TypeConfirmation
	TypeUnknown
)

// Priority is the C-Bus priority class. Priority4 is the lowest priority.
type Priority uint8

const (
	Priority4 Priority = iota
	Priority3
	Priority2
	Priority1
)

// Header contains the common serial-interface packet metadata.
type Header struct {
	Direction     wire.Direction
	Priority      Priority
	SourceAddress byte
	Checksum      bool
	Confirmation  byte
}

// Packet is one typed or preserved C-Bus serial-interface packet.
type Packet interface {
	Type() Type
	packet()
}

// PointToMultipoint contains an application payload. Application semantics are
// deliberately left to the owning application package.
type PointToMultipoint struct {
	Header
	Application wire.ApplicationID
	Payload     []byte
}

func (*PointToMultipoint) Type() Type { return TypePointToMultipoint }
func (*PointToMultipoint) packet()    {}

// PointToPoint contains addressing and raw CAL bytes. CAL interpretation is
// deliberately left to the owning status/CAL packages.
type PointToPoint struct {
	Header
	UnitAddress   byte
	Bridged       bool
	BridgeAddress byte
	Hops          []byte
	CAL           []byte
}

func (*PointToPoint) Type() Type { return TypePointToPoint }
func (*PointToPoint) packet()    {}

// DeviceManagement sets one PCI parameter to one byte value.
type DeviceManagement struct {
	Header
	Parameter byte
	Value     byte
}

func (*DeviceManagement) Type() Type { return TypeDeviceManagement }
func (*DeviceManagement) packet()    {}

type Reset struct{}

func (*Reset) Type() Type { return TypeReset }
func (*Reset) packet()    {}

type PowerUp struct{}

func (*PowerUp) Type() Type { return TypePowerUp }
func (*PowerUp) packet()    {}

type PCIError struct{}

func (*PCIError) Type() Type { return TypePCIError }
func (*PCIError) packet()    {}

// Confirmation reports whether the PCI accepted the command associated with
// Code. It does not prove that a physical load reached a requested state.
type Confirmation struct {
	Code    byte
	Success bool
}

func (*Confirmation) Type() Type { return TypeConfirmation }
func (*Confirmation) packet()    {}

// Unknown preserves an unsupported or malformed wire event exactly.
type Unknown struct {
	Event  wire.Event
	Reason string
}

func (*Unknown) Type() Type { return TypeUnknown }
func (*Unknown) packet()    {}

// Decode translates one complete wire event into a protocol packet. Malformed
// and unsupported input is returned as Unknown so it remains observable.
// Behaviour is derived from Michael Farrell's libcbus packet modules at
// cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1 (LGPL-3.0-or-later).
func Decode(event wire.Event) Packet {
	switch event.Kind {
	case wire.PowerUp:
		if string(event.Data) == "+" {
			return &PowerUp{}
		}
		return unknown(event, "invalid power-up response")
	case wire.PCIError:
		if string(event.Data) == "!" {
			return &PCIError{}
		}
		return unknown(event, "invalid PCI error response")
	case wire.Confirmation:
		if len(event.Data) != 2 || !isConfirmationCode(event.Data[0]) {
			return unknown(event, "invalid confirmation response")
		}
		return &Confirmation{Code: event.Data[0], Success: event.Data[1] == '.'}
	case wire.Command:
		if string(event.Data) == "~\r" {
			return &Reset{}
		}
		return decodeRegular(event, wire.ToPCI)
	case wire.Response:
		return decodeRegular(event, wire.FromPCI)
	default:
		return unknown(event, "wire event is not a complete protocol packet")
	}
}

// Encode serialises a packet into its complete serial-interface wire form.
func Encode(packet Packet) ([]byte, error) {
	switch value := packet.(type) {
	case *PointToMultipoint:
		if value == nil {
			return nil, errors.New("cannot encode a nil point-to-multipoint packet")
		}
		body := make([]byte, 0, len(value.Payload)+2)
		body = append(body, byte(value.Application), 0)
		body = append(body, value.Payload...)
		return encodeRegular(value.Header, 0x05, false, body, false)
	case *PointToPoint:
		if value == nil {
			return nil, errors.New("cannot encode a nil point-to-point packet")
		}
		body, err := encodePointToPoint(value)
		if err != nil {
			return nil, err
		}
		return encodeRegular(value.Header, 0x06, false, body, false)
	case *DeviceManagement:
		if value == nil {
			return nil, errors.New("cannot encode a nil device-management packet")
		}
		body := []byte{value.Parameter, 0, value.Value}
		return encodeRegular(value.Header, 0x03, true, body, true)
	case *Reset:
		if value == nil {
			return nil, errors.New("cannot encode a nil reset packet")
		}
		return []byte("~\r"), nil
	case *PowerUp:
		if value == nil {
			return nil, errors.New("cannot encode a nil power-up packet")
		}
		// libcbus emits two bytes because the first may be lost to bit errors.
		return []byte("++"), nil
	case *PCIError:
		if value == nil {
			return nil, errors.New("cannot encode a nil PCI-error packet")
		}
		return []byte("!"), nil
	case *Confirmation:
		if value == nil {
			return nil, errors.New("cannot encode a nil confirmation packet")
		}
		if !isConfirmationCode(value.Code) {
			return nil, fmt.Errorf("invalid confirmation code: %q", value.Code)
		}
		result := []byte{value.Code, '#'}
		if value.Success {
			result[1] = '.'
		}
		return result, nil
	case *Unknown:
		if value == nil {
			return nil, errors.New("cannot encode a nil unknown packet")
		}
		if len(value.Event.Data) == 0 {
			return nil, errors.New("unknown packet has no wire data")
		}
		return append([]byte(nil), value.Event.Data...), nil
	case nil:
		return nil, errors.New("cannot encode a nil packet")
	default:
		return nil, fmt.Errorf("unsupported packet type %T", packet)
	}
}

func decodeRegular(event wire.Event, direction wire.Direction) Packet {
	data := event.Data
	terminator := []byte(wire.EndCommand)
	if direction == wire.FromPCI {
		terminator = []byte(wire.EndResponse)
	}
	if len(data) < len(terminator) || string(data[len(data)-len(terminator):]) != string(terminator) {
		return unknown(event, "missing wire terminator")
	}
	encoded := append([]byte(nil), data[:len(data)-len(terminator)]...)

	basic := false
	confirmation := byte(0)
	if direction == wire.ToPCI {
		if len(encoded) > 0 && encoded[0] == '\\' {
			encoded = encoded[1:]
		} else {
			basic = true
		}
		if len(encoded) > 0 && !isUpperHex(encoded[len(encoded)-1]) {
			confirmation = encoded[len(encoded)-1]
			if !isConfirmationCode(confirmation) {
				return unknown(event, "confirmation code is not in range g..z")
			}
			encoded = encoded[:len(encoded)-1]
		}
	}

	if len(encoded) == 0 || len(encoded)%2 != 0 {
		return unknown(event, "packet has an empty or odd-length hexadecimal payload")
	}
	for _, value := range encoded {
		if !isUpperHex(value) {
			return unknown(event, "packet contains non-uppercase-hexadecimal input")
		}
	}
	binary := make([]byte, hex.DecodedLen(len(encoded)))
	if _, err := hex.Decode(binary, encoded); err != nil {
		return unknown(event, "packet hexadecimal decoding failed")
	}
	if len(binary) == 0 {
		return unknown(event, "packet has no flags byte")
	}

	dp := binary[0]&0x20 != 0
	hasChecksum := true
	if direction == wire.ToPCI && basic && dp && len(binary) == 4 {
		hasChecksum = false
	}
	if hasChecksum {
		if !wire.ValidateChecksum(binary) {
			return unknown(event, "C-Bus checksum is invalid")
		}
		binary = binary[:len(binary)-1]
	}
	if len(binary) == 0 {
		return unknown(event, "packet contains only a checksum")
	}

	flags := binary[0]
	if flags&0x18 != 0 {
		return unknown(event, "packet uses reserved flags")
	}
	header := Header{
		Direction:    direction,
		Priority:     Priority((flags >> 6) & 0x03),
		Checksum:     hasChecksum,
		Confirmation: confirmation,
	}
	body := binary[1:]
	if direction == wire.FromPCI {
		if len(body) == 0 {
			return unknown(event, "packet has no source address")
		}
		header.SourceAddress = body[0]
		body = body[1:]
	}

	if dp {
		if flags&0x07 != 0x03 {
			return unknown(event, "device-management packet has an invalid destination address type")
		}
		if len(body) != 3 || body[1] != 0 {
			return unknown(event, "invalid device-management payload")
		}
		return &DeviceManagement{Header: header, Parameter: body[0], Value: body[2]}
	}

	switch flags & 0x07 {
	case 0x05:
		if len(body) < 2 || body[1] != 0 {
			return unknown(event, "invalid point-to-multipoint routing data")
		}
		return &PointToMultipoint{
			Header:      header,
			Application: wire.ApplicationID(body[0]),
			Payload:     append([]byte(nil), body[2:]...),
		}
	case 0x06:
		return decodePointToPoint(event, header, body)
	default:
		return unknown(event, fmt.Sprintf("unsupported destination address type 0x%02X", flags&0x07))
	}
}

func decodePointToPoint(event wire.Event, header Header, body []byte) Packet {
	if len(body) < 2 {
		return unknown(event, "point-to-point payload is too short")
	}
	packet := &PointToPoint{Header: header}
	if body[1] == 0 {
		packet.UnitAddress = body[0]
		packet.CAL = append([]byte(nil), body[2:]...)
		return packet
	}

	hopCount, ok := bridgeHopCount(body[1])
	if !ok || len(body) < 3+hopCount {
		return unknown(event, "invalid point-to-point bridge routing data")
	}
	packet.Bridged = true
	packet.BridgeAddress = body[0]
	packet.Hops = append([]byte(nil), body[2:2+hopCount]...)
	packet.UnitAddress = body[2+hopCount]
	packet.CAL = append([]byte(nil), body[3+hopCount:]...)
	return packet
}

func encodePointToPoint(packet *PointToPoint) ([]byte, error) {
	body := make([]byte, 0, len(packet.CAL)+len(packet.Hops)+3)
	if packet.Bridged {
		if packet.BridgeAddress == 0 {
			return nil, errors.New("bridged point-to-point packet requires a bridge address")
		}
		routing, ok := bridgeRouting(len(packet.Hops))
		if !ok {
			return nil, fmt.Errorf("point-to-point hop count out of range: %d", len(packet.Hops))
		}
		body = append(body, packet.BridgeAddress, routing)
		body = append(body, packet.Hops...)
		body = append(body, packet.UnitAddress)
	} else {
		if packet.BridgeAddress != 0 || len(packet.Hops) != 0 {
			return nil, errors.New("unbridged point-to-point packet cannot contain bridge routing")
		}
		body = append(body, packet.UnitAddress, 0)
	}
	body = append(body, packet.CAL...)
	return body, nil
}

func encodeRegular(header Header, destination byte, dp bool, body []byte, basic bool) ([]byte, error) {
	if header.Direction != wire.FromPCI && header.Direction != wire.ToPCI {
		return nil, fmt.Errorf("invalid packet direction: %d", header.Direction)
	}
	if !header.Checksum && !(header.Direction == wire.ToPCI && basic && dp) {
		return nil, errors.New("regular packet requires a checksum")
	}
	if header.Priority > Priority1 {
		return nil, fmt.Errorf("invalid priority class: %d", header.Priority)
	}
	if header.Direction == wire.FromPCI && header.Confirmation != 0 {
		return nil, errors.New("packets from the PCI cannot request confirmation")
	}
	if header.Confirmation != 0 && !isConfirmationCode(header.Confirmation) {
		return nil, fmt.Errorf("invalid confirmation code: %q", header.Confirmation)
	}

	flags := destination | byte(header.Priority)<<6
	if dp {
		flags |= 0x20
	}
	binary := make([]byte, 0, len(body)+3)
	binary = append(binary, flags)
	if header.Direction == wire.FromPCI {
		binary = append(binary, header.SourceAddress)
	}
	binary = append(binary, body...)
	if header.Checksum {
		binary = wire.AppendChecksum(binary)
	}

	encoded := strings.ToUpper(hex.EncodeToString(binary))
	result := make([]byte, 0, len(encoded)+4)
	if header.Direction == wire.ToPCI && !basic {
		result = append(result, '\\')
	}
	result = append(result, encoded...)
	if header.Direction == wire.ToPCI && header.Confirmation != 0 {
		result = append(result, header.Confirmation)
	}
	if header.Direction == wire.FromPCI {
		result = append(result, wire.EndResponse...)
	} else {
		result = append(result, wire.EndCommand...)
	}
	return result, nil
}

func unknown(event wire.Event, reason string) *Unknown {
	copy := event
	copy.Data = append([]byte(nil), event.Data...)
	return &Unknown{Event: copy, Reason: reason}
}

func isUpperHex(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'A' && value <= 'F'
}

func isConfirmationCode(value byte) bool {
	return value >= 'g' && value <= 'z'
}

func bridgeHopCount(routing byte) (int, bool) {
	switch routing {
	case 0x09:
		return 0, true
	case 0x12:
		return 1, true
	case 0x1b:
		return 2, true
	case 0x24:
		return 3, true
	case 0x2d:
		return 4, true
	case 0x36:
		return 5, true
	default:
		return 0, false
	}
}

func bridgeRouting(hops int) (byte, bool) {
	values := [...]byte{0x09, 0x12, 0x1b, 0x24, 0x2d, 0x36}
	if hops < 0 || hops >= len(values) {
		return 0, false
	}
	return values[hops], true
}
