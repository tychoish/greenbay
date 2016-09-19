package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	conf    GreenbayTestConfig
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

}

func (s *ConfigSuite) Test() {
	s.Test(true)
	s.require.False(false)
}
