package check

import (
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/greenbay"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func init() {
	var name string

	commandGroupFactoryFactory := func(name string, gr GroupRequirements) func() amboy.Job {
		gr.Name = name
		return func() amboy.Job {
			return &shellGroup{
				Base:         NewBase(name, 0),
				Requirements: gr,
			}
		}
	}

	name = "all-commands"
	registry.AddJobType(name, commandGroupFactoryFactory(name, GroupRequirements{All: true}))

	name = "any-command"
	registry.AddJobType(name, commandGroupFactoryFactory(name, GroupRequirements{Any: true}))

	name = "one-command"
	registry.AddJobType(name, commandGroupFactoryFactory(name, GroupRequirements{One: true}))

	name = "no-commands"
	registry.AddJobType(name, commandGroupFactoryFactory(name, GroupRequirements{None: true}))
}

type shellGroup struct {
	Commands     []*shellOperation `bson:"commands" json:"commands" yaml:"commands"`
	Requirements GroupRequirements `bson:"requirements" json:"requirements" yaml:"requirements"`
	*Base
}

func (c *shellGroup) Run() {
	c.startTask()
	defer c.markComplete()

	if err := c.Requirements.Validate(); err != nil {
		c.setState(false)
		c.addError(err)
		return
	}

	if len(c.Commands) == 0 {
		c.setState(false)
		c.addError(errors.Errorf("no files specified for '%s' (%s) check",
			c.ID(), c.Name()))
		return
	}

	var success []*greenbay.CheckOutput
	var failure []*greenbay.CheckOutput

	for _, cmd := range c.Commands {
		cmd.Run()

		result := cmd.Output()
		if result.Passed {
			success = append(success, &result)
		} else {
			failure = append(failure, &result)
		}
	}

	result, err := c.Requirements.GetResults(len(success), len(failure))
	c.setState(result)
	c.addError(err)
	grip.Debugf("task '%s' recieved result %t, with %d successes and %d failures",
		c.ID(), result, len(success), len(failure))

	if !result {
		var output []string
		var errs []string

		printableResults := []*greenbay.CheckOutput{}
		if c.Requirements.None {
			printableResults = success
		} else if c.Requirements.Any || c.Requirements.One {
			printableResults = success
			printableResults = append(printableResults, failure...)
		} else {
			printableResults = failure
		}

		for _, cmd := range printableResults {
			if cmd.Message != "" {
				output = append(output, cmd.Message)
			}

			if cmd.Error != "" {
				errs = append(errs, cmd.Error)
			}
		}

		c.setMessage(output)
		c.addError(errors.New(strings.Join(errs, "\n")))
	}
}
