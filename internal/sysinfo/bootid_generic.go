// +build !linux

package sysinfo

// BootID returns the boot ID of the executing kernel.
func BootID() (string, error) {
	return "", ErrFeatureUnsupported
}
