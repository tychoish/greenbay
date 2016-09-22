package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"runtime"
	"sync"

	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

// GreenbayTestConfig defines the
type GreenbayTestConfig struct {
	Options struct {
		ContineOnError bool   `bson:"continue_on_error" json:"continue_on_error" yaml:"continue_on_error"`
		ReportFormat   string `bson:"report_format" json:"report_format" yaml:"report_format"`
		Jobs           int    `bson:"jobs" json:"jobs" yaml:"jobs"`
	} `bson:"options" json:"options" yaml:"options"`
	RawTests []rawTest `bson:"tests" json:"tests" yaml:"tests"`
	tests    map[string]amboy.Job
	suites   map[string][]string
	mutex    sync.RWMutex
}

func newTestConfig() *GreenbayTestConfig {
	conf := &GreenbayTestConfig{
		tests:  make(map[string]amboy.Job),
		suites: make(map[string][]string),
	}
	conf.Options.Jobs = runtime.NumCPU()

	return conf
}

// ReadConfig takes a path name to a configuration file (yaml
// formatted,) and returns a configuration format.
func ReadConfig(fn string) (*GreenbayTestConfig, error) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, errors.Wrapf(err, "problem reading greenbay config file: %s", fn)
	}

	format, err := getFormat(fn)
	if err != nil {
		return nil, errors.Wrapf(err, "problem determining format of file %s", fn)
	}

	// Parse data:
	data, err = getJSONFormatedConfig(format, data)
	if err != nil {
		return nil, errors.Wrap(err, "problem parsing config from file %s", fn)
	}

	c := newTestConfig()
	// we don't take the lock here because this function doesn't
	// spawn threads, and nothing else can see the object we're
	// building. If either of those things change we should take
	// the lock here.

	// now we have a json formated byte slice in data and we can
	// unmarshal it as we want.
	err = json.Unmarshal(data, c)
	if err != nil {
		return nil, errors.Wrapf(err, "problem parsing config: %s", fn)
	}

	err = c.parseTests()
	if err != nil {
		return nil, errors.Wrapf(err, "problem parsing tests from file: %s", fn)
	}
	return c, nil
}

func (c *GreenbayTestConfig) parseTests() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	catcher := grip.NewCatcher()
	for _, msg := range c.RawTests {
		for _, suite := range msg.Suites {
			if _, ok := c.suites[suite]; !ok {
				c.suites[suite] = []string{}
			}

			c.suites[suite] = append(c.suites[suite], msg.Name)
		}

		testJob, err := msg.getJob()
		if err != nil {
			catcher.Add(err)
			continue
		}

		if _, ok := c.tests[msg.Name]; ok {
			m := fmt.Sprintf("two tests named %s", msg.Name)
			grip.Alert(m)
			catcher.Add(errors.New(m))
			continue
		}

		c.tests[msg.Name] = testJob
	}

	return catcher.Resolve()
}

// GetTests takes the name of a suite and then produces a sequence of
// jobs that are part of that suite.
func (c *GreenbayTestConfig) GetTests(suite string) (<-chan amboy.Job, error) {
	c.mutex.RLock()
	tests, ok := c.suites[suite]
	c.mutex.RUnlock()

	if !ok {
		return nil, errors.Errorf("no suite named '%s' exists,", suite)
	}

	output := make(chan amboy.Job)
	go func() {
		c.mutex.RLock()
		defer c.mutex.RUnlock()

		for _, test := range tests {
			j, ok := c.tests[test]
			if !ok {
				grip.Warningf("test named %s doesn't exist, but should", test)
				continue
			}

			output <- j
		}

		close(output)
	}()

	return output, nil
}
