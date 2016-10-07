package check

import (
	"fmt"
	"os"
	"strings"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func init() {
	var name string

	name = "all-files"
	registry.AddJobType(name, func() amboy.Job {
		return &fileGroup{
			Base: NewBase(name, 0), // (name, version)
			Requirements: GroupRequirements{
				Name: name,
				All:  true,
			},
		}
	})

	name = "any-file"
	registry.AddJobType(name, func() amboy.Job {
		return &fileGroup{
			Base: NewBase(name, 0), // (name, version)
			Requirements: GroupRequirements{
				Name: name,
				Any:  true,
			},
		}
	})

	name = "one-file"
	registry.AddJobType(name, func() amboy.Job {
		return &fileGroup{
			Base: NewBase(name, 0), // (name, version)
			Requirements: GroupRequirements{
				Name: name,
				One:  true,
			},
		}
	})

	name = "no-files"
	registry.AddJobType(name, func() amboy.Job {
		return &fileGroup{
			Base: NewBase(name, 0), // (name, version)
			Requirements: GroupRequirements{
				Name: name,
				None: true,
			},
		}
	})
}

type fileGroup struct {
	FileNames    []string          `bson:"file_names" json:"file_names" yaml:"file_names"`
	Requirements GroupRequirements `bson:"requirements" json:"requirements" yaml:"requirements"`
	*Base
}

func (c *fileGroup) validate() bool {
	opts := []bool{c.allFiles, c.anyFile, c.oneFile}
	active := 0

	for _, opt := range opts {
		if opt {
			active++
		}
	}

	if active != 1 {
		c.addError(errors.Errorf("specified incorrect number of options for a '%s' check: "+
			"[all=%t, one=%t, any=%t, none=%t]", c.Name(),
			c.allFiles, c.oneFile, c.anyFile, c.noFiles))
		return false
	}

	if len(c.FileNames) < 1 {
		c.addError(errors.Errorf("no files specified for '%s' check", c.Name()))
		return false
	}

	return true
}

func (c *fileGroup) Run() {
	c.startTask()
	defer c.markComplete()

	if !c.validate() {
		c.setState(false)
		return
	}

	var extantFiles []string
	var missingFiles []string
	for _, fn := range c.FileNames {
		stat, err := os.Stat(fn)
		grip.Debugf("file '%s' stat: %+v", fn, stat)

		if os.IsNotExist(err) {
			missingFiles = append(missingFiles, fn)
			continue
		}

		extantFiles = append(extantFiles, fn)
	}

	msg := fmt.Sprintf("'%s' check. %d files exist, %d do not exist. "+
		"[existing=(%s), missing=(%s)]", c.Name(), len(extantFiles), len(missingFiles),
		strings.Join(extantFiles, ", "), strings.Join(missingFiles, ", "))
	grip.Debug(msg)

	success := c.getResults(extantFiles, missingFiles)
	c.setState(success)

	if !success {
		c.setMessage(msg)
	}
}
