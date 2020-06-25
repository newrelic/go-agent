// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"github.com/newrelic/go-agent/internal"
	"os"
)

// This tool will take an encoded, compressed serverless payload and print it out in a human readable format.
// To use it on a Mac with Bash, copy the payload to the clipboard (the whole thing - `[2,"NR_LAMBDA_MONITORING",{"metadata_version"...]`
// without backticks) and then run the app using pbpaste.  Example from the root of the project directory:
//
// go run internal/tools/uncompress-serverless/main.go $(pbpaste)
func main() {

	compressed := []byte(os.Args[1])
	metadata, uncompressedData, e := internal.ParseServerlessPayload(compressed)
	if nil != e {
		panic(e)
	}
	js, _ := json.MarshalIndent(map[string]interface{}{"metadata": metadata, "data": uncompressedData}, "", "  ")
	fmt.Println(string(js))

}
