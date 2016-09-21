package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"
	"sync"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/greenbay"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"gopkg.in/yaml.v2"
)

// GreenbayTestConfig defines the output
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

type rawTest struct {
	Name      string          `bson:"name" json:"name" yaml:"name"`
	Suites    []string        `bson:"suites" json:"suites" yaml:"suites"`
	Operation string          `bson:"type" json:"type" yaml:"type"`
	RawArgs   json.RawMessage `bson:"args" json:"args" yaml:"args"`
}

func (t *rawTest) getJob() (greenbay.Checker, error) {
	factory, err := registry.GetJobFactory(t.Operation)
	if err != nil {
		return nil, errors.Wrapf(err, "no test job named %s defined,",
			t.Operation)
	}

	testJob := factory()
	if err = json.Unmarshal(t.RawArgs, testJob); err != nil {
		return nil, errors.Wrapf(err, "problem parsing argument for job %s (%s)",
			t.Name, t.Operation)
	}

	check, ok := testJob.(greenbay.Checker)
	if !ok {
		return nil, errors.Errorf("job %s does not implement Checker interface", t.Name)
	}

	check.SetID(t.Name)
	check.SetSuites(t.Suites)
	return check, nil
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
	c := newTestConfig()
	// we don't take the lock here because this function doesn't
	// spawn threads, and nothing else can see the object we're
	// building. If either of those things change we should take
	// the lock here.

	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, errors.Wrapf(err, "problem reading greenbay config file: %s", fn)
	}

	if strings.HasSuffix(fn, ".yaml") || strings.HasSuffix(fn, ".yml") {
		// the yaml package does not include a way to do the kind of
		// delayed parsing that encoding/json permits, so we cycle
		// into a map and then through the JSON parser itself.
		intermediateOut := make(map[string]interface{})
		err = yaml.Unmarshal(data, intermediateOut)
		if err != nil {
			return nil, errors.Wrap(err, "problem parsing yaml config")
		}

		data, err = json.Marshal(intermediateOut)
		if err != nil {
			return nil, errors.Wrap(err, "problem converting yaml to intermediate json")
		}
	} else if !strings.HasSuffix(fn, ".json") {
		return nil, errors.Errorf("greenbay does not support configuration format for file %s", fn)
	}

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
