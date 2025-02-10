package pgtype

import (
	"strings"
	"testing"
)

func TestDecodeUUIDBinaryError(t *testing.T) {
	t.Parallel()
	_, err := decodeUUIDBinary([]byte{0x12, 0x34})

	if err == nil {
		t.Fatal("Expected error, got none")
	}
	if !strings.HasPrefix(err.Error(), "pgtype:") {
		t.Errorf("Expected error to start with %q, got %q", "pgtype:", err.Error())
	}
	if !strings.Contains(err.Error(), "bad length: 2") {
		t.Errorf("Expected error to contain length, got %q", err.Error())
	}
}

func BenchmarkDecodeUUIDBinary(b *testing.B) {
	x := []byte{0x03, 0xa3, 0x52, 0x2f, 0x89, 0x28, 0x49, 0x87, 0x84, 0xd6, 0x93, 0x7b, 0x36, 0xec, 0x27, 0x6f}
	for i := 0; i < b.N; i++ {
		_, _ = decodeUUIDBinary(x)
	}
}
