package main

import (
	"fmt"
	"os"
	"time"

	"github.com/stigok/go-io-pi"
)

func main() {
	path := "/dev/i2c-1"
	file, err := os.OpenFile(path, os.O_RDWR, os.ModeCharDevice)

	if err != nil {
		panic(err)
	}

	dev := iopi.NewDevice(file, 0x20) // Bus1: 0x20, Bus2: 0x21
	err = dev.Init()
	if err != nil {
		panic(err)
	}
	defer dev.Close()

	dev.SetPortMode(iopi.PortA, iopi.Output)
	dev.SetPortMode(iopi.PortB, iopi.Output)

	pins := []uint8{ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16 }

	fmt.Println("Enabling pins:", pins)
	for _, p := range(pins) {
		dev.WritePin(p, iopi.High)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("Disabling pins:", pins)
	for _, p := range(pins) {
		dev.WritePin(p, iopi.Low)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("Exiting!")
}
