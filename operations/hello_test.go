package operations

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

// CommandsSuite provide a group of tests of the entry points and
// command wrappers for command-line interface to curator.
type CommandsSuite struct {
	suite.Suite
}

func TestCommandSuite(t *testing.T) {
	suite.Run(t, new(CommandsSuite))
}

func (s *CommandsSuite) TestHelloWorldOperationViaDirectCall() {
	cmd := exec.Command("../greenbay", "hello")
	output, err := cmd.CombinedOutput()
	s.NoError(err)

	// check the results.
	s.Equal("hello world!", strings.Trim(string(output), "\n "))
}

func (s *CommandsSuite) TestHelloCommandObjectAttributes() {
	cmd := HelloWorld()
	s.Equal(cmd.Name, "hello")
	s.Contains(cmd.Aliases, "hi")
	s.Contains(cmd.Aliases, "hello-world")
	s.Len(cmd.Flags, 0)
	action, ok := cmd.Action.(func(c *cli.Context) error)
	s.True(ok)
	s.Nil(action(nil))
}

func (s *CommandsSuite) TestHelloWorldFunctionReturnsHelloWorldString() {
	s.Equal(helloWorld(), "hello world!")
}
