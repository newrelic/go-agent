package internal

import (
	"syscall"
	"time"
)

func filetimeToInt64(ft *syscall.Filetime) int64 {
	var time64 int64
	time64 = int64(ft.HighDateTime)<<32 + int64(ft.LowDateTime)
	return time64
}

//Get the current process kerneltime usertime
func getProcessTimes() (time.Duration, time.Duration, error) {

	var creationTime syscall.Filetime
	var exitTime syscall.Filetime
	var kernelTime syscall.Filetime
	var userTime syscall.Filetime

	curhandle, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, 0, err
	}

	errgetTimes := syscall.GetProcessTimes(curhandle, &creationTime, &exitTime, &kernelTime, &userTime)
	if errgetTimes != nil {
		return 0, 0, errgetTimes
	}

	sys := filetimeToInt64(&kernelTime)
	user := filetimeToInt64(&userTime)

	return time.Duration(sys), time.Duration(user), nil
}
