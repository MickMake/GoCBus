package capture

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWriteAndReplay(t *testing.T) {
	want := []Record{
		{Timestamp: time.Date(2026, 10, 1, 1, 2, 3, 4, time.FixedZone("AEST", 10*60*60)), Direction: RX, Data: []byte{0x05, 0x38, 0x00}},
		{Timestamp: time.Date(2026, 10, 1, 1, 2, 4, 5, time.UTC), Direction: TX, Data: []byte{'\\', 'A', '3', '2', '1', '\r'}},
	}

	var buffer bytes.Buffer
	writer := NewWriter(&buffer)
	for _, record := range want {
		if err := writer.Write(record); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	reader := NewReader(&buffer)
	for index, expected := range want {
		got, err := reader.Next()
		if err != nil {
			t.Fatalf("Next() record %d error = %v", index, err)
		}
		if !got.Timestamp.Equal(expected.Timestamp) || got.Direction != expected.Direction || !bytes.Equal(got.Data, expected.Data) {
			t.Errorf("Next() record %d = %+v, want %+v", index, got, expected)
		}
	}
	if _, err := reader.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("Next() after records error = %v, want EOF", err)
	}
}

func TestReplayFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "capture", "libcbus-lighting.ndjson")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	reader := NewReader(file)
	first, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if first.Direction != RX || string(first.Data) != "0538000108BA\r\n" {
		t.Fatalf("first fixture record = %+v", first)
	}
	second, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if second.Direction != TX || string(second.Data) != "A3210038\r" {
		t.Fatalf("second fixture record = %+v", second)
	}
}

func TestRejectsInvalidRecords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"unknown direction", `{"timestamp":"2026-10-01T00:00:00Z","direction":"sideways","data":"01"}`, "invalid capture direction"},
		{"bad timestamp", `{"timestamp":"eventually","direction":"rx","data":"01"}`, "invalid timestamp"},
		{"bad hex", `{"timestamp":"2026-10-01T00:00:00Z","direction":"rx","data":"XX"}`, "invalid data"},
		{"empty data", `{"timestamp":"2026-10-01T00:00:00Z","direction":"rx","data":""}`, "capture data is empty"},
		{"unknown field", `{"timestamp":"2026-10-01T00:00:00Z","direction":"rx","data":"01","packet":1}`, "unknown field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReader(strings.NewReader(tt.input)).Next()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Next() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestWriterRejectsInvalidRecord(t *testing.T) {
	err := NewWriter(io.Discard).Write(Record{Direction: RX, Data: []byte{1}})
	if err == nil || !strings.Contains(err.Error(), "timestamp") {
		t.Fatalf("Write() error = %v, want timestamp error", err)
	}
}

func TestWriterSupportsConcurrentTransportTraffic(t *testing.T) {
	var output bytes.Buffer
	writer := NewWriter(&output)
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			err := writer.Write(Record{
				Timestamp: time.Date(2026, 10, 1, 0, 0, index, 0, time.UTC),
				Direction: RX,
				Data:      []byte{byte(index)},
			})
			if err != nil {
				t.Errorf("Write() error = %v", err)
			}
		}()
	}
	close(start)
	group.Wait()

	reader := NewReader(bytes.NewReader(output.Bytes()))
	count := 0
	for {
		_, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		count++
	}
	if count != 20 {
		t.Fatalf("record count = %d, want 20", count)
	}
}
