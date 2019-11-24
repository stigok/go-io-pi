package iopi

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type Port uint8
type Mode byte
type PinPolarity byte
type PinState uint8

type I2CDevice struct {
	Address byte   // I2C device address
	Path    string // e.g. /dev/i2c-1
	bus     *os.File
}

const (
	// As defined in the C implementation
	IODIRA = 0x00
	IODIRB = 0x01
	IPOLA  = 0x02
	IPOLB  = 0x03
	IOCON  = 0x0A
	GPPUA  = 0x0C
	GPPUB  = 0x0D
	GPIOA  = 0x12
	GPIOB  = 0x13

	// As defined in /usr/include/linux/i2c-dev.h
	I2C_SLAVE = 0x0703
)

const (
	// A single bus is split into two ports: pins 1-8 and 9-16
	PortA Port = iota
	PortB
)

const (
	PinPolarityNormal PinPolarity = iota
	PinPolarityInverted
)

const (
	Output Mode = iota
	Input
)

const (
	StateLow PinState = iota
	StateHigh
)

// Create a new device object.
// `bus` can be a string path to a file, or an os.File pointer to let multiple
// devices share the same file descriptor.
//
// TODO: It is not yet safe to share device file descriptors in a multi-threaded
// environment.
func NewI2CDevice(bus interface{}, addr byte) *I2CDevice {
	dev := I2CDevice{}
	dev.Address = addr

	// Accepting either a string or an *os.File makes us able to create multiple
	// devices with different addresses sharing the same file descriptor.
	switch b := bus.(type) {
	case string:
		dev.Path = b
		dev.bus = nil
	case *os.File:
		dev.Path = b.Name()
		dev.bus = b
	}

	return &dev
}

// Initialise device. This must be called once per device. You are expected
// to call `.Close()` to clean up resources when you're done.
func (dev *I2CDevice) Init() error {
	// If device object was initialised with a string path, open the file.
	if dev.bus == nil {
		file, err := os.OpenFile(dev.Path, os.O_RDWR, os.ModeCharDevice)
		if err != nil {
			return fmt.Errorf("failed to open i2c device at '%s': %s", dev.Path, err)
		}
		dev.bus = file
	}

	// Initialise the I2C bus
	err := unix.IoctlSetInt(int(dev.bus.Fd()), I2C_SLAVE, int(dev.Address))
	if err != nil {
		return fmt.Errorf("failed to write to i2c device at address '%02b': %s",
			dev.Address, err)
	}

	dev.driverInit()

	return nil
}

func (dev *I2CDevice) driverInit() {
	// Board initialisation
	// TODO: Handle errors
	dev.WriteByteData(IOCON, 0x22) // MCP23017 specific
	dev.SetPortDirection(PortA, Input)
	dev.SetPortDirection(PortB, Input)
	dev.SetPortPullups(PortA, 0x00)
	dev.SetPortPullups(PortB, 0x00)
	dev.SetPortPolarity(PortA, PinPolarityNormal)
	dev.SetPortPolarity(PortB, PinPolarityNormal)
}

// Clean up resources.
func (dev *I2CDevice) Close() error {
	return dev.bus.Close()
}

// Read raw data from a register.
// This is a low-level interface. You probably want to use the higher level
// functions to manipulate the board.
func (dev *I2CDevice) ReadByteData(reg byte) (byte, error) {
	buf := make([]byte, 1)
	buf[0] = reg

	n, err := dev.bus.Write(buf)
	if err != nil {
		return 0x0, fmt.Errorf("failed to write to slave before read (wrote %v bytes): %s\n", n, err)
	}

	n, err = dev.bus.Read(buf)
	if err != nil {
		return 0x0, fmt.Errorf("failed to read from slave: %s\n", err)
	}
	//fmt.Printf("read 0x%X (%v bytes) <- 0x%X\n", buf, n, reg)

	return buf[0], nil
}

// Write raw data to a register.
// This is a low-level interface. You probably want to use the higher level
// functions to manipulate the board.
func (dev *I2CDevice) WriteByteData(reg byte, value byte) error {
	buf := []byte{reg, value}

	//fmt.Printf("write 0x%08b to addr 0x%08b\n", value, reg)
	n, err := dev.bus.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write to slave (wrote %v bytes): %s\n", n, err)
	}

	return nil
}

// Collectively enable 100K pull-up resistors on all pins on a port.
func (dev *I2CDevice) SetPortPullups(port Port, state byte) error {
	switch port {
	case PortA:
		return dev.WriteByteData(GPPUA, state)
	case PortB:
		return dev.WriteByteData(GPPUB, state)
	default:
		return fmt.Errorf("invalid port: %v\n", port)
	}
}

