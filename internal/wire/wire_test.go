package wire

import "testing"

func TestAppendChecksumLibCBusExamples(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want []byte
	}{
		{"empty", nil, []byte{0x00}},
		{"zero", []byte{0x00}, []byte{0x00, 0x00}},
		{"one", []byte{0x01}, []byte{0x01, 0xff}},
		{"largest byte", []byte{0xff}, []byte{0xff, 0x01}},
		{"lighting off", []byte{0x05, 0x38, 0x00, 0x01, 0x08}, []byte{0x05, 0x38, 0x00, 0x01, 0x08, 0xba}},
		{"enable", []byte{0x05, 0xcb, 0x00, 0x02, 0x37, 0x82}, []byte{0x05, 0xcb, 0x00, 0x02, 0x37, 0x82, 0x75}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendChecksum(tt.data)
			if string(got) != string(tt.want) {
				t.Fatalf("AppendChecksum(% X) = % X, want % X", tt.data, got, tt.want)
			}
			if !ValidateChecksum(got) {
				t.Fatalf("ValidateChecksum(% X) = false", got)
			}
		})
	}
}

func TestAppendChecksumEveryByteMatchesLibCBus(t *testing.T) {
	// This is the exhaustive one-byte compatibility case from pinned libcbus
	// tests/test_common.py.
	for value := 0; value <= 255; value++ {
		got := AppendChecksum([]byte{byte(value)})
		wantChecksum := byte((-value) & 0xff)
		if len(got) != 2 || got[0] != byte(value) || got[1] != wantChecksum {
			t.Fatalf("AppendChecksum(%02X) = % X, want %02X %02X", value, got, value, wantChecksum)
		}
	}
}

func TestValidateChecksumRejectsInvalid(t *testing.T) {
	for _, data := range [][]byte{nil, {0x01}, {0x05, 0x38, 0x00, 0x01, 0x08, 0xbb}} {
		if ValidateChecksum(data) {
			t.Errorf("ValidateChecksum(% X) = true", data)
		}
	}
}

func TestAddressValidation(t *testing.T) {
	tests := []struct {
		value int
		ok    bool
	}{
		{-1, false},
		{0, true},
		{255, true},
		{256, false},
	}
	for _, tt := range tests {
		if _, err := NewGroupAddress(tt.value); (err == nil) != tt.ok {
			t.Errorf("NewGroupAddress(%d) error = %v, want ok %v", tt.value, err, tt.ok)
		}
		if _, err := NewApplicationID(tt.value); (err == nil) != tt.ok {
			t.Errorf("NewApplicationID(%d) error = %v, want ok %v", tt.value, err, tt.ok)
		}
	}
}
