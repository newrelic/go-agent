// +build !linux

package sysinfo

import "os"

func getHostname() (string, error) {
	return os.Hostname()
}
