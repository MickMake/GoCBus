// Package transport provides protocol-agnostic byte-stream connections to a
// C-Bus PCI over serial or a CNI over TCP.
package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"go.bug.st/serial"

	"github.com/MickMake/GoCBus/internal/capture"
)

const PCIBaudRate = 9600

// Dialer opens one raw C-Bus byte stream. Calling Dial again after closing a
// connection is the reconnect hook; retry policy belongs to a later slice.
type Dialer interface {
	Dial(context.Context) (io.ReadWriteCloser, error)
}

type serialOpener func(string, *serial.Mode) (serial.Port, error)

// SerialDialer opens a physical C-Bus PCI using the serial settings used by
// the pinned libcbus reference: 9600 baud, eight data bits, no parity and one
// stop bit.
type SerialDialer struct {
	Device string
	open   serialOpener
}

func NewSerialDialer(device string) *SerialDialer {
	return &SerialDialer{Device: device, open: serial.Open}
}

func (dialer *SerialDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("open serial PCI: %w", err)
	}
	if strings.TrimSpace(dialer.Device) == "" {
		return nil, errors.New("open serial PCI: device is required")
	}

	open := dialer.open
	if open == nil {
		open = serial.Open
	}
	port, err := open(dialer.Device, &serial.Mode{
		BaudRate: PCIBaudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	})
	if err != nil {
		return nil, fmt.Errorf("open serial PCI %q: %w", dialer.Device, err)
	}
	if err := ctx.Err(); err != nil {
		_ = port.Close()
		return nil, fmt.Errorf("open serial PCI: %w", err)
	}
	return port, nil
}

// TCPDialer opens a TCP connection to a C-Bus CNI.
type TCPDialer struct {
	Address string
	Dialer  *net.Dialer
}

func NewTCPDialer(address string) *TCPDialer {
	return &TCPDialer{Address: address}
}

func (dialer *TCPDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	if strings.TrimSpace(dialer.Address) == "" {
		return nil, errors.New("open TCP CNI: address is required")
	}
	host, port, err := net.SplitHostPort(dialer.Address)
	if err != nil {
		return nil, fmt.Errorf("open TCP CNI: invalid address %q: %w", dialer.Address, err)
	}
	portNumber, err := strconv.Atoi(port)
	if strings.TrimSpace(host) == "" || err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("open TCP CNI: invalid address %q: require host and numeric port 1-65535", dialer.Address)
	}

	networkDialer := dialer.Dialer
	if networkDialer == nil {
		networkDialer = &net.Dialer{}
	}
	connection, err := networkDialer.DialContext(ctx, "tcp", dialer.Address)
	if err != nil {
		return nil, fmt.Errorf("open TCP CNI %q: %w", dialer.Address, err)
	}
	return connection, nil
}

// Observer receives the bytes actually read or written by a connection.
// Returning an error makes the current I/O operation fail after reporting its
// completed byte count, so callers do not unknowingly lose capture failures.
type Observer func(capture.Direction, []byte) error

// Observe wraps a connection with protocol-agnostic RX/TX observation.
func Observe(connection io.ReadWriteCloser, observers ...Observer) io.ReadWriteCloser {
	return &observedConnection{ReadWriteCloser: connection, observers: observers}
}

type observedConnection struct {
	io.ReadWriteCloser
	observers []Observer
}

func (connection *observedConnection) Read(buffer []byte) (int, error) {
	count, readErr := connection.ReadWriteCloser.Read(buffer)
	return count, errors.Join(readErr, connection.observe(capture.RX, buffer[:count]))
}

func (connection *observedConnection) Write(data []byte) (int, error) {
	count, writeErr := connection.ReadWriteCloser.Write(data)
	return count, errors.Join(writeErr, connection.observe(capture.TX, data[:count]))
}

func (connection *observedConnection) observe(direction capture.Direction, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	stable := append([]byte(nil), data...)
	var result error
	for _, observer := range connection.observers {
		if observer == nil {
			continue
		}
		if err := observer(direction, stable); err != nil {
			result = errors.Join(result, fmt.Errorf("observe %s traffic: %w", direction, err))
		}
	}
	return result
}
