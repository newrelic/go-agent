# Copyright 2020 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

set -x
set -e

LATEST_VERSION="go1.15"

# NOTE: Once we get rid of travis for good, this whole section can be removed
# along with the .travis.yml file.
if [[ -n "$(go version | grep $LATEST_VERSION)" ]] && [[ "$TRAVIS" == "true" ]]; then
  echo "Installing updated glibc\n"
  # can we get this from an actual repository?
  curl -LO 'http://launchpadlibrarian.net/130794928/libc6_2.17-0ubuntu4_amd64.deb'
  sudo dpkg -i libc6_2.17-0ubuntu4_amd64.deb
else
  echo "Skipping glibc update\n"
fi

pwd=$(pwd)

IFS=","
for dir in $DIRS; do
  cd "$pwd/$dir"

  if [ -f "go.mod" ]; then
    go mod edit -replace github.com/newrelic/go-agent/v3=$pwd/v3
  fi

  # go get is necessary for testing v2 integrations since they do not have
  # a go.mod file.
  if [[ $dir =~ "_integrations" ]]; then
    go get -t ./...
  fi
  # avoid testing v3 code when testing v2 newrelic package
  if [ $dir == "." ]; then
    rm -rf v3/
  else
    # Only v3 code version 1.9+ needs GRPC dependencies
    VERSION=$(go version)
    V17="1.7"
    V18="1.8"
    V19="1.9"
    if [[ "$VERSION" =~ .*"$V17".* || "$VERSION" =~ .*"$V18".* ]]; then
      echo "Not installing GRPC for old versions"
    elif [[ "$VERSION" =~ .*"$V19" ]]; then
      # install v3 dependencies that support this go version
      set +e
      go get -u google.golang.org/grpc # this go get will fail to build
      set -e
      cd $GOPATH/src/google.golang.org/grpc
      git checkout v1.31.0
      cd -

      go get -u github.com/golang/protobuf/protoc-gen-go
    else
      go get -u github.com/golang/protobuf/protoc-gen-go
      go get -u google.golang.org/grpc
    fi
  fi

  go test -race -benchtime=1ms -bench=. ./...
  go vet ./...

  # Test again against the latest version of the dependencies to ensure that
  # our instrumentation is up to date.  TODO: Perhaps it is possible to
  # upgrade all go.mod dependencies to latest master with a go command.
  if [ -n "$EXTRATESTING" ]; then
    eval "$EXTRATESTING"
    go test -race -benchtime=1ms -bench=. ./...
  fi

  if [[ -n "$(go version | grep $LATEST_VERSION)" ]]; then
    # golint requires a supported version of Go, which in practice is currently 1.9+.
    # See: https://github.com/golang/lint#installation
    # For simplicity, run it on a single Go version.
    go get -u golang.org/x/lint/golint
    # do not expect golint to be in the PATH, instead use go list to discover
    # the path to the binary.
    $(go list -f {{.Target}} golang.org/x/lint/golint) -set_exit_status ./...

    # only run gofmt on a single version as the format changed from 1.10 to
    # 1.11.
    if [ -n "$(gofmt -s -l .)" ]; then
      exit 1
    fi
  fi
done
