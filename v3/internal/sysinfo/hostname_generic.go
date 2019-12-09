// +build !linux

package sysinfo

import "os"

// Hostname returns the host name.
func Hostname() (string, error) {
	hostname.Lock()
	defer hostname.Unlock()
	if hostname.name != "" {
		return hostname.name, nil
	}

	host, err := os.Hostname()
	hostname.name = host
	return host, err
}
