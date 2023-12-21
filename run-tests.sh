#!/bin/bash
# run_tests.sh
export PATH=$PATH:/usr/local/go/bin
# Test directory is passed in as an argument
TEST_DIR=$1
COVERAGE_DIR=$2
CODECOV_TOKEN=$3
COVERAGE_FILE="$COVERAGE_DIR/coverage.out"

echo "Coverage profile will be created at $COVERAGE_FILE"

verify_go_fmt() {
  needsFMT=$(gofmt -d .)
  if [ ! -z "$needsFMT" ]; then
    echo "$needsFMT"
    echo "Please format your code with \"gofmt .\""
    # exit 1
  fi
}
cd go-agent/v3
# replace go-agent with local pull
go mod edit -replace github.com/newrelic/go-agent/v3="$pwd"/v3
cd ../
cd $TEST_DIR

go mod tidy
# Run Tests and Create Cover Profile for Code Coverage
go test -race -benchtime=1ms -bench=. -coverprofile="$COVERAGE_FILE" -covermode=atomic -coverpkg=./... ./...
go vet ./...
verify_go_fmt
echo "Coverage Profile Created at $COVERAGE_FILE"
# Remove sql_driver_optional_methods from coverage.out file if it exists
sed -i '/sql_driver_optional_methods/d' "$COVERAGE_FILE"

## CodeCov Uploader
curl https://keybase.io/codecovsecurity/pgp_keys.asc | gpg --no-default-keyring --import # One-time step
curl -Os https://uploader.codecov.io/latest/linux/codecov
curl -Os https://uploader.codecov.io/latest/linux/codecov.SHA256SUM
curl -Os https://uploader.codecov.io/latest/linux/codecov.SHA256SUM.sig
gpg --verify codecov.SHA256SUM.sig codecov.SHA256SUM
shasum -a 256 -c codecov.SHA256SUM
chmod +x codecov
./codecov -t ${CODECOV_TOKEN}