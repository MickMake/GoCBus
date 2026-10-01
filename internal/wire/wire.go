// Package wire contains protocol-independent C-Bus wire primitives.
package wire

import "fmt"

// Framing constants are from the C-Bus Serial Interface User Guide and match
// the pinned libcbus cbus/common.py values.
const (
	EndCommand  = "\r"
	EndResponse = "\r\n"

	MinMessageSize = 2
	MaxBufferSize  = 256

	MinGroupAddress = 0
	MaxGroupAddress = 255
)

type ApplicationID uint8
type GroupAddress uint8

const ApplicationLighting ApplicationID = 0x38

func NewApplicationID(value int) (ApplicationID, error) {
	if value < 0 || value > 255 {
		return 0, fmt.Errorf("application ID out of range (0..255): %d", value)
	}
	return ApplicationID(value), nil
}

func NewGroupAddress(value int) (GroupAddress, error) {
	if value < MinGroupAddress || value > MaxGroupAddress {
		return 0, fmt.Errorf("group address out of range (%d..%d): %d", MinGroupAddress, MaxGroupAddress, value)
	}
	return GroupAddress(value), nil
}

// Checksum returns the two's-complement checksum used by C-Bus commands.
// Behaviour is derived from Michael Farrell's libcbus cbus/common.py at
// cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1 (LGPL-3.0-or-later).
func Checksum(data []byte) byte {
	var sum byte
	for _, value := range data {
		sum += value
	}
	return -sum
}

func AppendChecksum(data []byte) []byte {
	result := make([]byte, len(data), len(data)+1)
	copy(result, data)
	return append(result, Checksum(data))
}

func ValidateChecksum(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	return Checksum(data[:len(data)-1]) == data[len(data)-1]
}
