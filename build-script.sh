set -x
set -e

LATEST_VERSION="go1.13"

pwd=`pwd`

IFS=","
for dir in $DIRS
do
	cd "$pwd/$dir"

	if [ -f "go.mod" ]; then
		go mod edit -replace github.com/newrelic/go-agent/v3=$pwd/v3
	else
		go get -t ./...
	fi

	go test -race -benchtime=1ms -bench=. ./...
	go vet ./...

	# Test again against the latest version of the dependencies to ensure that
	# our instrumentation is up to date.  TODO: Perhaps it is possible to
	# upgrade all go.mod dependencies to latest master with a go command.
	if [ -n "$TESTMASTER" ]; then
		go get -u "$TESTMASTER@master"
		go test -race -benchtime=1ms -bench=. ./...
	fi

	if [[ -n "$(go version | grep $LATEST_VERSION)" ]]; then
		# golint requires a supported version of Go, which in practice is currently 1.9+.
		# See: https://github.com/golang/lint#installation
		# For simplicity, run it on a single Go version.
		go get -u golang.org/x/lint/golint
		golint -set_exit_status ./...

		# only run gofmt on a single version as the format changed from 1.10 to
		# 1.11.
		if [ -n "$(gofmt -s -l .)" ]; then
			exit 1
		fi
	fi
done
