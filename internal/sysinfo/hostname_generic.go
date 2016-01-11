// +build !linux

package sysinfo

import "os"

func Hostname() (string, error) {
	return os.Hostname()
}
