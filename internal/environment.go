package internal

import (
	"encoding/json"
	"reflect"
	"runtime"
)

type environment struct {
	Compiler string `env:"Compiler"`
	GOARCH   string `env:"GOARCH"`
	GOOS     string `env:"GOOS"`
	Version  string `env:"Version"`
}

var (
	sampleEnvironment = environment{
		Compiler: "comp",
		GOARCH:   "arch",
		GOOS:     "goos",
		Version:  "vers",
	}
)

func newEnvironment() environment {
	return environment{
		Compiler: runtime.Compiler,
		GOARCH:   runtime.GOARCH,
		GOOS:     runtime.GOOS,
		Version:  runtime.Version(),
	}
}

func (e environment) MarshalJSON() ([]byte, error) {
	var arr [][]interface{}

	val := reflect.ValueOf(e)
	numFields := val.NumField()

	arr = make([][]interface{}, numFields)

	for i := 0; i < numFields; i++ {
		v := val.Field(i)
		t := val.Type().Field(i).Tag.Get("env")

		arr[i] = []interface{}{
			t,
			v.Interface(),
		}
	}

	return json.Marshal(arr)
}
