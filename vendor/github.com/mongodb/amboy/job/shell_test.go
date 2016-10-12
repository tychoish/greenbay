package job

import (
	"strings"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ShellJobSuite collects tests of the generic shell command running
// amboy.Job implementation. The actual implementation of the command
// execution is straightforward, and so this test mostly checks the
// constructor and the environment variable construction.
type ShellJobSuite struct {
	job     *ShellJob
	require *require.Assertions
	suite.Suite
}

func TestShellJobSuite(t *testing.T) {
	suite.Run(t, new(ShellJobSuite))
}

func (s *ShellJobSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *ShellJobSuite) SetupTest() {
	s.job = NewShellJobInstance()
}

func (s *ShellJobSuite) TestShellJobProducesObjectsThatImplementJobInterface() {
	s.Implements((*amboy.Job)(nil), s.job)
	s.Implements((*amboy.Job)(nil), NewShellJobInstance())
}

func (s *ShellJobSuite) TestShellJobFactoryImplementsInterfaceWithCorrectTypeInfo() {
	sj := shellJobFactory()

	s.IsType(sj, s.job)
	s.Equal(sj.Type(), s.job.Type())

	s.Equal(sj.Type().Name, "shell")
	s.Equal(sj.Type().Version, 0)
}

func (s *ShellJobSuite) TestShellJobDefaultsToAlwaysDependency() {
	s.Equal(s.job.Dependency().Type().Name, "always")
}

func (s *ShellJobSuite) TestShellJobConstructorHasCreatesFileDependency() {
	job := NewShellJob("foo", "bar")
	s.Equal(job.Dependency().Type().Name, "create-file")
}

func (s *ShellJobSuite) TestShellDependencyAccessorReturnsPersistedValue() {
	s.Equal(s.job.Dependency(), s.job.dep)
}

func (s *ShellJobSuite) TestSetDependencyChangesDependencyStrategy() {
	s.Equal(s.job.Dependency().Type().Name, "always")
	s.job.SetDependency(dependency.NewCreatesFile("foo"))
	s.Equal(s.job.Dependency().Type().Name, "create-file")
}

func (s *ShellJobSuite) TestShellJobNameConstructedFromCommandNames() {
	job := NewShellJob("foo", "bar")
	s.Equal(job.ID(), job.Name)

	s.True(strings.HasPrefix(job.ID(), "foo"))

	job = NewShellJob("touch foo bar", "baz")
	s.True(strings.HasPrefix(job.ID(), "touch"))
}

func (s *ShellJobSuite) TestRunTrivialCommandReturnsWithoutError() {
	s.job = NewShellJob("true", "")

	s.False(s.job.Completed())
	s.job.Run()
	s.NoError(s.job.Error())
	s.True(s.job.Completed())
}

func (s *ShellJobSuite) TestRunWithErroneousCommandReturnsError() {
	s.job = NewShellJob("foo", "")

	s.False(s.job.Completed())
	s.job.Run()
	s.Error(s.job.Error())
	s.True(s.job.Completed())
}

func (s *ShellJobSuite) TestEnvironmentVariableIsPassedToCommand() {
	s.job = NewShellJob("env", "")
	s.job.Env["MSG"] = "foo"
	s.job.Run()
	s.NoError(s.job.Error())
	s.Equal("MSG=foo", s.job.Output)
}
