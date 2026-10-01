package wire

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MickMake/GoCBus/internal/capture"
)

func TestFramerKnownItemsAtEverySplit(t *testing.T) {
	tests := []struct {
		name      string
		direction Direction
		data      []byte
		want      []Event
	}{
		{
			name:      "PCI response",
			direction: FromPCI,
			data:      []byte("0538000108BA\r\n"),
			want:      []Event{{Kind: Response, Data: []byte("0538000108BA\r\n")}},
		},
		{
			name:      "command",
			direction: ToPCI,
			data:      []byte("A3210038\r"),
			want:      []Event{{Kind: Command, Data: []byte("A3210038\r")}},
		},
		{
			name:      "repeated power-up notification",
			direction: FromPCI,
			data:      []byte("++"),
			want: []Event{
				{Kind: PowerUp, Data: []byte("+")},
				{Kind: PowerUp, Data: []byte("+")},
			},
		},
		{
			name:      "PCI error",
			direction: FromPCI,
			data:      []byte("!"),
			want:      []Event{{Kind: PCIError, Data: []byte("!")}},
		},
		{
			name:      "successful confirmation",
			direction: FromPCI,
			data:      []byte("g."),
			want:      []Event{{Kind: Confirmation, Data: []byte("g.")}},
		},
		{
			name:      "failed confirmation",
			direction: FromPCI,
			data:      []byte("z#"),
			want:      []Event{{Kind: Confirmation, Data: []byte("z#")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for split := 0; split <= len(tt.data); split++ {
				framer := newTestFramer(t, tt.direction)
				got := framer.Feed(tt.data[:split])
				got = append(got, framer.Feed(tt.data[split:])...)
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("split %d: events = %#v, want %#v", split, got, tt.want)
				}
			}
		})
	}
}

func TestFramerEmitsMultipleItemsFromOneRead(t *testing.T) {
	data := []byte("0538\r\n++!g.h#0506\r\n")
	want := []Event{
		{Kind: Response, Data: []byte("0538\r\n")},
		{Kind: PowerUp, Data: []byte("+")},
		{Kind: PowerUp, Data: []byte("+")},
		{Kind: PCIError, Data: []byte("!")},
		{Kind: Confirmation, Data: []byte("g.")},
		{Kind: Confirmation, Data: []byte("h#")},
		{Kind: Response, Data: []byte("0506\r\n")},
	}

	framer := newTestFramer(t, FromPCI)
	got := framer.Feed(data)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}

	framer = newTestFramer(t, FromPCI)
	got = nil
	for _, value := range data {
		got = append(got, framer.Feed([]byte{value})...)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("byte-at-a-time events = %#v, want %#v", got, want)
	}
}

func TestFramerEmitsMultipleCommandsFromOneRead(t *testing.T) {
	framer := newTestFramer(t, ToPCI)
	want := []Event{
		{Kind: Command, Data: []byte("A3210038\r")},
		{Kind: Command, Data: []byte("A3220038\r")},
	}
	if got := framer.Feed([]byte("A3210038\rA3220038\r")); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestFramerHandlesReadLargerThanBufferWhenItemsAreBounded(t *testing.T) {
	framer := newTestFramer(t, FromPCI)
	data := bytes.Repeat([]byte("A\r\n"), MaxBufferSize)
	got := framer.Feed(data)
	if len(got) != MaxBufferSize {
		t.Fatalf("event count = %d, want %d", len(got), MaxBufferSize)
	}
	for index, event := range got {
		if event.Kind != Response || !bytes.Equal(event.Data, []byte("A\r\n")) {
			t.Fatalf("event %d = %#v", index, event)
		}
	}
}

func TestFramerAcceptsMessageAtBufferLimit(t *testing.T) {
	tests := []struct {
		name      string
		direction Direction
		data      []byte
		kind      EventKind
	}{
		{
			name:      "PCI response",
			direction: FromPCI,
			data:      append(bytes.Repeat([]byte{'A'}, MaxBufferSize-len(EndResponse)), EndResponse...),
			kind:      Response,
		},
		{
			name:      "command",
			direction: ToPCI,
			data:      append(bytes.Repeat([]byte{'A'}, MaxBufferSize-len(EndCommand)), EndCommand...),
			kind:      Command,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			framer := newTestFramer(t, tt.direction)
			got := framer.Feed(tt.data)
			want := []Event{{Kind: tt.kind, Data: tt.data}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("events = %#v, want %#v", got, want)
			}
		})
	}
}

