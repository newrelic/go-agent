package sysinfo

import (
	"os"
	"strings"
)

// Hostname returns the host name.
func Hostname(useDynoNames bool, dynoNamePrefixesToShorten []string) (string, error) {
	if dyno := getDynoName(os.Getenv, useDynoNames, dynoNamePrefixesToShorten); dyno != "" {
		return dyno, nil
	}
	return getHostname()
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
