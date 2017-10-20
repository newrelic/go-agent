// +build !linux

package sysinfo

func BootID() (string, error) {
	return "", ErrFeatureUnsupported
}
