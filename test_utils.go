package iopi

import "fmt"

// Used as a spy when testing
type Call struct {
	Fn string
	Arg []byte
}

type FakeFile struct {
	Buf []byte
	CallHistory []Call
}

func NewFakeFile() *FakeFile {
	return &FakeFile{
		Buf: make([]byte, 2),
	}
}

func (c Call) String() string {
	return fmt.Sprintf("%s: 0x%02x", c.Fn, c.Arg)
}

// Records a call to the file API
func (f *FakeFile) recordCall(fn string, arg []byte) {
	call := Call{ fn, arg }
	f.CallHistory = append(f.CallHistory, call)
}

// Determine if a call with the specified signature has been made to this File
// instance.
func (f *FakeFile) HasCall(fn string, arg []byte) bool {
	for _, call := range f.CallHistory {
		if fmt.Sprintf("%s", call) == fmt.Sprintf("%s", Call{ fn, arg}) {
			return true
		}
	}
	return false
}

func (f *FakeFile) Read(b []byte) (int, error) {
	f.recordCall("Read", b)

	n := copy(b, f.Buf)
	return n, nil
}

func (f *FakeFile) Write(b []byte) (int, error) {
	f.recordCall("Write", b)

	//fmt.Printf("write befor: %b\n", b)
	n := copy(f.Buf, b)
	//fmt.Printf("write after: %b\n", f.Buf)
	return n, nil
}

func (f FakeFile) Close() error {
	return nil
}

func (f FakeFile) Fd() uintptr {
	return 0
}

func (f FakeFile) Name() string {
	return "fake"
}

func (f *FakeFile) Reset() {
	for i := range f.Buf {
		f.Buf[i] = 0
	}
}
