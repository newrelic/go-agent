# Copyright 2020 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

set -x
set -e

# inputs
# 1: repo pin; example: github.com/rewrelic/go-agent@v1.9.0
pin_go_dependency() {
  if [[ ! -z "$1" ]]; then
    echo "Pinning: $1"
    repo=$(echo "$1" | cut -d '@' -f1)
    pinTo=$(echo "$1" | cut -d '@' -f2)
    set +e
    go get -u "$repo" # this go get will fail to build
    set -e
    cd "$GOPATH"/src/"$repo"
    git checkout "$pinTo"
    cd -
  fi
}

verify_go_fmt() {
  needsFMT=$(gofmt -d .)
  if [ ! -z "$needsFMT" ]; then
    echo "$needsFMT"
    echo "Please format your code with \"gofmt .\""
    # exit 1
  fi
}

pwd=$(pwd)
version=$(go version)
echo $version

tmp=$(echo $version | cut -d 'o' -f4)
shortVersion=${tmp%.*}

IFS=","
for dir in $DIRS; do
  cd "$pwd/$dir"

  # replace go-agent with local pull
  go mod edit -replace github.com/newrelic/go-agent/v3="$pwd"/v3

  # manage dependencies
  go mod tidy -go=$shortVersion -compat=$shortVersion
  pin_go_dependency "$PIN"

  # run tests
  go test -race -benchtime=1ms -bench=. ./...
  go vet ./...
  verify_go_fmt
  
  # Test again against the latest version of the dependencies to ensure that
  # our instrumentation is up to date.  TODO: Perhaps it is possible to
  # upgrade all go.mod dependencies to latest master with a go command.
  if [ -n "$EXTRATESTING" ]; then
    eval "$EXTRATESTING"
    go test -race -benchtime=1ms -bench=. ./...
  fi
done
