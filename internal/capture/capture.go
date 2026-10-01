// Package capture reads and writes protocol-agnostic, timestamped traffic.
package capture

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

type Direction string

const (
	RX Direction = "rx"
	TX Direction = "tx"
)

type Record struct {
	Timestamp time.Time
	Direction Direction
	Data      []byte
}

type encodedRecord struct {
	Timestamp string    `json:"timestamp"`
	Direction Direction `json:"direction"`
	Data      string    `json:"data"`
}

func (record Record) validate() error {
	if record.Timestamp.IsZero() {
		return errors.New("capture timestamp is required")
	}
	if record.Direction != RX && record.Direction != TX {
		return fmt.Errorf("invalid capture direction %q", record.Direction)
	}
	if len(record.Data) == 0 {
		return errors.New("capture data is empty")
	}
	return nil
}

type Writer struct {
	encoder *json.Encoder
	mu      sync.Mutex
}

func NewWriter(output io.Writer) *Writer {
	return &Writer{encoder: json.NewEncoder(output)}
}

func (writer *Writer) Write(record Record) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if err := record.validate(); err != nil {
		return err
	}
	encoded := encodedRecord{
		Timestamp: record.Timestamp.UTC().Format(time.RFC3339Nano),
		Direction: record.Direction,
		Data:      strings.ToUpper(hex.EncodeToString(record.Data)),
	}
	if err := writer.encoder.Encode(encoded); err != nil {
		return fmt.Errorf("write capture record: %w", err)
	}
	return nil
}

type Reader struct {
	scanner *bufio.Scanner
	line    int
}

func NewReader(input io.Reader) *Reader {
	return &Reader{scanner: bufio.NewScanner(input)}
}

func (reader *Reader) Next() (Record, error) {
	if !reader.scanner.Scan() {
		if err := reader.scanner.Err(); err != nil {
			return Record{}, fmt.Errorf("read capture: %w", err)
		}
		return Record{}, io.EOF
	}
	reader.line++

	decoder := json.NewDecoder(strings.NewReader(reader.scanner.Text()))
	decoder.DisallowUnknownFields()
	var encoded encodedRecord
	if err := decoder.Decode(&encoded); err != nil {
		return Record{}, fmt.Errorf("capture line %d: decode: %w", reader.line, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Record{}, fmt.Errorf("capture line %d: expected one JSON object", reader.line)
		}
		return Record{}, fmt.Errorf("capture line %d: decode: %w", reader.line, err)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, encoded.Timestamp)
	if err != nil {
		return Record{}, fmt.Errorf("capture line %d: invalid timestamp: %w", reader.line, err)
	}
	data, err := hex.DecodeString(encoded.Data)
	if err != nil {
		return Record{}, fmt.Errorf("capture line %d: invalid data: %w", reader.line, err)
	}
	record := Record{Timestamp: timestamp, Direction: encoded.Direction, Data: data}
	if err := record.validate(); err != nil {
		return Record{}, fmt.Errorf("capture line %d: %w", reader.line, err)
	}
	return record, nil
}
