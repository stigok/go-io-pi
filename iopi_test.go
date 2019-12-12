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

func TestSetPortPullup(t *testing.T) {
	t.Run("port A", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.SetPortPullup(PortA, 0x55)
		if !file.HasCall("Write", []byte{ GPPUA, 0x55}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("port B", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.SetPortPullup(PortB, 0x55)
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

func TestSetPortPolarity(t *testing.T) {
	t.Run("port A", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.SetPortPolarity(PortA, 0x55)
		if !file.HasCall("Write", []byte{ IPOLA, 0x55}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("port B", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.SetPortPolarity(PortB, PolarityInverted)
		if !file.HasCall("Write", []byte{ IPOLB, 0xFF}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}

func TestSetPinPolarity(t *testing.T) {
	t.Run("pin number < 8", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		file.NextRead = []byte{ 0x00, 0x00 }
		dev.SetPinPolarity(7, 1)

		if !file.HasCall("Write", []byte{ IPOLA, 0b01000000 }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("pin number > 8", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		file.NextRead = []byte{ 0x00, 0x00 }
		dev.SetPinPolarity(16, 1)

		if !file.HasCall("Write", []byte{ IPOLB, 0b10000000 }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}

func TestSetPortDirection(t *testing.T) {
	t.Run("port A", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.SetPortDirection(PortA, 0x55)
		if !file.HasCall("Write", []byte{ IODIRA, 0x55}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("port B", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.SetPortDirection(PortB, 0x55)
		if !file.HasCall("Write", []byte{ IODIRB, 0x55}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}

func TestSetPinDirection(t *testing.T) {
	t.Run("pin number < 8", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		file.NextRead = []byte{ 0x00, 0x00 }
		dev.SetPinDirection(7, 1)

		if !file.HasCall("Write", []byte{ IODIRA, 0b01000000 }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("pin number > 8", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		file.NextRead = []byte{ 0x00, 0x00 }
		dev.SetPinDirection(16, 1)

		if !file.HasCall("Write", []byte{ IODIRB, 0b10000000 }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}

func TestWritePort(t *testing.T) {
	t.Run("port A", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.WritePort(PortA, 0xFF)

		if !file.HasCall("Write", []byte{ GPIOA, 0xFF}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("port B", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.WritePort(PortB, 0xFF)

		if !file.HasCall("Write", []byte{ GPIOB, 0xFF}) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}

func TestReadPort(t *testing.T) {
	t.Run("port A", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.ReadPort(PortA)

		if !file.HasCall("Read", []byte{ GPIOA }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("port B", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		dev.ReadPort(PortB)

		if !file.HasCall("Write", []byte{ GPIOB }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}

func TestWritePin(t *testing.T) {
	t.Run("pin <= 8", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		file.NextRead = []byte{ 0x00, 0x00 }
		dev.WritePin(7, High)

		if !file.HasCall("Write", []byte{ GPIOA, 0b01000000 }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})

	t.Run("pin > 8", func (t *testing.T) {
		file := NewFakeFile()
		dev := NewDevice(file, 0x20)

		file.NextRead = []byte{ 0x00, 0x00 }
		dev.WritePin(15, High)

		if !file.HasCall("Write", []byte{ GPIOB, 0b01000000 }) {
			t.Error("did not write expected data", file.CallHistory)
		}
	})
}

func TestTranslatePin(t *testing.T) {
	t.Run("pin <= 8", func (t *testing.T) {
		pin, port := translatePin(7)
		if pin != 7 - 1 {
			t.Error("pin number off by one")
		}
		if port != PortA {
			t.Error("wrong port returned")
		}
	})

	t.Run("pin > 8", func (t *testing.T) {
		pin, port := translatePin(16)
		if pin != 8 - 1 {
			t.Error("pin number off by one")
		}
		if port != PortB {
			t.Error("wrong port returned")
		}
	})
}


func TestReadPin(t *testing.T) {
	// I don't know how to test this yet
	t.Run("pin <= 8", func (t *testing.T) {
		t.Skip()
	})

	t.Run("pin > 8", func (t *testing.T) {
		t.Skip()
	})
}

func TestSetBit(t *testing.T) {
	var b byte = 0b00000000
	b = setBit(b, 3, 1)

	if b != 0b00001000 {
		t.Error("expected bit was not set")
	}
}

func TestGetBit(t *testing.T) {
	var b byte = 0b01000000
	bit := getBit(b, 6)
	if bit != 1 {
		t.Error("expected bit was not set")
	}
}
