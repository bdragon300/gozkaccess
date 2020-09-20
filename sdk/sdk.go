package sdk

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

var ControlOperations = map[string]int{
	"output": 1,
	"cancel_alarm": 2,
	"restart": 3,
}

type ZKSDK struct {
	handle int
	dll syscall.Handle
}

func NewZKSDK(dllpath string) (*ZKSDK, error) {
	dllHandle, err := syscall.LoadLibrary(dllpath)
	if err != nil {
		return nil, err
	}

	return &ZKSDK{0, dllHandle}, nil
}

func (z ZKSDK) IsConnected() bool {
	return z.handle != 0
}

func (z *ZKSDK) Connect(connstr string) error {
	strPtr, err := strToPtr(connstr)
	if err != nil {
		return err
	}
	fun := z._getDllFunc("Connect")
	dllres, dllerr, _ := syscall.Syscall(fun, 1, strPtr, 0, 0)
	z.handle = 0

	if dllerr == 0 {
		z.handle = *(*int)(unsafe.Pointer(dllres))
	} else {
		proc := z._getDllFunc("PullLastError")
		dllres, _, _ = syscall.Syscall(proc, 0, 0, 0, 0)
		return errors.New(fmt.Sprintf("%d", dllres))  // TODO
	}
	return nil
}

func (z *ZKSDK) Disconnect() {
	if z.handle == 0 {
		return
	}
	fun := z._getDllFunc("Disconnect")
	_, _, _ = syscall.Syscall(fun, 1, uintptr(unsafe.Pointer(&z.handle)), 0, 0)
	z.handle = 0
}

func (z ZKSDK) ControlDevice(operation string, p1 int, p2 int, p3 int, p4 int) error {
	opcode, ok := ControlOperations[operation]
	if ! ok {
		return errors.New("unknown operation")
	}

	optionsPtr := 0
	fun := z._getDllFunc("ControlDevice")
	_, dllerr, _ := syscall.Syscall9(
		fun,
		7,
		uintptr(unsafe.Pointer(&z.handle)),
		uintptr(opcode),
		uintptr(p1),
		uintptr(p2),
		uintptr(p3),
		uintptr(p4),
		uintptr(unsafe.Pointer(&optionsPtr)),
		0,
		0,
	)
	fmt.Println(dllerr)
	return nil
}

func (z ZKSDK) _getDllFunc(name string) uintptr {
	fun, err := syscall.GetProcAddress(z.dll, name)
	if err != nil {
		panic(errors.New(fmt.Sprintf("%s: %s", name, err)))
	}
	return fun
}

func strToPtr(str string) (uintptr, error) {
	strBytes, err := syscall.BytePtrFromString(str)
	if err != nil {
		return 0, err
	}
	return uintptr(unsafe.Pointer(strBytes)), nil
}
