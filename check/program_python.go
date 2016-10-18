package check

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

func pythonCompilerFactoryTable() map[string]compilerFactory {
	return map[string]compilerFactory{}
}

type pythonCompiler struct {
	bin string
}

func (c *pythonCompiler) Validate() error {
	if c.bin == "" {
		return errors.New("no python interpreter")
	}

	if _, err := os.Stat(c.bin); !os.IsNotExist(err) {
		return errors.Errorf("python interpreter '%s' does not exist", c.bin)
	}

	return nil
}

func (c *pythonCompiler) Compile(testBody string, _ ...string) error {
	_, sourceName, err := writeTestBody(testBody, "py")
	if err != nil {
		return errors.Wrap(err, "problem writing test")
	}

	defer grip.Warning(os.Remove(sourceName))

	cmd := exec.Command(c.bin, sourceName)
	grip.Infof("running python script with command: %s", strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "problem running test script %s: %s", sourceName,
			string(output))
	}

	return nil
}

func (c *pythonCompiler) CompileAndRun(testBody string, _ ...string) (string, error) {
	_, sourceName, err := writeTestBody(testBody, "py")
	if err != nil {
		return "", errors.Wrap(err, "problem writing test")
	}

	cmd := exec.Command(c.bin, sourceName)
	grip.Infof("running python script with command: %s", strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	output = string(out)
	if err != nil {
		return output, errors.Wrapf(err, "problem running test script %s", sourceName)
	}

	output = strings.Trim(output, "\t\n ")
	return output, nil
}
