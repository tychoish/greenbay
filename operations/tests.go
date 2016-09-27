package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/greenbay/config"
	"github.com/mongodb/greenbay/output"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"github.com/urfave/cli"
)

// RunChecks returns the urfave/cli.Command object for running specific
// greenbay tests by name.
func RunChecks() cli.Command {
	return cli.Command{
		Name:  "check",
		Usage: "run greenbay checks",
		Flags: checkFlags(
			cli.StringSliceFlag{
				Name:  "test",
				Usage: "specify a check, by name",
				Value: &cli.StringSlice{"base"},
			}),
		Action: func(c *cli.Context) error {
			conf, out, err := prepTests(c.String("conf"), c.String("output"), c.String("format"), c.Bool("quiet"))

			if err != nil {
				return errors.Wrap(err, "problem prepping to run tests")
			}

			jobs := conf.TestsByName(c.StringSlice("test")...)

			return runTests(jobs, c.Int("jobs"), out, conf)
		},
	}
}

// RunSuites returns the urfave/cli.Command object for running suites
// of greenbay tests.
func RunSuites() cli.Command {
	return cli.Command{
		Name:  "run",
		Usage: "run greenbay suites",
		Flags: checkFlags(
			cli.StringSliceFlag{
				Name:  "suite",
				Usage: "specify a suite or suites, by name",
				Value: &cli.StringSlice{"all"},
			}),
		Action: func(c *cli.Context) error {
			conf, out, err := prepTests(c.String("conf"), c.String("output"), c.String("format"), c.Bool("quiet"))
			if err != nil {
				return errors.Wrap(err, "problem prepping to run tests")
			}

			jobs := conf.TestsForSuites(c.StringSlice("suite")...)

			return runTests(jobs, c.Int("jobs"), out, conf)
		},
	}
}

func checkFlags(args ...cli.Flag) []cli.Flag {
	defaultNumJobs := runtime.NumCPU()
	cwd, _ := os.Getwd()
	configPath := filepath.Join(cwd, "greenbay.yaml")

	flags := []cli.Flag{
		cli.IntFlag{
			Name: "jobs",
			Usage: fmt.Sprintf("specify the number of parallel tests to run. (Default %s)",
				defaultNumJobs),
			Value: defaultNumJobs,
		},
		cli.StringFlag{
			Name: "conf",
			Usage: fmt.Sprintln("path to config file. '.json', '.yaml', and '.yml' extensions ",
				"supported.", "Default path:", configPath),
			Value: configPath,
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "path of file to write output too. Defaults to *not* writing output to a file",
			Value: "",
		},
		cli.BoolFlag{
			Name:  "quiet",
			Usage: "specify to disable printed (standard output) results",
		},
		cli.StringFlag{
			Name: "format",
			Usage: fmt.Sprintln("Selects the output format, defautls to a format that mirrors gotest,",
				"but also supports evergreen's results format.",
				"Use either 'gotest' (default) or 'results'."),
			Value: "gotest",
		},
	}

	flags = append(flags, args...)

	return flags
}

////////////////////////////////////////////////////////////////////////
//
// run tests
//
////////////////////////////////////////////////////////////////////////

func prepTests(confPath, output, format string, quiet bool) (*config.GreenbayTestConfig, *outputSpec, error) {
	conf, err := config.ReadConfig(confPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "problem parsing config file")
	}

	out, err := getOutputSpec(output, format, quiet)
	if err != nil {
		return nil, nil, errors.Wrap(err, "problem generating output definition")
	}

	return conf, out, nil
}

func runTests(jobs <-chan config.JobWithError, numWorkers int, out *outputSpec, conf *config.GreenbayTestConfig) error {
	q := queue.NewLocalUnordered(numWorkers)
	catcher := grip.NewCatcher()

	start := time.Now()
	for check := range jobs {
		if check.Err != nil {
			catcher.Add(check.Err)
			continue
		}

		catcher.Add(q.Put(check.Job))
	}

	if catcher.HasErrors() {
		return errors.Wrap(catcher.Resolve(), "error adding checks to queue")
	}

	stats := q.Stats()
	grip.Infof("running #%s checks...", stats.Total)
	q.Wait()
	grip.Infof("checks complete in [num=%d, runtime=%s] ", stats.Total, time.Since(start))

	if err := out.produce(q); err != nil {
		return errors.Wrap(err, "problems encountered during tests")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////
//
// output generation
//
////////////////////////////////////////////////////////////////////////

type outputSpec struct {
	writeFile   bool
	writeStdOut bool
	fn          string
	format      string
}

func (o *outputSpec) produce(queue amboy.Queue) error {
	// Get results generator
	factory, ok := output.GetResultsFactory(o.format)
	if !ok {
		return errors.Errorf("could not find results output type registered for '%s'",
			o.format)
	}

	r := factory()

	if err := r.Populate(queue); err != nil {
		return errors.Wrap(err, "problem generating results content")
	}

	// Actually write output to respective streems
	catcher := grip.NewCatcher()

	if o.writeStdOut {
		catcher.Add(r.Print())
	}

	if o.writeFile {
		catcher.Add(r.ToFile(o.fn))
	}

	return catcher.Resolve()
}

func getOutputSpec(fn, format string, quiet bool) (*outputSpec, error) {
	o := &outputSpec{}
	if !quiet {
		o.writeStdOut = true
	}

	if fn != "" {
		o.writeFile = true
		o.fn = fn
	}

	if format == "gotest" || format == "results" {
		o.format = format
	} else {
		return nil, errors.Errorf("output format '%s' is not supported", format)
	}

	return o, nil
}
