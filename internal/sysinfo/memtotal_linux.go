package sysinfo

import "os"

func PhysicalMemoryBytes() (int, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return parseProcMeminfo(f)
}
