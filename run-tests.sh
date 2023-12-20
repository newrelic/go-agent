#!/bin/bash
# run_tests.sh
export PATH=$PATH:/usr/local/go/bin
# Test directory is passed in as an argument
TEST_DIR=$1


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
go test -race -benchtime=1ms -bench=. -coverprofile=coverage.out -covermode=atomic -coverpkg=$pwd ./...
go vet ./...
verify_go_fmt

# Remove sql_driver_optional_methods from coverage.out file if it exists
sed -i '/sql_driver_optional_methods/d' coverage.out
