package wire

import "fmt"

// Direction identifies which side produced bytes presented to a Framer.
type Direction uint8

const (
	FromPCI Direction = iota + 1
	ToPCI
)

// EventKind identifies one complete wire-level item.
type EventKind uint8

const (
	Response EventKind = iota + 1
	Command
	PowerUp
	PCIError
	Confirmation
	Incomplete
	Overflow
)

// Event is one framed message, special response, or framing diagnostic. Data
// contains the exact wire bytes, including terminators. Overflow events have
// no Data and report the number of dropped buffered bytes in Dropped.
type Event struct {
	Kind    EventKind
	Data    []byte
	Dropped int
}

// Framer incrementally separates a C-Bus serial-interface byte stream. Its
// behaviour is derived from Michael Farrell's libcbus buffered_protocol.py and
// packet.py at cc0bdf3a25bd5646dd2d8e7d88a46fcd198f53a1
// (LGPL-3.0-or-later).
type Framer struct {
	direction    Direction
	buffer       []byte
	discarded    int
	discardSawCR bool
}

// NewFramer creates a bounded framer for bytes received from or sent to a PCI.
func NewFramer(direction Direction) (*Framer, error) {
	if direction != FromPCI && direction != ToPCI {
		return nil, fmt.Errorf("invalid wire direction: %d", direction)
	}
	return &Framer{
		direction: direction,
		buffer:    make([]byte, 0, MaxBufferSize),
	}, nil
}

// Feed adds arbitrary stream bytes and returns every complete event they make
// available. Returned Data does not alias either input or the internal buffer.
func (framer *Framer) Feed(data []byte) []Event {
	var events []Event
	for _, value := range data {
		if framer.discarded != 0 {
			if event, complete := framer.discard(value); complete {
				events = append(events, event)
			}
			continue
		}

		if len(framer.buffer) == MaxBufferSize {
			framer.discarded = len(framer.buffer)
			framer.discardSawCR = framer.direction == FromPCI && framer.buffer[len(framer.buffer)-1] == EndResponse[0]
			framer.buffer = framer.buffer[:0]
			if event, complete := framer.discard(value); complete {
				events = append(events, event)
			}
			continue
		}

		framer.buffer = append(framer.buffer, value)
		for {
			kind, size, complete := framer.next()
			if !complete {
				break
			}
			eventData := append([]byte(nil), framer.buffer[:size]...)
			events = append(events, Event{Kind: kind, Data: eventData})
			copy(framer.buffer, framer.buffer[size:])
			framer.buffer = framer.buffer[:len(framer.buffer)-size]
		}
	}
	return events
}

// Flush reports and clears bytes left without a complete terminator or special
// response. Callers use this when a stream ends so partial input is observable
// and cannot leak into a later connection.
func (framer *Framer) Flush() []Event {
	if framer.discarded != 0 {
		event := Event{Kind: Overflow, Dropped: framer.discarded}
		framer.discarded = 0
		framer.discardSawCR = false
		return []Event{event}
	}
	if len(framer.buffer) == 0 {
		return nil
	}
	data := append([]byte(nil), framer.buffer...)
	framer.buffer = framer.buffer[:0]
	return []Event{{Kind: Incomplete, Data: data}}
}

func (framer *Framer) discard(value byte) (Event, bool) {
	framer.discarded++
	complete := value == EndCommand[0]
	if framer.direction == FromPCI {
		complete = framer.discardSawCR && value == EndResponse[1]
		framer.discardSawCR = value == EndResponse[0]
	}
	if !complete {
		return Event{}, false
	}

	event := Event{Kind: Overflow, Dropped: framer.discarded}
	framer.discarded = 0
	framer.discardSawCR = false
	return event, true
}

func (framer *Framer) next() (EventKind, int, bool) {
	if len(framer.buffer) == 0 {
		return 0, 0, false
	}

	if framer.direction == ToPCI {
		if framer.buffer[len(framer.buffer)-1] == EndCommand[0] {
			return Command, len(framer.buffer), true
		}
		return 0, 0, false
	}

	switch framer.buffer[0] {
	case '+':
		return PowerUp, 1, true
	case '!':
		return PCIError, 1, true
	}

	if isConfirmationCode(framer.buffer[0]) {
		if len(framer.buffer) < MinMessageSize {
			return 0, 0, false
		}
		return Confirmation, MinMessageSize, true
	}

	if len(framer.buffer) >= len(EndResponse) &&
		framer.buffer[len(framer.buffer)-2] == EndResponse[0] &&
		framer.buffer[len(framer.buffer)-1] == EndResponse[1] {
		return Response, len(framer.buffer), true
	}
	return 0, 0, false
}

func isConfirmationCode(value byte) bool {
	return value >= 'g' && value <= 'z'
}
