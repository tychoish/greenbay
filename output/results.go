package output

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/greenbay"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

////////////////////////////////////////////////////////////////////////
//
// Public Interface for results.json output format
//
////////////////////////////////////////////////////////////////////////

// Results defines a ResultsProducer implementation for the Evergreen
// results.json output format.
type Results struct {
	out *resultsDocument
}

// Populate generates output, based on the content (via the Results()
// method) of an amboy.Queue instance. All jobs processed by that
// queue must also implement the greenbay.Checker interface.
func (r *Results) Populate(queue amboy.Queue) error {
	out, err := newResultsDocument(queue)
	if err != nil {
		return errors.Wrap(err, "problem generating results structure")
	}

	r.out = out

	return nil
}

// ToFile writes results.json output output to the specified file.
func (r *Results) ToFile(fn string) error {
	if err := r.out.writeToFile(fn); err != nil {
		return errors.Wrap(err, "problem writing results to json")
	}

	return nil
}

// Print writes, to standard output, the results.json data.
func (r *Results) Print() error {
	if err := r.out.print(); err != nil {
		return errors.Wrap(err, "problem printing results")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////
//
// Implementation for construction and generation of resultsDocument structure.
//
////////////////////////////////////////////////////////////////////////

// type definition and constructors

type resultsDocument struct {
	Results []*resultsItem `bson:"results" json:"results" yaml:"results"`
}

type resultsItem struct {
	Status  string        `bson:"status" json:"status" yaml:"status"`
	Test    string        `bson:"test_file" json:"test_file" yaml:"test_file"`
	Code    int           `bson:"exit_code" json:"exit_code" yaml:"exit_code"`
	Elapsed time.Duration `bson:"elapsed" json:"elapsed" yaml:"elapsed"`
	Start   time.Time     `bson:"start" json:"start" yaml:"start"`
	End     time.Time     `bson:"end" json:"end" yaml:"end"`
}

func newResultsDocument(queue amboy.Queue) (*resultsDocument, error) {
	r := &resultsDocument{}

	if err := r.populate(jobsToCheck(queue.Results())); err != nil {
		return nil, errors.Wrap(err, "problem constructing results document")
	}

	return r, nil
}

// implementation of content generation.

func (r *resultsDocument) populate(checks <-chan workUnit) error {
	catcher := grip.NewCatcher()
	for wu := range checks {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}

		r.addItem(wu.output)
	}

	return catcher.Resolve()
}

func (r *resultsDocument) addItem(check greenbay.CheckOutput) {
	item := &resultsItem{
		Test:    check.Name,
		Elapsed: check.Timing.Duration(),
		Start:   check.Timing.Start,
		End:     check.Timing.End,
	}

	if check.Passed {
		item.Status = "fail"
	} else {
		item.Status = "pass"
		item.Code = 1
	}

	r.Results = append(r.Results, item)
}

// output production

func (r *resultsDocument) write(w io.Writer) error {
	out, err := json.MarshalIndent(r, "   ", "")
	if err != nil {
		return errors.Wrap(err, "problem converting results to json")
	}

	if _, err = w.Write(out); err != nil {
		return errors.Wrapf(err, "problem writing results to %s (%T)", w, w)
	}

	return nil
}

func (r *resultsDocument) print() error {
	return r.write(os.Stdout)
}

func (r *resultsDocument) writeToFile(fn string) error {
	w, err := os.Create(fn)
	if err != nil {
		return errors.Wrapf(err, "problem opening file %s", fn)
	}
	defer grip.Error(w.Close())

	if err := r.write(w); err != nil {
		return errors.Wrapf(err, "problem writing json to file: %s", fn)
	}

	grip.Infoln("wrote results document to:", fn)
	return nil
}
