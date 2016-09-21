package config

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	conf    *GreenbayTestConfig
	require *require.Assertions
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (s *ConfigSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *ConfigSuite) SetupTest() {
	s.conf = newTestConfig()
}

func (s *ConfigSuite) TestInitializedConfObjectHasCorrectInitialValues() {
	s.NotNil(s.conf.tests)
	s.NotNil(s.conf.suites)

	s.Len(s.conf.tests, 0)
	s.Len(s.conf.suites, 0)
	s.Len(s.conf.RawTests, 0)

	s.Equal(runtime.NumCPU(), s.conf.Options.Jobs)
}
