package config

import (
	"testing"

	"github.com/mongodb/amboy"
	"github.com/stretchr/testify/assert"
	"github.com/tychoish/grip"
)

func TestGetFormatFromFileName(t *testing.T) {
	assert := assert.New(t)

	// should return yaml
	for _, fn := range []string{"f.yaml", ".yaml", ".yml", "f.yml", ".json.yaml"} {
		frmt, err := getFormat(fn)
		assert.NoError(err)
		assert.Equal(amboy.YAML, frmt)
	}

	// should return json
	for _, fn := range []string{"f.json", ".json", ".yaml.abzckdfj_.json"} {
		frmt, err := getFormat(fn)
		assert.NoError(err)
		assert.Equal(amboy.JSON, frmt)
	}

	// should return error
	for _, fn := range []string{"json", "yaml", "f_json", "f_yaml", "foo.bson", "a.json-yaml", "b.yaml-json"} {
		frmt, err := getFormat(fn)
		assert.Error(err)
		assert.Equal(amboy.Format(-1), frmt)
	}

}

func TestGetJsonConfig(t *testing.T) {
	assert := assert.New(t)

	inputs := [][]byte{
		[]byte{},
		[]byte(`{foo: 1, bar: true}`),
		[]byte(`{}`),
	}

	// because all valid json is also valid yaml, we can sort of fake this test, at least in the easy case:

	for _, data := range inputs {
		out, err := getJSONFormatedConfig(amboy.JSON, data)
		assert.Equal(out, data)
		assert.NoError(err)

		_, err = getJSONFormatedConfig(amboy.YAML, data)
		if !assert.NoError(err) {
			grip.Error(err)
		}
	}

	// yaml config can't handle "[]" because it converts through a map:
	out, err := getJSONFormatedConfig(amboy.YAML, []byte(`[]`))
	assert.Error(err)
	assert.Nil(out)

	out, err = getJSONFormatedConfig(amboy.BSON, []byte{})
	assert.Error(err)
	assert.Nil(out)
}
