package errors

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
)

// Example illustrates how a StackTracer can be created from a wrapped error
// and how the according stack trace would look like. Note that since this
// example is compiled and executed automatically by the testing package the
// produced stack trace is a bit deeper than shown here in the output.
// To keep the example clean and reproducible only the first 3 stack frames
// are shown.
func Example() {
	err := doSomething()
	if err != nil {
		fmt.Println("ERROR:", err)
		err = WithStackTrace(err)
		// at this point we would usually call txn.Notice(err)
		st := err.(*StackTracer).StackTrace()
		for _, f := range st[:3] {
			fmt.Printf("%v:%v %v\n", filepath.Base(f.File), f.Line, f.Function)
		}
	}

	// Output:
	// ERROR: it failed: something went wrong
	// example_test.go:40 anotherFunction
	// example_test.go:36 doSomething
	// example_test.go:17 Example
}

func doSomething() error {
	return errors.Wrap(anotherFunction(), "it failed")
}

func anotherFunction() error {
	return errors.New("something went wrong")
}
