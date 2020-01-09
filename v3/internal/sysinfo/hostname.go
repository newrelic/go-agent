package sysinfo

// Hostname returns the host name.
func Hostname() (string, error) {
	return getHostname()
}
