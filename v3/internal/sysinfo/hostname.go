package sysinfo

import (
	"os"
	"strings"
)

func GetDynoName(useDynoNames bool, dynoNamePrefixesToShorten []string) string {
	return getDynoName(os.Getenv, useDynoNames, dynoNamePrefixesToShorten)
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
