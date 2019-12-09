// +build !linux

package sysinfo

import (
	"os"
)

func getHostname(getenv func(string) string, useDynoNames bool, dynoNamePrefixesToShorten []string) (string, error) {
	hostname.Lock()
	defer hostname.Unlock()
	if hostname.name != "" {
		return hostname.name, nil
	}

	if host := getDynoName(getenv, useDynoNames, dynoNamePrefixesToShorten); host != "" {
		hostname.name = host
		return host, nil
	}

	host, err := os.Hostname()
	hostname.name = host
	return host, err
}
