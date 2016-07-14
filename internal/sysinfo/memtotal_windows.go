package sysinfo

import (
	"syscall"
	"unsafe"
)

// PhysicalMemoryBytes returns the total amount of host memory.
// https://msdn.microsoft.com/en-us/library/windows/desktop/cc300158(v=vs.85).aspx
func PhysicalMemoryBytes() (uint64, error) {
	var mod = syscall.NewLazyDLL("kernel32.dll")
	var proc = mod.NewProc("GetPhysicallyInstalledSystemMemory")
	var memkb uint64

	ret, _, err := proc.Call(uintptr(unsafe.Pointer(&memkb)))
	//return value TRUE(1) succeeds, FAILED(0) fails
	if ret != 1 {
		return 0, err
	}

	return memkb * 1024, nil
}
