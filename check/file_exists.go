package check

import (
	"fmt"
	"os"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/tychoish/grip"
)

func init() {
	name := "file-exists"
	registry.AddJobType(name, func() amboy.Job {
		return &fileExistance{
			ShouldExist: true,
			Base:        NewBase(name, 0), // (name, version)
		}
	})

	name = "file-does-not-exist"
	registry.AddJobType(name, func() amboy.Job {
		return &fileExistance{
			ShouldExist: false,
			Base:        NewBase(name, 0), // (name, version)
		}
	})
}

type fileExistance struct {
	FileName    string `bson:"name" json:"name" yaml:"name"`
	ShouldExist bool   `bson:"should_exist" json:"should_exist" yaml:"should_exist"`
	*Base
}

func (c *fileExistance) Run() {
	c.startTask()
	defer c.markComplete()

	var fileExists bool
	var verb string

	stat, err := os.Stat(c.FileName)
	fileExists = os.IsNotExist(err)

	c.setState(fileExists == c.ShouldExist)

	if c.ShouldExist {
		verb = "should"
	} else {
		verb = "should not"
	}

	m := fmt.Sprintf("file '%s' %s exist. stats=%+v", c.Name, verb, stat)
	grip.Debug(m)
	c.setMessage(m)
}
