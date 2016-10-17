package check

import (
	"fmt"

	"github.com/mongodb/amboy"
	"github.com/pkg/errors"
)

func registerPackageGroupChecks() {
	packageGroupFactoryFactory := func(name string, gr GroupRequirements, checker packageChecker) func() amboy.Job {
		return func() amboy.Job {
			gr.Name = name
			return &packageGroup{
				Base:         NewBase(name, 0),
				Requirements: gr,
				checker:      checker,
			}
		}
	}

	for pkg, checker := range packageCheckerRegistry {
		for group, requirements := range groupRequirementRegistry {
			name := fmt.Sprintf("%s-group-%s", pkg, group)
			regisry.AddJobType(name, packageGroupFactoryFactory(name, requirements, checker))
		}
	}
}

type packageGroup struct {
	Packages     []string          `bson:"packages" json:"packages" yaml:"packages"`
	Requirements GroupRequirements `bson:"requirements" json:"requirements" yaml:"requirements"`
	*Base        `bson:"metadata" json:"metadata" yaml:"metadata"`
	checker      packageChecker
}

func (c *packageGroup) Run() {
	c.startTask()
	defer c.markComplete()

	if err := c.Requirements.Validate(); err != nil {
		c.setState(false)
		c.addError(err)
		return
	}

	if len(c.Packages) == 0 {
		c.setState(false)
		c.addError(errors.Errorf("no packages for '%s' (%s) check",
			c.ID(), c.Name()))
		return
	}

	var installed []string
	var missing []string
	var messages []string

	for _, pkg := range c.Packages {
		exists, msg, err := c.checker(pkg)
		if exists {
			installed = append(installed, pkg)
		} else {
			missing = append(missing, pkg)
		}
		c.addError(err)
		messages = append(messages, msg)
	}

	result, err := c.Requirements.GetResults(len(installed), len(missing))
	c.setState(result)
	c.addError(err)

	if !result {
		c.setMessage(messages)
		c.addError(errors.New("group of packages does not satisfy check requirements"))
	}
}
