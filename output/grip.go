package output

import (
	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

// GripOutput provides a ResultsProducer implementation that writes
// the results of a greenbay run to logging using the grip logging
// package.
type GripOutput struct {
	gripOutputData
}

// Populate generates output messages based on a sequence of
// amboy.Jobs. All jobs must also implement the greenbay.Checker
// interface. Returns an error if there are any invalid jobs.
func (r *GripOutput) Populate(jobs <-chan amboy.Job) error {
	catcher := grip.NewCatcher()

	r.useJsonLoggers = false

	for wu := range jobsToCheck(jobs) {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}

		dur := wu.output.Timing.Start.Sub(wu.output.Timing.End)
		if wu.output.Passed {
			r.passedMsgs = append(r.passedMsgs,
				message.NewFormatted("PASSED: '%s' [time='%s', msg='%s', error='%s']",
					wu.output.Name, dur, wu.output.Message, wu.output.Error))
		} else {
			r.failedMsgs = append(r.failedMsgs,
				message.NewFormatted("FAILED: '%s' [time='%s', msg='%s', error='%s']",
					wu.output.Name, dur, wu.output.Message, wu.output.Error))
		}
	}

	return catcher.Resolve()
}

// JSONResults provides a structured output JSON format.
type JSONResults struct {
	gripOutputData
}

// Populate generates output messages based on a sequence of
// amboy.Jobs. All jobs must also implement the greenbay.Checker
// interface. Returns an error if there are any invalid jobs.
func (r *JSONResults) Populate(jobs <-chan amboy.Job) error {
	catcher := grip.NewCatcher()
	r.useJsonLoggers = true

	for wu := range jobsToCheck(jobs) {
		if wu.err != nil {
			catcher.Add(wu.err)
			continue
		}
		if wu.output.Passed {
			r.passedMsgs = append(r.passedMsgs, &jsonOutput{output: wu.output})
		} else {
			r.passedMsgs = append(r.failedMsgs, &jsonOutput{output: wu.output})
		}
	}
	return catcher.Resolve()
}

type gripOutputData struct {
	useJsonLoggers bool
	passedMsgs     []message.Composer
	failedMsgs     []message.Composer
}

// ToFile logs, to the specified file, the results of the greenbay
// operation. If any tasks failed, this operation returns an error.
func (r *gripOutputData) ToFile(fn string) error {
	var sender send.Sender
	var err error
	logger := grip.NewJournaler("greenbay")

	if r.useJsonLoggers {

	} else {
		sender, err = send.NewFileLogger("greenbay", fn, send.LevelInfo{Default: level.Info, Threshold: level.Info})
		if err != nil {
			return errors.Wrapf(err, "problem setting up output logger to file '%s'", fn)
		}
	}

	logger.SetSender(sender)

	r.logResults(logger)

	numFailed := len(r.failedMsgs)
	if numFailed > 0 {
		return errors.Errorf("%d test(s) failed", numFailed)
	}

	return nil
}

// Print logs, to standard output, the results of the greenbay
// operation. If any tasks failed, this operation returns an error.
func (r *gripOutputData) Print() error {
	logger := grip.NewJournaler("greenbay")
	sender, err := send.NewNativeLogger("greenbay", send.LevelInfo{Default: level.Info, Threshold: level.Info})
	if err != nil {
		return errors.Wrap(err, "problem setting up logger")
	}
	logger.SetSender(sender)

	r.logResults(logger)

	numFailed := len(r.failedMsgs)
	if numFailed > 0 {
		return errors.Errorf("%d test(s) failed", numFailed)
	}

	return nil
}

func (r *gripOutputData) logResults(logger grip.Journaler) {
	for _, msg := range r.passedMsgs {
		logger.Notice(msg)
	}

	for _, msg := range r.failedMsgs {
		logger.Alert(msg)
	}
}
