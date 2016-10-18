package check

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageCheckerFactory(t *testing.T) {
	assert := assert.New(t)

	alwaysPasses := packageCheckerFactory([]string{"echo", "passing-test"})
	alwaysFails := packageCheckerFactory([]string{"exit"})

	for _, name := range []string{"foo", "bar", "1", "true"} {
		result, message, err := alwaysPasses(name)
		assert.True(result)
		assert.Equal(message, strings.Join([]string{"passing-test", name}, " "), name)
		assert.NoError(err)

		result, message, err = alwaysFails(name)
		assert.False(result)
		assert.Equal("", message)
		assert.Error(err)
	}
}
