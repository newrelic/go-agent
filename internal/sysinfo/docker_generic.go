// +build !linux

package sysinfo

func DockerID() (string, error) {
  return "", ErrFeatureUnsupported
}
