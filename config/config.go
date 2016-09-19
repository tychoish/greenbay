package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/greenbay"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"gopkg.in/yaml.v2"
)

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

	check, ok := testJob.(grenbay.Checker)
	if !ok {
		return nil, errors.Errorf("job %s does not implement Checker interface", t.Name)
	}

	check.SetID(t.Name)
	check.SetSuites(t.Suites)
	return check, nil
}

func newTestConfig() *GreenbayTestConfig {
	return &GreenbayTestConfig{
		tests:  make(map[string]amboy.Job),
		suites: make(map[string][]string),
	}
}

func ReadConfig(fn string) (*GreenbayTestConfig, error) {
	c := newTestConfig()

	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, errors.Wrap(err, "problem reading greenbay config file")
	}

	intermediateOut := make(map[string]interface{})
	err = yaml.Unmarshal(data, intermediateOut)
	if err != nil {
		return nil, errors.Wrap(err, "problem parsing yaml config")
	}

	jsonOut, err := json.Marshal(intermediateOut)
	if err != nil {
		return nil, errors.Wrap(err, "problem converting yaml to intermediate json")
	}

	err = json.Unmarshal(jsonOut, c)
	if err != nil {
		return nil, errors.Wrap(err, "problem converting yaml to document")
	}

	catcher := grip.NewCatcher()
	for _, msg := range c.RawTests {
		for _, suite := range msg.Suites {
			if _, ok := c.Suites[suite]; !ok {
				c.Suites[suite] = []string{}
			}

			c.Suites[suite] = append(c.Suites[suite], msg.Name)
		}

		testJob, err := msg.getJob()
		if err != nil {
			catcher.Add(err)
			continue
		}

		if _, ok := c.Tests[msg.Name]; ok {
			m := fmt.Sprintf("two tests named %s in config file %s", msg.Name, fn)
			grip.Alert(m)
			catcher.Add(errors.New(m))
			continue
		}

		c.Tests[msg.Name] = testJob
	}

	if catcher.HasErrors() {
		return nil, catcher.Resolve()
	}

	return c, nil
}

// func (c *GreenbayTestConfig)
