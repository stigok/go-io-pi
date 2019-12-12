package iopi

import (
	"reflect"
	"testing"
)

func TestWrite(t *testing.T) {
	file := NewFakeFile()
	dev := NewDevice(file, 0x20)

	file.Read([]byte{0x12})

	t.Run("writes a byte to specified register", func(t *testing.T) {
		err := dev.WriteByteData(0x42, 0x11)
		if err != nil {
			t.Error("failed to write")
		}

		if !reflect.DeepEqual(file.Buf, []byte{0x42, 0x11}) {
			t.Error("did not write expected data")
		}
	})
}

func TestRead(t *testing.T) {
	file := NewFakeFile()
	dev := NewDevice(file, 0x20)

	// This bears the same meaning as "reads from specified register"
	t.Run("write register addr before read", func(t *testing.T) {
		reg := byte(0x42)
		_, err := dev.ReadByteData(reg)
		if err != nil {
			t.Error("failed to read")
		}

		if !reflect.DeepEqual(file.Buf, []byte{0x42, 0x00}) {
			t.Error("did not write expected register before read")
		}
	})
}
