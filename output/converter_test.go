package output

import (
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/greenbay"
	"github.com/mongodb/greenbay/check"
	"github.com/stretchr/testify/assert"
)

type mockCheck struct {
	check.Base
}

func (c *mockCheck) Run() {
}

func TestConverter(t *testing.T) {
	assert := assert.New(t)

	j := job.NewShellJob("echo foo", "")
	assert.NotNil(j)
	c, err := convert(j)
	assert.Error(err)
	assert.Nil(c)

	mc := &mockCheck{}
	assert.Implements((*amboy.Job)(nil), mc)
	assert.Implements((*greenbay.Checker)(nil), mc)

	c, err = convert(mc)
	assert.NoError(err)
	assert.NotNil(c)
}
