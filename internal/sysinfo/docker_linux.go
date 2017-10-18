package sysinfo

import (
  "bufio"
  "bytes"
  "fmt"
  "io"
  "os"
)

func DockerID() (string, error) {
  f, err := os.Open("/proc/self/cgroup")
  if err != nil {
    return "", err
  }
  defer f.Close()
  return parseDockerID(f)
}

// parseDockerID reads (normally from /proc/self/cgroup) and parses input to
// find what looks like a cgroup from a Docker container. This is conveniently
// also the hash that represents the container. Returns a 64-character hex
// string or an error.
func parseDockerID(r io.Reader) (string, error) {
  // Each line in the cgroup file consists of three colon delimited fields.
  //   1. hierarchy ID  - we don't care about this
  //   2. subsystems    - comma separated list of cgroup subsystem names
  //   3. control group - control group to which the process belongs
  //
  // Example
  //   5:cpuacct,cpu,cpuset:/daemons

  for scanner := bufio.NewScanner(r); scanner.Scan(); {
    line := scanner.Bytes()
    cols := bytes.SplitN(line, []byte(":"), 3)

    if len(cols) < 3 {
      continue
    }

    //  We're only interested in the cpu subsystem.
    if !isCPUCol(cols[1]) {
      continue
    }

    // We're only interested in Docker generated cgroups.
    /* Reference Implementation:
       case cpu_cgroup
       # docker native driver w/out systemd (fs)
       when %r{^/docker/([0-9a-f]+)$}                      then $1
       # docker native driver with systemd
       when %r{^/system\.slice/docker-([0-9a-f]+)\.scope$} then $1
       # docker lxc driver
       when %r{^/lxc/([0-9a-f]+)$}                         then $1
    */
    var id string
    if bytes.HasPrefix(cols[2], []byte("/docker/")) {
      id = string(cols[2][len("/docker/"):])
    } else if bytes.HasPrefix(cols[2], []byte("/lxc/")) {
      id = string(cols[2][len("/lxc/"):])
    } else if bytes.HasPrefix(cols[2], []byte("/system.slice/docker-")) &&
      bytes.HasSuffix(cols[2], []byte(".scope")) {
      id = string(cols[2][len("/system.slice/docker-") : len(cols[2])-len(".scope")])
    } else {
      continue
    }

    if err := validateDockerID(id); err != nil {
      // We can stop searching at this point, the CPU subsystem should
      // only occur once, and its cgroup is not docker or not a format
      // we accept.
      return "", err
    }
    return id, nil
  }

  return "", ErrIdentifierNotFound
}

func isCPUCol(col []byte) bool {
  // Sometimes we have multiple subsystems in one line (as in this example from
  // https://source.datanerd.us/newrelic/cross_agent_tests/blob/master/docker_container_id/docker-1.1.2-native-driver-systemd.txt):
  //
  // 3:cpuacct,cpu:/system.slice/docker-67f98c9e6188f9c1818672a15dbe46237b6ee7e77f834d40d41c5fb3c2f84a2f.scope
  splitCSV := func(r rune) bool { return r == ',' }
  subsysCPU := []byte("cpu")

  for _, subsys := range bytes.FieldsFunc(col, splitCSV) {
    if bytes.Equal(subsysCPU, subsys) {
      return true
    }
  }
  return false
}

type invalidDockerID string

func (e invalidDockerID) Error() string {
  return fmt.Sprintf("Docker container id has unrecognized format, id=%q", string(e))
}

func isHex(r rune) bool {
  return ('0' <= r && r <= '9') || ('a' <= r && r <= 'f')
}

func validateDockerID(id string) error {
  if len(id) != 64 {
    return invalidDockerID(id)
  }

  for _, c := range id {
    if !isHex(c) {
      return invalidDockerID(id)
    }
  }

  return nil
}