// Enable 100K pull-up resistor on a single pin
func (dev *I2CDevice) SetPinPullup(pin uint8, state byte) error {
	pin, port := translatePin(pin)

	var reg byte
	if port == PortA {
		reg = GPPUA
	} else {
		reg = GPPUB
	}

	state, err := dev.ReadByteData(reg)
	if err != nil {
		return fmt.Errorf("failed to set pin pullup: %s", err)
	}

	return dev.WriteByteData(reg, setBit(state, pin, int(state)))
}

// Collectively set the polarity of all pins on a port.
// Also known as normal and inverted logic.
func (dev *I2CDevice) SetPortPolarity(port Port, pol PinPolarity) error {
	switch port {
	case PortA:
		return dev.WriteByteData(IPOLA, byte(pol))
	case PortB:
		return dev.WriteByteData(IPOLB, byte(pol))
	default:
		return fmt.Errorf("invalid port: %v\n", port)
	}
}

// Set polarity of a single pin
func (dev *I2CDevice) SetPinPolarity(pin uint8, pol PinPolarity) error {
	pin, port := translatePin(pin)

	var reg byte
	if port == PortA {
		reg = IPOLA
	} else {
		reg = IPOLB
	}

	state, err := dev.ReadByteData(reg)
	if err != nil {
		return fmt.Errorf("failed to set pin polarity: %s", err)
	}

	return dev.WriteByteData(reg, setBit(state, pin, int(pol)))
}

// Collectively set all pins on a port to specific mode.
func (dev *I2CDevice) SetPortDirection(port Port, mode Mode) error {
	switch port {
	case PortA:
		return dev.WriteByteData(IODIRA, byte(mode))
	case PortB:
		return dev.WriteByteData(IODIRB, byte(mode))
	default:
		return fmt.Errorf("invalid port: %v\n", port)
	}
}

// Set direction of a single pin
func (dev *I2CDevice) SetPinDirection(pin uint8, mode Mode) error {
	pin, port := translatePin(pin)

	var reg byte
	if port == PortA {
		reg = IODIRA
	} else {
		reg = IODIRB
	}

	state, err := dev.ReadByteData(reg)
	if err != nil {
		return fmt.Errorf("failed to set pin direction: %s", err)
	}

	return dev.WriteByteData(reg, setBit(state, pin, int(mode)))
}

// Collectively set all pins on the port to a specific state.
func (dev *I2CDevice) WritePort(port Port, state byte) error {
	switch port {
	case PortA:
		return dev.WriteByteData(GPIOA, state)
	case PortB:
		return dev.WriteByteData(GPIOB, state)
	default:
		return fmt.Errorf("invalid port: %v\n", port)
	}
}

// Return a byte describing the state of all pins on the selected port.
// Returns a zero byte if error != nil.
func (dev *I2CDevice) ReadPort(port Port) (byte, error) {
	switch port {
	case PortA:
		return dev.ReadByteData(GPIOA)
	case PortB:
		return dev.ReadByteData(GPIOB)
	default:
		return 0x00, fmt.Errorf("invalid port: %v\n", port)
	}
}

// Set single pin to a specific state.
func (dev *I2CDevice) WritePin(pin uint8, state PinState) error {
	pin, port := translatePin(pin)
	portState, err := dev.ReadPort(port)
	if err != nil {
		return fmt.Errorf("failed to write to pin %v: %s\n", pin, err)
	}

	newState := setBit(portState, pin, int(state))
	return dev.WritePort(port, newState)
}

// Translate a pin number 1-16 into 0-index pin on a specific port.
func translatePin(pin uint8) (uint8, Port) {
	if pin > 8 {
		return pin - 1 - 8, PortB
	} else {
		return pin - 1, PortA
	}
}

// Return the state of a single pin.
func (dev *I2CDevice) ReadPin(pin uint8) (PinState, error) {
	pin, port := translatePin(pin)
	portState, err := dev.ReadPort(port)
	return PinState(getBit(portState, pin)), err
}

// Set a single bit in a byte. All values except 0 is considered 1.
func setBit(byt byte, bit uint8, value int) byte {
	if value == 0 {
		return (byt &^ (1 << bit)) // clear bit
	} else {
		return (byt | (1 << bit)) // set bit
	}
}

// Get a single bit in a byte.
func getBit(byt byte, bit uint8) uint8 {
	if byt&(1<<bit) > 0 {
		return 1
	} else {
		return 0
	}
}
