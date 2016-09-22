package config

import (
	"encoding/json"

	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
)

type rawTest struct {
	Name      string          `bson:"name" json:"name" yaml:"name"`
	Suites    []string        `bson:"suites" json:"suites" yaml:"suites"`
	Operation string          `bson:"type" json:"type" yaml:"type"`
	RawArgs   json.RawMessage `bson:"args" json:"args" yaml:"args"`
}

func (t *rawTest) getJob() (greenbay.Checker, error) {
	check, err := t.getCheck()
	if err != nil {
		return nil, errors.Wrap(err, "problem determining job type")
	}

	if err = json.Unmarshal(t.RawArgs, check); err != nil {
		return nil, errors.Wrapf(err, "problem parsing argument for job %s (%s)",
			t.Name, t.Operation)
	}

	check.SetID(t.Name)
	check.SetSuites(t.Suites)

	return check, nil
}

func (t *rawTest) getCheck() (greenbay.Checker, error) {
	factory, err := registry.GetJobFactory(t.Operation)
	if err != nil {
		return nil, errors.Wrapf(err, "no test job named %s defined,",
			t.Operation)
	}

	testJob := factory()

	check, ok := testJob.(greenbay.Checker)
	if !ok {
		return nil, errors.Errorf("job %s does not implement Checker interface", t.Name)
	}

	return check, nil
}
