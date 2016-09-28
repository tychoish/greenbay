package output

import (
	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

type Options struct {
	writeFile   bool
	writeStdOut bool
	fn          string
	format      string
}

func NewOptions(fn, format string, quiet bool) (*Options, error) {
	_, exists := GetResultsFactory(format)
	if !exists {
		return nil, errors.Errorf("no results format named '%s' exists", format)
	}

	o := &Options{}
	o.format = format
	o.writeStdOut = !quiet

	if fn != "" {
		o.writeFile = true
		o.fn = fn
	}

	return o, nil
}

func (o *Options) GetResultsProducer() (ResultsProducer, error) {
	factory, ok := GetResultsFactory(o.format)
	if !ok {
		return nil, errors.Errorf("no results format named '%s' exists", o.format)
	}

	rp := factory()

	return rp, nil
}

func (o *Options) ProduceResults(q amboy.Queue) error {
	rp, err := o.GetResultsProducer()
	if err != nil {
		return errors.Wrap(err, "problem fetching results producer")
	}

	if err := rp.Populate(q); err != nil {
		return errors.Wrap(err, "problem generating results content")
	}

	// Actually write output to respective streems
	catcher := grip.NewCatcher()

	if o.writeStdOut {
		catcher.Add(rp.Print())
	}

	if o.writeFile {
		catcher.Add(rp.ToFile(o.fn))
	}

	return catcher.Resolve()
}
