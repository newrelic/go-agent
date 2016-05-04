package main

import (
	"encoding/json"
	"fmt"

	"github.com/newrelic/go-sdk/internal/utilization"
	"github.com/newrelic/go-sdk/log"
)

func main() {
	log.SetLogFile("stdout", log.LevelDebug)

	util := utilization.Gather(utilization.Config{
		DetectAWS:    true,
		DetectDocker: true,
	})

	js, err := json.MarshalIndent(util, "", "\t")
	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		fmt.Printf("%s\n", js)
	}
}
