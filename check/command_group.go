package check

import (
	"strings"

	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/greenbay"
	"github.com/pkg/errors"
)

func init() {
	var name string

	name = "all-commands"
	registry.AddJobType(name, func() amby.Job {
		return &shellGroup{
			Base: NewBase(name, 0),
			Requirements: GroupRequirements{
				Name: name,
				All:  true,
			},
		}
	})

}

type shellGroup struct {
	Commands     []*shellOperation `bson:"commands" json:"commands" yaml:"commands"`
	Requirements GroupRequirements `bson:"requirements" json:"requirements" yaml:"requirements"`
	*Base
}

func (c *shellGroup) Run() {
	c.startTask()
	defer c.markComplete()

	success := []*greenbay.CheckOutput{}
	failure := []*greenbay.CheckOutput{}

	for _, cmd := range c.Commands {
		cmd.Run()

		result := cmd.Output()
		if result.Passed {
			success = append(success, &result)
		} else {
			failure = append(failure, &result)
		}
	}

	result, err := c.gr.GroupSatisfiesRequirements(len(success), len(failure))
	c.setState(result)
	c.addError(err)

	if !result {
		var output []string
		var errs []string

		printableResults := []*greenbay.CheckOutput{}
		if c.gr.None {
			printableResults = success
		} else if c.gr.Any || c.gr.One {
			printableResults = success
			printableResults = append(printableResults, failure...)
		} else {
			printableResults = failure
		}

		for _, failedCmd := range printableResults {
			if failedCmd.Message != "" {
				output = append(output, failedCmd.Message)
			}

			if failedCmd.Error != "" {
				errs = append(errs, failedCmd.Error)
			}
		}

		c.setMessage(output)
		c.addError(errors.New(strings.Join(errs, "\n")))
	}
}
