package sysinfo

import (
	"syscall"
	"unsafe"
)

func PhysicalMemoryBytes() (uint64, error) {
	mib := []int32{6 /* CTL_HW */, 24 /* HW_MEMSIZE */}

	buf := make([]byte, 8)
	buf_len := uintptr(8)

	_, _, e1 := syscall.Syscall6(syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])), uintptr(len(mib)),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&buf_len)),
		uintptr(0), uintptr(0))

	if e1 != 0 {
		return 0, e1
	}

	if buf_len != 8 {
		return 0, syscall.EIO
	}

	return *(*uint64)(unsafe.Pointer(&buf[0])), nil
}
