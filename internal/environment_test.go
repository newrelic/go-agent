package internal

import (
	"encoding/json"
	"runtime"
	"testing"
)

func TestMarshalEnvironment(t *testing.T) {
	js, err := json.Marshal(&SampleEnvironment)
	if nil != err {
		t.Fatal(err)
	}
	expect := `[["Compiler","comp"],["GOARCH","arch"],["GOOS","goos"],["Version","vers"],["NumCPU","numCPU"]]`
	if string(js) != expect {
		t.Fatal(string(js))
	}
}

func TestEnvironmentFields(t *testing.T) {
	env := NewEnvironment()
	if env.Compiler != runtime.Compiler {
		t.Error(env.Compiler, runtime.Compiler)
	}
	if env.GOARCH != runtime.GOARCH {
		t.Error(env.GOARCH, runtime.GOARCH)
	}
	if env.GOOS != runtime.GOOS {
		t.Error(env.GOOS, runtime.GOOS)
	}
	if env.Version != runtime.Version() {
		t.Error(env.Version, runtime.Version())
	}
	if env.NumCPU != runtime.NumCPU() {
		t.Error(env.NumCPU, runtime.NumCPU())
	}
}
