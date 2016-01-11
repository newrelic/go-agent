package crossagent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

var crossAgentDir = mustExist("crossagent/cross_agent_tests")

// ReadFile reads a file from the crossagent tests directory given as with
// ioutil.ReadFile.
func ReadFile(name string) ([]byte, error) {
	data, err := ioutil.ReadFile(filepath.Join(crossAgentDir, name))
	if err == nil {
		return data, nil
	}

	if os.IsNotExist(err) {
		err = &notExistError{name}
	}
	return data, err
}

// ReadJSON takes the name of a file and parses it using JSON.Unmarshal into
// the interface given.
func ReadJSON(name string, v interface{}) error {
	data, err := ReadFile(name)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ReadDir reads a directory relative to crossagent tests and returns an array
// of absolute filepaths of the files in that directory.
func ReadDir(name string) ([]string, error) {
	dir := filepath.Join(crossAgentDir, name)

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &notExistError{name}
		}
		return nil, err
	}

	var files []string
	for _, info := range entries {
		if !info.IsDir() {
			files = append(files, filepath.Join(dir, info.Name()))
		}
	}
	return files, nil
}

type submoduleError struct {
	dir string
}

func (e *submoduleError) Error() string {
	return fmt.Sprintf("cross agent tests not found: %s\n"+
		"Make sure you have run 'git submodule init && git submodule update'\n"+
		"to initialize the cross_agent_tests submodule.", e.dir)
}

type notExistError struct {
	name string
}

func (e *notExistError) Error() string {
	return fmt.Sprintf("cross_agent_tests: file not found: %s\n"+
		"Make sure you have run 'git submodule update' and that the submodule\n"+
		"points to the correct revision.", e.name)
}

func mustExist(submodule string) string {
	_, here, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(here), "..", submodule)

	_, err := os.Stat(dir)
	if err == nil {
		return dir
	}

	if os.IsNotExist(err) {
		err = &submoduleError{dir}
	}

	panic(err)
}
