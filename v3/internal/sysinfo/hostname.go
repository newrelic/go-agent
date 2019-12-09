package sysinfo

import (
	"os"
	"strings"
	"sync"
)

var hostname struct {
	sync.Mutex
	name string
}

func getDynoName(getenv func(string) string, useDynoNames bool, dynoNamePrefixesToShorten []string) string {
	if !useDynoNames {
		return ""
	}

	dyno := getenv("DYNO")
	if dyno == "" {
		return dyno
	}

	for _, prefix := range dynoNamePrefixesToShorten {
		if prefix == "" {
			continue
		}
		if strings.HasPrefix(dyno, prefix) {
			dyno = prefix + ".*"
			break
		}
	}

	return dyno
}

// Hostname returns the host name.
func Hostname(useDynoNames bool, dynoNamePrefixesToShorten []string) (string, error) {
	return getHostname(os.Getenv, useDynoNames, dynoNamePrefixesToShorten)
}

// ResetHostname resets the cached hostname value. Should only be used for
// testing.
func ResetHostname() {
	hostname.Lock()
	defer hostname.Unlock()
	hostname.name = ""
}
