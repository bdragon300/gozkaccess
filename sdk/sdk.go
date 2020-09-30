package sdk

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var ControlOperations = map[string]int{
	"output": 1,
	"cancel_alarm": 2,
	"restart": 3,
}

// ZKSDK is an interface to dll SDK functions. This struct keeps dll handle and
// and handle to a current device connection.
type ZKSDK struct {
	handle int
	dll syscall.Handle
}

// NewZKSDK loads a dll with given path and returns new ZKSDK instance with
// handle to that dll and empty connection handle
func NewZKSDK(dllpath string) (*ZKSDK, error) {
	dllHandle, err := syscall.LoadLibrary(dllpath)
	if err != nil {
		return nil, err
	}

	return &ZKSDK{0, dllHandle}, nil
}

// IsConnected returns true when we are connected to a device
func (z *ZKSDK) IsConnected() bool {
	return z.handle != 0
}

// Connect begins a connection to a device with given connstr
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

// Disconnect disconnects from a device
func (z *ZKSDK) Disconnect() {
	if z.handle == 0 {
		return
	}
	fun := z._getDllFunc("Disconnect")
	_, _, _ = syscall.Syscall(fun, 1, uintptr(unsafe.Pointer(&z.handle)), 0, 0)
	z.handle = 0
}

// ControlDevice sends control command to device. Typically this command could be
// relay switching, restarting a device and so on. `operation` parameter must be
// one of `ControlOperations`. The rest parameters meaning is depended on operation.
// See SDK documentation.
func (z *ZKSDK) ControlDevice(operation string, p1 int, p2 int, p3 int, p4 int) error {
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
		0)
	if dllerr < 0 {
		return errors.New(fmt.Sprintf("function GetRTLog returned %d", dllerr))
	}
	return nil
}


// GetRTLog retrieves unread realtime events from a device.
func (z *ZKSDK) GetRTLog(bufferSize uint) ([]string, error) {
	buf := make([]byte, bufferSize)
	fun := z._getDllFunc("GetRTLog")
	_, dllerr, _ := syscall.Syscall(
		fun,
		3,
		uintptr(unsafe.Pointer(&z.handle)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(bufferSize))
	if dllerr < 0 {
		return nil, errors.New(fmt.Sprintf("function GetRTLog returned %d", dllerr))
	}

	lines := strings.Split(string(buf), "\r\n")
	return lines[:len(lines) - 1], nil
}

// SearchDevice performs network scan in order to collect available ZK devices.
func (z *ZKSDK) SearchDevice(broadcastAddress string, bufferSize uint) ([]string, error) {
	buf := make([]byte, bufferSize)
	fun := z._getDllFunc("SearchDevice")
	proto := "UDP"
	_, dllerr, _ := syscall.Syscall(
		fun,
		3,
		uintptr(unsafe.Pointer(&proto)),
		uintptr(unsafe.Pointer(&broadcastAddress)),
		uintptr(unsafe.Pointer(&buf)))
	if dllerr < 0 {
		return nil, errors.New(fmt.Sprintf("function SearchDevice returned %d", dllerr))
	}

	lines := strings.Split(string(buf), "\r\n")
	return lines[:len(lines) - 1], nil
}

// GetDeviceParam fetches given device parameters
func (z *ZKSDK) GetDeviceParam(parameters []string, bufferSize uint) (map[string]string, error) {
	buf := make([]byte, bufferSize)
	fun := z._getDllFunc("GetDeviceParam")
	paramLen := len(parameters)
	res := make(map[string]string)

	// Device can return maximum 30 parameters for one call. See SDK
	//  docs. So fetch them in loop by bunches of 30 items
	for offset := 0; offset < paramLen; offset += 30 {
		query := strings.Join(parameters[offset:offset + 30], ",")
		_, dllerr, _ := syscall.Syscall6(
			fun,
			4,
			uintptr(unsafe.Pointer(&z.handle)),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(bufferSize),
			uintptr(unsafe.Pointer(&query)),
			0,
			0)
		if dllerr < 0 {
			return nil, errors.New(fmt.Sprintf("function GetDeviceParam returned %d", dllerr))
		}

		for _, pair := range strings.Split(string(buf), ",") {
			keyval := strings.SplitN(pair, "=", 2)  // [key, value]
			res[keyval[0]] = keyval[1]
		}
	}

	if len(res) != len(parameters) {
		return nil, errors.New("parameters returned by a device are differ than parameters was requested")
	}

	return res, nil
}

// SetDeviceParam sets given device parameters
func (z *ZKSDK) SetDeviceParam(parameters map[string]string) error {
	if len(parameters) == 0 {
		return nil
	}

	fun := z._getDllFunc("SetDeviceParam")
	setParams := func (query string) error {
		_, dllerr, _ := syscall.Syscall(
			fun,
			2,
			uintptr(unsafe.Pointer(&z.handle)),
			uintptr(unsafe.Pointer(&query)),
			0)
		if dllerr < 0 {
			return errors.New(fmt.Sprintf("function SetDeviceParam returned %d", dllerr))
		}
		return nil
	}

	// Device can accept maximum 20 parameters for one call. See SDK
	//  docs. So send them in loop by bunches of 20 items
	pairs := make([]string, 20)
	c := 0
	for key, value := range parameters {
		pairs[c] = fmt.Sprintf("%s=%s", key, value)
		c++
		if c >= 20 {
			c = 0
			if err := setParams(strings.Join(pairs, ",")); err != nil {
				return err
			}
		}
	}
	if c > 0 {
		if err := setParams(strings.Join(pairs[:c], ",")); err != nil {
			return err
		}
	}

	return nil
}

func (z *ZKSDK) _getDllFunc(name string) uintptr {
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
