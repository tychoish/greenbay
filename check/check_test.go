package check

import (
	"fmt"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/greenbay"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CheckSuite struct {
	name    string
	factory registry.JobFactory
	check   greenbay.Checker
	require *require.Assertions
	suite.Suite
}

// Test constructors. For every new check, you should register a new
// version of the suite, specifying a different "name" value.

func TestMockCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "mock-check"
	suite.Run(t, s)
}

func TestShellOperationNoErrorCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "shell-operation"
	suite.Run(t, s)
}

func TestShellOperationErrorCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "shell-operation-error"
	suite.Run(t, s)
}

func TestShellGroupOperationAllCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "all-commands"
	suite.Run(t, s)
}

func TestShellGroupOperationAnyCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "any-command"
	suite.Run(t, s)
}

func TestShellGroupOperationOneCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "one-command"
	suite.Run(t, s)
}

func TestShellGroupOperationNoneCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "no-commands"
	suite.Run(t, s)
}

func TestFileExistsCheck(t *testing.T) {
	s := new(CheckSuite)
	s.name = "file-exists"
	suite.Run(t, s)
}

func TestFileDoesNotExistsCheck(t *testing.T) {
	s := new(CheckSuite)
	s.name = "file-does-not-exist"
	suite.Run(t, s)
}

func TestFileGroupExistsAllCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "all-files"
	suite.Run(t, s)
}

func TestFileGroupExistsAnyCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "any-file"
	suite.Run(t, s)
}

func TestFileGroupExistsOneCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "one-file"
	suite.Run(t, s)
}

func TestFileGroupExistsNoneCheckSuite(t *testing.T) {
	s := new(CheckSuite)
	s.name = "no-files"
	suite.Run(t, s)
}

// Test Fixtures

func (s *CheckSuite) SetupSuite() {
	s.require = s.Require()
	factory, err := registry.GetJobFactory(s.name)
	s.NoError(err)

	s.factory = factory
}

func (s *CheckSuite) SetupTest() {
	s.require.NotNil(s.factory)
	s.check = s.factory().(greenbay.Checker)
	s.require.NotNil(s.check)
}

// Test Cases

func (s *CheckSuite) TestCheckImplementsRequiredInterface() {
	s.Implements((*amboy.Job)(nil), s.check)
	s.Implements((*greenbay.Checker)(nil), s.check)
}

func (s *CheckSuite) TestInitialStateHasCorrectDefaults() {
	output := s.check.Output()
	s.False(output.Completed)
	s.False(output.Passed)
	s.False(s.check.Completed())
	s.NoError(s.check.Error())
	s.Equal("", output.Error)
	s.Equal(s.name, output.Check)
	s.Equal(s.name, s.check.Type().Name)
}

func (s *CheckSuite) TestRunningTestsHasImpact() {
	output := s.check.Output()
	s.False(output.Completed)
	s.False(s.check.Completed())
	s.False(output.Passed)

	s.check.Run()

	output = s.check.Output()
	s.True(output.Completed)
	s.True(s.check.Completed())
}

func (s *CheckSuite) TestFailedChecksShouldReturnErrors() {
	s.check.Run()
	output := s.check.Output()
	s.True(s.check.Completed())

	err := s.check.Error()

	msg := fmt.Sprintf("%T: %+v", s.check, output)
	if output.Passed {
		s.NoError(err, msg)
	} else {
		s.Error(err, msg)
	}
}
