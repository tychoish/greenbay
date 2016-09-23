package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mongodb/amboy/job"
	"github.com/mongodb/greenbay/check"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	tempDir  string
	confFile string
	conf     *GreenbayTestConfig
	require  *require.Assertions
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (s *ConfigSuite) SetupSuite() {
	s.require = s.Require()

	dir, err := ioutil.TempDir("", uuid.NewV4().String())
	s.require.NoError(err)
	s.tempDir = dir

	conf := newTestConfig()
	num := 30

	jsonJob, err := json.Marshal(&mockShellCheck{
		shell: job.NewShellJob("echo foo", ""),
		Base:  check.Base{},
	})
	s.NoError(err)

	for i := 0; i < num; i++ {
		conf.RawTests = append(conf.RawTests,
			rawTest{
				Name:      fmt.Sprintf("check-working-shell-%d", i),
				Suites:    []string{"one", "two"},
				RawArgs:   jsonJob,
				Operation: mockShellCheckName,
			})
	}

	dump, err := json.Marshal(conf)
	s.require.NoError(err)
	fn := filepath.Join(dir, "conf.json")
	s.confFile = fn
	err = ioutil.WriteFile(fn, dump, 0644)
	s.require.NoError(err)
}

func (s *ConfigSuite) SetupTest() {
	s.conf = newTestConfig()
}

func (s *ConfigSuite) TearDownSuite() {
	s.require.NoError(os.RemoveAll(s.tempDir))
}

func (s *ConfigSuite) TestTemporyFileConfigIsCorrect() {
	conf, err := ReadConfig(s.confFile)

	s.NoError(err)
	s.NotNil(conf)
}

func (s *ConfigSuite) TestInitializedConfObjectHasCorrectInitialValues() {
	s.NotNil(s.conf.tests)
	s.NotNil(s.conf.suites)

	s.Len(s.conf.tests, 0)
	s.Len(s.conf.suites, 0)
	s.Len(s.conf.RawTests, 0)

	s.Equal(runtime.NumCPU(), s.conf.Options.Jobs)
}

func (s *ConfigSuite) TestAddingDuplicateJobsToConfig() {
	jsonJob, err := json.Marshal(&mockShellCheck{
		shell: job.NewShellJob("echo foo", ""),
		Base:  check.Base{},
	})
	s.NoError(err)

	num := 3
	for i := 0; i < num; i++ {
		s.conf.RawTests = append(s.conf.RawTests,
			rawTest{
				Name:      "check-working-shell",
				Suites:    []string{"one", "two"},
				RawArgs:   jsonJob,
				Operation: mockShellCheckName,
			})
	}

	s.Len(s.conf.tests, 0)
	s.Error(s.conf.parseTests())

	s.Len(s.conf.tests, 1)
	s.Len(s.conf.suites, num-1)
	s.Len(s.conf.RawTests, num)
}

func (s *ConfigSuite) TestAddingInvalidDocumentsToConfig() {
	s.conf.RawTests = append(s.conf.RawTests,
		rawTest{
			Name:      "foo",
			Suites:    []string{"one", "two"},
			RawArgs:   []byte(`{a:1}`),
			Operation: "bar",
		})

	s.Len(s.conf.tests, 0)
	s.Len(s.conf.RawTests, 1)
	s.Error(s.conf.parseTests())
	s.Len(s.conf.tests, 0)
}

func (s *ConfigSuite) TestReadingConfigFromFileDoesntExist() {
	conf, err := ReadConfig(filepath.Join(s.tempDir, "foo", filepath.Base(s.confFile)))
	s.Error(err)
	s.Nil(conf)
}

func (s *ConfigSuite) TestReadConfigWithInvalidFormat() {
	fn := s.confFile + ".foo"
	err := os.Link(s.confFile, fn)
	s.NoError(err)

	conf, err := ReadConfig(fn)

	s.Error(err)
	s.Nil(conf)
}

func (s *ConfigSuite) TestGetterObject() {
	conf, err := ReadConfig(s.confFile)

	s.NoError(err)
	s.NotNil(conf)

	tests, err := conf.GetTests("one")
	s.NoError(err)

	for t := range tests {
		s.NotNil(t)
	}
}

func (s *ConfigSuite) TestGetterGeneratorWithInvalidSuite() {
	tests, err := s.conf.GetTests("DOES-NOT-EXIST")
	s.Error(err)
	s.Nil(tests)

}
