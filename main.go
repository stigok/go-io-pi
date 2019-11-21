package main

import "fmt"
import "os"
import "time"
import "golang.org/x/sys/unix"

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

	// A single bus is split into two ports: pins 1-8 and 9-16
	BoardPortA BoardPort = 0
	BoardPortB BoardPort = 1

	PinPolarityNormal   PinPolarity = 0x00
	PinPolarityInverted PinPolarity = 0xFF

	ModeOutput PinMode = 0x00
	ModeInput  PinMode = 0xFF

	StateLow  PinState = 0
	StateHigh PinState = 1
)

type BoardPort uint8
type PinMode byte
type PinPolarity byte
type PinState int

type I2CDevice struct {
	Address byte   // I2C device address
	Path    string // e.g. /dev/i2c-1
	bus     *os.File
}

// Create a new device object.
// `bus` can be a string path to a file, or an os.File pointer to let multiple
// devices share the same file descriptor.
//
// It is not safe to share device file descriptors in a multi-threaded
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

	// Board initialisation
	// TODO: Handle errors
	dev.WriteByteData(IOCON, 0x22) // MCP23017 specific
	dev.SetPortDirection(BoardPortA, ModeInput);
	dev.SetPortDirection(BoardPortB, ModeInput);
	dev.SetPortPullups(BoardPortA, 0x00)
	dev.SetPortPullups(BoardPortB, 0x00)
	dev.SetPortPolarity(BoardPortA, PinPolarityNormal)
	dev.SetPortPolarity(BoardPortB, PinPolarityNormal)

	return nil
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
func (dev *I2CDevice) SetPortPullups(port BoardPort, state byte) error {
	switch port {
	case BoardPortA:
		return dev.WriteByteData(GPPUA, state)
	case BoardPortB:
		return dev.WriteByteData(GPPUB, state)
	default:
		return fmt.Errorf("invalid port: %v\n", port)
	}
}

// Enable 100K pull-up resistor on a single pin
func (dev *I2CDevice) SetPinPullup(pin uint8, state byte) error {
	pin, port := translatePin(pin)

	var reg byte
	if port == BoardPortA {
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
func (dev *I2CDevice) SetPortPolarity(port BoardPort, pol PinPolarity) error {
	switch port {
	case BoardPortA:
		return dev.WriteByteData(IPOLA, byte(pol))
	case BoardPortB:
		return dev.WriteByteData(IPOLB, byte(pol))
	default:
		return fmt.Errorf("invalid port: %v\n", port)
	}
}

// Set polarity of a single pin
func (dev *I2CDevice) SetPinPolarity(pin uint8, pol PinPolarity) error {
	pin, port := translatePin(pin)

	var reg byte
	if port == BoardPortA {
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
func (dev *I2CDevice) SetPortDirection(port BoardPort, mode PinMode) error {
	switch port {
	case BoardPortA:
		return dev.WriteByteData(IODIRA, byte(mode))
	case BoardPortB:
		return dev.WriteByteData(IODIRB, byte(mode))
	default:
		return fmt.Errorf("invalid port: %v\n", port)
	}
}

// Set direction of a single pin
func (dev *I2CDevice) SetPinDirection(pin uint8, mode PinMode) error {
	pin, port := translatePin(pin)

	var reg byte
	if port == BoardPortA {
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
func (dev *I2CDevice) WritePort(port BoardPort, state byte) error {
	switch port {
	case BoardPortA:
		return dev.WriteByteData(GPIOA, state)
	case BoardPortB:
		return dev.WriteByteData(GPIOB, state)
	default:
		return fmt.Errorf("invalid port: %v\n", port)
	}
}

// Return a byte describing the state of all pins on the selected port.
// Returns a zero byte if error != nil.
func (dev *I2CDevice) ReadPort(port BoardPort) (byte, error) {
	switch port {
	case BoardPortA:
		return dev.ReadByteData(GPIOA)
	case BoardPortB:
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
func translatePin(pin uint8) (uint8, BoardPort) {
	if pin > 8 {
		return pin - 1 - 8, BoardPortB
	} else {
		return pin - 1, BoardPortA
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

func main() {
	path := "/dev/i2c-1"
	bus := byte(0x20) // Bus1: 0x20, Bus2: 0x21
	dev := NewI2CDevice(path, bus)
	err := dev.Init()
	if err != nil {
		panic(err)
	}
	defer dev.Close()

	// Set mode on all ports
	//dev.SetPortDirection(BoardPortA, ModeOutput)
	//dev.SetPortDirection(BoardPortB, ModeInput)

	// Set all outputs to LOW
	//dev.WritePort(BoardPortA, 0x00)
	//dev.WritePort(BoardPortB, 0x00)

	//defer dev.WritePort(BoardPortA, 0x00)
	//defer dev.WritePort(BoardPortB, 0x00)

	dev.SetPortDirection(BoardPortA, ModeInput)
	dev.SetPortDirection(BoardPortB, ModeInput)

	for true {
		//val, err := dev.ReadPin(3)
		a, err := dev.ReadByteData(GPIOA)
		if err != nil {
			panic(err)
		}

		b, err := dev.ReadByteData(GPIOB)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%08b %08b\n", a, b)
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("Done!")
}
