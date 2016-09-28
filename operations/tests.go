package operations

import (
	"context"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/greenbay/config"
	"github.com/mongodb/greenbay/output"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

////////////////////////////////////////////////////////////////////////
//
// run tests
//
////////////////////////////////////////////////////////////////////////

type GreenbayApp struct {
	out        *output.Options
	conf       *config.GreenbayTestConfig
	numWorkers int
	tests      []string
	suites     []string
}

func NewApp(confPath, outFn, format string, quiet bool, jobs int, suite, tests []string) (*GreenbayApp, error) {
	conf, err := config.ReadConfig(confPath)
	if err != nil {
		return nil, errors.Wrap(err, "problem parsing config file")
	}

	out, err := output.NewOptions(outFn, format, quiet)
	if err != nil {
		return nil, errors.Wrap(err, "problem generating output definition")
	}

	app := &GreenbayApp{
		conf:       conf,
		out:        out,
		numWorkers: jobs,
		tests:      tests,
		suites:     suite,
	}

	return app, nil
}

func (a *GreenbayApp) addSuites(q amboy.Queue) error {
	if len(a.suites) == 0 {
		return nil
	}

	catcher := grip.NewCatcher()

	for check := range a.conf.TestsForSuites(a.suites...) {
		if check.Err != nil {
			catcher.Add(check.Err)
		}
		catcher.Add(q.Put(check.Job))
	}

	return catcher.Resolve()
}

func (a *GreenbayApp) addTests(q amboy.Queue) error {
	if len(a.tests) == 0 {
		return nil
	}

	catcher := grip.NewCatcher()

	for check := range a.conf.TestsByName(a.tests...) {
		if check.Err != nil {
			catcher.Add(check.Err)
		}
		catcher.Add(q.Put(check.Job))
	}

	return catcher.Resolve()
}

func (a *GreenbayApp) Run(ctx context.Context) error {
	// make sure we clean up after ourselves if we return early
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	q := queue.NewLocalUnordered(a.numWorkers)

	if err := q.Start(ctx); err != nil {
		return errors.Wrap(err, "problem starting workers")
	}

	// begin "real" work
	start := time.Now()

	if err := a.addTests(q); err != nil {
		return errors.Wrap(err, "problem processing checks from suites")
	}

	if err := a.addSuites(q); err != nil {
		return errors.Wrap(err, "problem processing checks from suites")
	}

	stats := q.Stats()
	grip.Noticef("registered %d jobs, running checks now", stats.Total)
	q.Wait()

	grip.Noticef("checks complete in [num=%d, runtime=%s] ", stats.Total, time.Since(start))
	if err := a.out.ProduceResults(q); err != nil {
		return errors.Wrap(err, "problems encountered during tests")
	}

	return nil
}
