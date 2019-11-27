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
	fi

	# go get is necessary for testing v2 integrations since they do not have
	# a go.mod file.
	if [[ $dir =~ "_integrations" ]]; then
		go get -t ./...
	fi
	# avoid testing v3 code when testing v2 newrelic package
	if [ $dir == "." ]; then
		rm -rf v3/
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
		golint -set_exit_status ./...

		# only run gofmt on a single version as the format changed from 1.10 to
		# 1.11.
		if [ -n "$(gofmt -s -l .)" ]; then
			exit 1
		fi
	fi
done
