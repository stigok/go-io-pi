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

func TestInit(t *testing.T) {
	file := NewFakeFile()
	dev := NewDevice(file, 0x20)

	dev.driverInit()

	t.Run("performs mcp23017 chip init", func(t *testing.T) {
		if !file.HasCall("Write", []byte{ IOCON, 0x22 }) {
			t.Error("expected registers not written to")
		}
	})

	t.Run("port mode set to input", func(t *testing.T) {
		if !file.HasCall("Write", []byte{ IODIRA, 0xFF }) {
			t.Error("port A not configured")
		}
		if !file.HasCall("Write", []byte{ IODIRB, 0xFF }) {
			t.Error("port B not configured")
		}
	})

	t.Run("port pullup resistors disabled", func (t *testing.T) {
		if !file.HasCall("Write", []byte{ GPPUA, 0x00 }) {
			t.Error("port A not configured")
		}
		if !file.HasCall("Write", []byte{ GPPUB, 0x00 }) {
			t.Error("port B not configured")
		}
	})

	t.Run("port polarity inversion disabled", func (t *testing.T) {
		if !file.HasCall("Write", []byte{ IPOLA, 0x00 }) {
			t.Error("port A not configured")
		}
		if !file.HasCall("Write", []byte{ IPOLB, 0x00 }) {
			t.Error("port B not configured")
		}
	})
}

func TestClose(t *testing.T) {
	file := NewFakeFile()
	dev := NewDevice(file, 0x20)
	dev.Close()

	if !file.HasCall("Close", nil) {
		t.Error("file was not closed")
	}
}

func TestSetPortPullups(t *testing.T) {
	t.Run("port A", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.SetPortPullups(PortA, 0x55)
		if !file.HasCall("Write", []byte{ GPPUA, 0x55}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("port B", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.SetPortPullups(PortB, 0x55)
		if !file.HasCall("Write", []byte{ GPPUB, 0x55}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}

func TestSetPinPullup(t *testing.T) {
	t.Run("pin number < 8", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		file.NextRead = []byte{ 0x00, 0x00 }
		dev.SetPinPullup(7, 1)

		if !file.HasCall("Write", []byte{ GPPUA, 0b01000000 }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("pin number > 8", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		file.NextRead = []byte{ 0x00, 0x00 }
		dev.SetPinPullup(16, 1)

		if !file.HasCall("Write", []byte{ GPPUB, 0b10000000 }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}
