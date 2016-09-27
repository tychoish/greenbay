package output

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/greenbay"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

// GoTest defines a ResultsProducer implementation that generates
// output in the format of "go test -v"
type GoTest struct {
	buf *bytes.Buffer
}

// Populate generates output, based on the content (via the Results()
// method) of an amboy.Queue instance. All jobs processed by that
// queue must also implement the greenbay.Checker interface.
func (r *GoTest) Populate(queue amboy.Queue) error {
	err := produceResults(r.buf, jobsToCheck(queue.Results()))
	if err != nil {
		return errors.Wrap(err, "problem generating gotest results")
	}

	return nil
}

// ToFile writes the "go test -v" output to a file.
func (r *GoTest) ToFile(fn string) error {
	return errors.Wrapf(ioutil.WriteFile(fn, r.buf.Bytes(), 0644),
		"problem writing output to %s", fn)
}

// Print writes the "go test -v" output to standard output.
func (r *GoTest) Print() error {
	fmt.Println(strings.TrimRight(r.buf.String(), "\n"))

	return nil
}

////////////////////////////////////////////////////////////////////////
//
// Implementation of go test output generation
//
////////////////////////////////////////////////////////////////////////

func produceResults(w io.Writer, checks <-chan workUnit) error {
	catcher := grip.NewCatcher()

	for wu := range checks {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}

		printTestResult(w, wu.output)
	}

	return catcher.Resolve()
}

func printTestResult(w io.Writer, check greenbay.CheckOutput) {
	fmt.Fprintln(w, "=== RUN", check.Name)
	fmt.Fprintln(w, "    message:", check.Message)
	fmt.Fprintln(w, "    error:", check.Error)

	dur := check.Timing.Start.Sub(check.Timing.End)

	if check.Passed {
		fmt.Fprintf(w, "--- PASS: %s (%s)\n", check.Name, dur)
	} else {
		fmt.Fprintf(w, "--- FAIL: %s (%s)\n", check.Name, dur)
	}
}