set -x
set -e

LATEST_VERSION="go1.13"

pwd=`pwd`

IFS=","
for dir in $DIRS
do
	cd "$pwd/$dir"

	# This can be removed once we add go.mod files for everything.
	go get -t ./...

	go test -race -benchtime=1ms -bench=. ./...
	go vet ./...

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
