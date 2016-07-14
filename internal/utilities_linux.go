package internal

import (
	"syscall"
	"time"
)

func timevalToDuration(tv syscall.Timeval) time.Duration {
	return time.Duration(tv.Nano()) * time.Nanosecond
}

//Get the current process kerneltime usertime
func getProcessTimes() (time.Duration, time.Duration, error) {
	ru := syscall.Rusage{}
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru)

	if nil == err {
		user := timevalToDuration(ru.Utime)
		sys := timevalToDuration(ru.Stime)
		return sys, user, nil
	}

	return 0, 0, err
}