func TestFramerReportsOverflowAndRecovers(t *testing.T) {
	tests := []struct {
		name      string
		direction Direction
		tail      []byte
		want      []Event
	}{
		{
			name:      "PCI response",
			direction: FromPCI,
			tail:      []byte("B\r\nC\r\n"),
			want: []Event{
				{Kind: Overflow, Dropped: MaxBufferSize + len("B\r\n")},
				{Kind: Response, Data: []byte("C\r\n")},
			},
		},
		{
			name:      "command",
			direction: ToPCI,
			tail:      []byte("B\rC\r"),
			want: []Event{
				{Kind: Overflow, Dropped: MaxBufferSize + len("B\r")},
				{Kind: Command, Data: []byte("C\r")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			framer := newTestFramer(t, tt.direction)
			if got := framer.Feed(bytes.Repeat([]byte{'A'}, MaxBufferSize)); got != nil {
				t.Fatalf("initial events = %#v, want nil", got)
			}
			if got := framer.Feed(tt.tail); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("events = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestFramerResynchronisesOverflowAtSpecialResponses(t *testing.T) {
	tests := []struct {
		name string
		tail []byte
		want []Event
	}{
		{
			name: "power-up",
			tail: []byte("+B\r\n"),
			want: []Event{
				{Kind: Overflow, Dropped: MaxBufferSize},
				{Kind: PowerUp, Data: []byte("+")},
				{Kind: Response, Data: []byte("B\r\n")},
			},
		},
		{
			name: "PCI error",
			tail: []byte("!B\r\n"),
			want: []Event{
				{Kind: Overflow, Dropped: MaxBufferSize},
				{Kind: PCIError, Data: []byte("!")},
				{Kind: Response, Data: []byte("B\r\n")},
			},
		},
		{
			name: "confirmation",
			tail: []byte("g.B\r\n"),
			want: []Event{
				{Kind: Overflow, Dropped: MaxBufferSize},
				{Kind: Confirmation, Data: []byte("g.")},
				{Kind: Response, Data: []byte("B\r\n")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for split := 0; split <= len(tt.tail); split++ {
				framer := newTestFramer(t, FromPCI)
				if got := framer.Feed(bytes.Repeat([]byte{'A'}, MaxBufferSize)); got != nil {
					t.Fatalf("initial events = %#v, want nil", got)
				}
				got := framer.Feed(tt.tail[:split])
				got = append(got, framer.Feed(tt.tail[split:])...)
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("split %d: events = %#v, want %#v", split, got, tt.want)
				}
			}
		})
	}
}

func TestFramerFlushReportsUnterminatedOverflow(t *testing.T) {
	framer := newTestFramer(t, FromPCI)
	framer.Feed(bytes.Repeat([]byte{'A'}, MaxBufferSize+1))
	want := []Event{{Kind: Overflow, Dropped: MaxBufferSize + 1}}
	if got := framer.Flush(); !reflect.DeepEqual(got, want) {
		t.Fatalf("flush events = %#v, want %#v", got, want)
	}
}

func TestFramerFlushReportsIncompleteInputAndRecovers(t *testing.T) {
	framer := newTestFramer(t, FromPCI)
	if got := framer.Feed([]byte("0538\r")); got != nil {
		t.Fatalf("initial events = %#v, want nil", got)
	}
	want := []Event{{Kind: Incomplete, Data: []byte("0538\r")}}
	if got := framer.Flush(); !reflect.DeepEqual(got, want) {
		t.Fatalf("flush events = %#v, want %#v", got, want)
	}
	if got := framer.Flush(); got != nil {
		t.Fatalf("second flush events = %#v, want nil", got)
	}

	want = []Event{{Kind: Response, Data: []byte("0506\r\n")}}
	if got := framer.Feed([]byte("0506\r\n")); !reflect.DeepEqual(got, want) {
		t.Fatalf("events after flush = %#v, want %#v", got, want)
	}
}

func TestFramerPreservesUnknownFramedInput(t *testing.T) {
	framer := newTestFramer(t, FromPCI)
	want := []Event{{Kind: Response, Data: []byte("NOT-HEX\r\n")}}
	if got := framer.Feed([]byte("NOT-HEX\r\n")); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestFramerMatchesLibCBusConfirmationFallback(t *testing.T) {
	// Pinned libcbus treats a confirmation as successful only when its second
	// byte is '.', and therefore exposes every other suffix as a failed
	// confirmation for the packet layer to interpret.
	framer := newTestFramer(t, FromPCI)
	want := []Event{{Kind: Confirmation, Data: []byte("g?")}}
	if got := framer.Feed([]byte("g?")); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestFramerReturnedDataIsStable(t *testing.T) {
	framer := newTestFramer(t, FromPCI)
	input := []byte("0538\r\n")
	events := framer.Feed(input)
	input[0] = 'X'
	framer.Feed([]byte("0506\r\n"))
	if got := string(events[0].Data); got != "0538\r\n" {
		t.Fatalf("first event changed to %q", got)
	}
}

func TestFramerRejectsInvalidDirection(t *testing.T) {
	if _, err := NewFramer(0); err == nil {
		t.Fatal("NewFramer(0) error = nil")
	}
}

func TestFramerReplaysCaptureFixture(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "testdata", "capture", "libcbus-lighting.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	framers := map[capture.Direction]*Framer{
		capture.RX: newTestFramer(t, FromPCI),
		capture.TX: newTestFramer(t, ToPCI),
	}
	reader := capture.NewReader(file)
	for {
		record, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		events := framers[record.Direction].Feed(record.Data)
		if len(events) != 1 || !bytes.Equal(events[0].Data, record.Data) {
			t.Fatalf("record %+v produced %#v", record, events)
		}
		wantKind := Response
		if record.Direction == capture.TX {
			wantKind = Command
		}
		if events[0].Kind != wantKind {
			t.Fatalf("record direction %q produced kind %d, want %d", record.Direction, events[0].Kind, wantKind)
		}
	}
}

func newTestFramer(t *testing.T, direction Direction) *Framer {
	t.Helper()
	framer, err := NewFramer(direction)
	if err != nil {
		t.Fatal(err)
	}
	return framer
}
