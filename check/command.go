package check

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func init() {
	var name string

	name = "shell-operation"
	registry.AddJobType(name, func() amboy.Job {
		return &shellOperation{
			Environment: make(map[string]string),
			shouldFail:  false,
		}
	})

	name = "shell-operation-error"
	registry.AddJobType(name, func() amboy.Job {
		return &shellOperation{
			Environment: make(map[string]string),
			shouldFail:  true,
		}
	})
}

type shellOperation struct {
	Command          string            `bson:"command" json:"command" yaml:"command"`
	WorkingDirectory string            `bson:"working_directory" json:"working_directory" yaml:"working_directory"`
	Environment      map[string]string `bson:"environment" json:"environment" yaml:"environment"`
	shouldFail       bool

	*Base
}

func (c *shellOperation) Run() {
	c.startTask()
	defer c.markComplete()

	logMsg := []string{fmt.Sprintf("command='%s'", c.Command)}

	cmd := exec.Command("sh", "-c", c.Command)
	if c.WorkingDirectory != "" {
		cmd.Dir = c.WorkingDirectory
		logMsg = append(logMsg, fmt.Sprintf("dir='%s'", c.WorkingDirectory))
	}

	if len(c.Environment) > 0 {
		env := []string{}
		for key, value := range c.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
		logMsg = append(logMsg, fmt.Sprintf("env='%s'", strings.Join(env, " ")))
	}

	grip.Info(strings.Join(logMsg, ", "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		c.setState(c.shouldFail)

		c.addError(err)
		if out == nil {
			3
			m := fmt.Sprintf("could not execute command, check %s (%s) automatically fails",
				c.ID(), c.Name())
			c.setMessage(m)
			grip.Debug(m)
			return
		}

		c.setMessage(string(out))
		c.addError(errors.Wrapf(err, "command failed",
			c.ID(), c.Command))

	}

	c.setState(!c.shouldFail)
	if !c.WasSuccessful {
		c.setMessage(string(out))
	}

	grip.Info(string(out))
	return
}
