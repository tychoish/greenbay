package operations

import (
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/rest"
	"github.com/mongodb/greenbay/config"
	"github.com/mongodb/greenbay/output"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"golang.org/x/net/context"
)

type GreenbayClient struct {
	Conf   *config.GreenbayTestConfig
	Output *output.Options
	client *rest.Client
	Tests  []string
	Suites []string
}

func NewClient(confPath, host string, port int, outFn, format string, quiet bool, suite, tests []string) (*GreenbayClient, error) {
	conf, err := config.ReadConfig(confPath)
	if err != nil {
		return nil, errors.Wrap(err, "problem parsing config file")
	}

	out, err := output.NewOptions(outFn, format, quiet)
	if err != nil {
		return nil, errors.Wrap(err, "problem generating output definition")
	}

	c, err := rest.NewClient(host, port, "")
	if err != nil {
		return nil, errors.Wrap(err, "problem constructing amboy rest client")
	}

	client := &GreenbayClient{
		client: c,
		Conf:   conf,
		Output: out,
		Tests:  tests,
		Suites: suite,
	}

	return client, nil
}

func (c *GreenbayClient) Run(ctx context.Context) error {
	if c.Conf == nil || c.Output == nil {
		return errors.New("GreenbayApp is not correctly constructed:" +
			"system and output configuration must be specified.")
	}

	// make sure we clean up after ourselves if we return early
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// begin "real" work
	start := time.Now()
	catcher := grip.NewCatcher()
	ids := []string{}

	for check := range c.Conf.GetAllTests(c.Tests, c.Suites) {
		if check.Err != nil {
			catcher.Add(check.Err)
			continue
		}
		id, err := c.client.SubmitJob(ctx, check.Job)
		if err != nil {
			catcher.Add(err)
			continue
		}
		ids = append(ids, id)
	}

	if catcher.HasErrors() {
		return errors.Wrap(catcher.Resolve(), "problem collecting and submitting jobs")
	}

	// TODO: make the Ids avalible in the app to retry waiting.

	// wait all will block for 20 seconds (by default, we could
	// timeout the context if needed to have control over that);
	// our main risk is that another client will submit jobs at
	// the same time, and we'll end up waiting for each other's
	// jobs. We could become much more clever here.
	//
	// However, the assumption is that 20 seconds will be enough
	// given that these jobs should complete faster than that.
	if !c.client.WaitAll(ctx) {
		return errors.New("timed out waiting for jobs to complete")
	}

	jobs := make(chan amboy.Job, len(ids))
	for _, id := range ids {
		j, err := c.client.FetchJob(ctx, id)
		if err != nil {
			catcher.Add(err)
		}
		jobs <- j
	}

	if catcher.HasErrors() {
		return errors.Wrap(catcher.Resolve(), "problem collecting and submitting jobs")
	}

	grip.Noticef("checks complete in [num=%d, runtime=%s] ", len(ids), time.Since(start))
	return c.Output.CollectResults(jobs)
}
